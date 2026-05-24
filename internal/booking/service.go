package booking

// Core Requirements: R-RES-CRUD-001, R-RES-CRUD-013, R-RES-CRUD-014, R-RES-CRUD-018, ADR-015, ADR-018

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/util"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// Service implements the reservation domain business logic.
// Handlers call Service methods; Service calls SQLC-generated store.Queries
// via ExecuteTx for transactional consistency.
type Service struct {
	pool *pgxpool.Pool
	q    *store.Queries
	rdb  *redis.Client
	log  *slog.Logger
}

// NewService creates a new booking Service.
func NewService(pool *pgxpool.Pool, q *store.Queries, rdb *redis.Client, log *slog.Logger) *Service {
	return &Service{
		pool: pool,
		q:    q,
		rdb:  rdb,
		log:  log,
	}
}

// --- CreateReservation ---

// CreateReservation creates a new reservation with items, ledger rows,
// booked daily rates, and a stub Folio A. The initial status depends on
// source and walkin flag (see SourceToInitialStatus map).
//
// Guest dedup by email is deferred to the guest-profile PR. Frontend will
// use pg_trgm autocomplete against identity.guests for live search.
//
// include controls response depth. When flags.Items is false, the response
// omits the items array. When flags.Guest is true and primary_guest_id is
// set, the guest object is expanded inline.
func (s *Service) CreateReservation(ctx context.Context, input *CreateReservationInput, include IncludeFlags) (*ReservationResponse, error) {
	initial, ok := SourceToInitialStatus[input.Source]
	if !ok {
		return nil, ErrSourceDeferred
	}

	// Walk-in override: internal + is_walkin=true → checked_in
	if SourceIsWalkin(input.Source, input.IsWalkin) {
		initial = struct {
			ReservationStatus ReservationStatus
			ItemStatus        ItemStatus
		}{StatusCheckedIn, ItemStatusCheckedIn}

		// If source is walk-in, every room must be assigned
		for i, item := range input.Items {
			if item.AssignedRoomID == nil {
				return nil, ErrUnassignedItems.WithMessage(fmt.Sprintf("item %d: walk-in requires assigned_room_id", i))
			}
		}
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		// Past-date check uses property timezone, not server UTC. Skip for walk-in
		// (R-RES-VALID-002); retroactive_create is gate-checked in handler.
		if !SourceIsWalkin(input.Source, input.IsWalkin) {
			propertyID := helpers.GetPropertyIDFromCtx(ctx)
			tzName, err := qtx.GetPropertyTimezone(ctx, propertyID)
			if err != nil {
				return nil, fmt.Errorf("load property timezone: %w", err)
			}
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				return nil, fmt.Errorf("invalid property timezone %q: %w", tzName, err)
			}
			now := time.Now().In(loc)
			midnightToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
			for _, item := range input.Items {
				if item.ArrivalDate.Before(midnightToday) {
					return nil, ErrInvalidDates.WithMessage("arrival date must be today or later")
				}
			}
		}

		result, callErr := s.createReservationInTx(ctx, qtx, input, initial, include)
		if callErr != nil {
			s.log.Error("createReservationInTx failed", "error", callErr, "source", input.Source)
		}
		return result, callErr
	})
}

// createReservationInTx is the transactional body of CreateReservation.
func (s *Service) createReservationInTx(
	ctx context.Context,
	qtx *store.Queries,
	input *CreateReservationInput,
	initial struct {
		ReservationStatus ReservationStatus
		ItemStatus        ItemStatus
	},
	include IncludeFlags,
) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	now := time.Now()

	// Compute stay_period_envelope across all items (ADR-020)
	envLower, envUpper := computeEnvelope(input.Items)

	expiresAt := computeExpiresAt(input.Source, initial.ReservationStatus, now)

	// --- 2. Resolve guest before creating reservation (FK requires valid primary_guest_id) ---
	primaryGuestID, err := resolvePrimaryGuest(ctx, qtx, input, propertyID)
	if err != nil {
		return nil, fmt.Errorf("resolve guest: %w", err)
	}

	// --- 3. Insert reservation ---
	res, err := qtx.CreateReservation(ctx, &store.CreateReservationParams{
		PropertyID:         propertyID,
		PrimaryGuestID:     primaryGuestID,
		GroupID:            uuid.NullUUID{UUID: util.PtrUUID(input.GroupID), Valid: input.GroupID != nil},
		Source:             store.OperationsReservationSource(input.Source),
		TravelAgentID:      uuid.NullUUID{UUID: util.PtrUUID(input.TravelAgentID), Valid: input.TravelAgentID != nil},
		Notes:              pgtype.Text{String: input.Notes, Valid: input.Notes != ""},
		Status:             store.OperationsReservationStatus(initial.ReservationStatus),
		StayPeriodEnvelope: util.ToRange(envLower, envUpper),
		ExpiresAt:          expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create reservation: %w", err)
	}

	// --- 4. Insert items + ledger rows + booked daily rates ---
	itemResponses, err := insertAllItems(ctx, qtx, input, initial, propertyID, res.ID)
	if err != nil {
		return nil, err
	}
	if _, err := qtx.CreateFolio(ctx, &store.CreateFolioParams{
		PropertyID:    propertyID,
		ReservationID: uuid.NullUUID{UUID: res.ID, Valid: true},
		FolioPart:     store.FinanceFolioPartA,
	}); err != nil {
		return nil, fmt.Errorf("create folio a: %w", err)
	}

	if err := notifyReservationChange(ctx, qtx, "created", res.ID); err != nil {
		return nil, fmt.Errorf("notify reservation_changes: %w", err)
	}

	response := reservationToResponse(&res)

	if include.IncludeItems() {
		response.Items = itemResponses
	}
	if include.Guest && primaryGuestID != uuid.Nil {
		guest, err := qtx.GetGuest(ctx, primaryGuestID)
		if err != nil {
			s.log.Warn("failed to fetch guest for expansion", "error", err, "guest_id", primaryGuestID)
		} else {
			response.Guest = guestToResponse(&guest)
		}
	}
	return response, nil
}

// ConfirmReservation transitions a hold reservation to confirmed.
// Requires the reservation to be in 'hold' status and have a guest attached.
// If-Match version is extracted from context (set by RequireIfMatch middleware).
//
// include controls response depth (items, guest expansion).
func (s *Service) ConfirmReservation(ctx context.Context, id uuid.UUID, include IncludeFlags) (*ReservationResponse, error) {
	version := helpers.GetIfMatchVersion(ctx)

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		resRow, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get reservation: %w", err)
		}

		// State machine check: only hold → confirmed is allowed
		if err := ValidateReservationTransition(
			ReservationStatus(resRow.Status),
			StatusConfirmed,
		); err != nil {
			return nil, err
		}

		// Guest must be attached (R-RES-CRUD-018)
		if resRow.PrimaryGuestID == uuid.Nil {
			return nil, ErrGuestNotAttached
		}

		rowsAffected, err := qtx.UpdateReservationStatus(ctx, &store.UpdateReservationStatusParams{
			ID:      id,
			Version: version,
			Status:  store.OperationsReservationStatusConfirmed,
		})
		if err != nil {
			return nil, fmt.Errorf("confirm reservation: %w", err)
		}
		if rowsAffected == 0 {
			return nil, ErrVersionMismatch
		}

		// Re-fetch to get updated row
		updatedRow, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("re-fetch reservation: %w", err)
		}

		response := reservationFromRow(&updatedRow)

		if include.IncludeItems() {
			items, err := qtx.GetReservationItems(ctx, &store.GetReservationItemsParams{
				ReservationID: id,
				PropertyID:    resRow.PropertyID,
			})
			if err != nil {
				return nil, fmt.Errorf("get items: %w", err)
			}
			for _, item := range items {
				it := itemToResponse(&item)
				response.Items = append(response.Items, *it)
			}
		}

		if include.Guest && resRow.PrimaryGuestID != uuid.Nil {
			guest, err := qtx.GetGuest(ctx, resRow.PrimaryGuestID)
			if err != nil {
				s.log.Warn("failed to fetch guest for expansion", "error", err, "guest_id", resRow.PrimaryGuestID)
			} else {
				response.Guest = guestToResponse(&guest)
			}
		}

		if err := notifyReservationChange(ctx, qtx, "confirmed", id); err != nil {
			return nil, fmt.Errorf("notify reservation_changes: %w", err)
		}

		return response, nil
	})
}

// --- Helpers ---

// reservationToResponse converts store.OperationsReservation to ReservationResponse.
func reservationToResponse(r *store.OperationsReservation) *ReservationResponse {
	return &ReservationResponse{
		ID:                 r.ID,
		PropertyID:         r.PropertyID,
		PrimaryGuestID:     util.PtrToUUID(r.PrimaryGuestID),
		GroupID:            util.NullUUIDToPtr(r.GroupID),
		Source:             ReservationSource(r.Source),
		TravelAgentID:      util.NullUUIDToPtr(r.TravelAgentID),
		Notes:              util.NullText(r.Notes),
		Status:             ReservationStatus(r.Status),
		Version:            r.Version,
		CreatedAt:          util.TSToTime(r.CreatedAt),
		UpdatedAt:          util.TSToTime(r.UpdatedAt),
		DeletedAt:          util.TSToTimePtr(r.DeletedAt),
		Sequential:         r.Sequential,
		Code:               r.Code,
		StayPeriodEnvelope: util.FormatRange(r.StayPeriodEnvelope),
		ExpiresAt:          util.TSToTimePtr(r.ExpiresAt),
	}
}

// reservationFromRow converts GetReservationRow (joined with items JSON) to ReservationResponse.
func reservationFromRow(r *store.GetReservationRow) *ReservationResponse {
	return &ReservationResponse{
		ID:                 r.ID,
		PropertyID:         r.PropertyID,
		PrimaryGuestID:     util.PtrToUUID(r.PrimaryGuestID),
		GroupID:            util.NullUUIDToPtr(r.GroupID),
		Source:             ReservationSource(r.Source),
		TravelAgentID:      util.NullUUIDToPtr(r.TravelAgentID),
		Notes:              util.NullText(r.Notes),
		Status:             ReservationStatus(r.Status),
		Version:            r.Version,
		CreatedAt:          util.TSToTime(r.CreatedAt),
		UpdatedAt:          util.TSToTime(r.UpdatedAt),
		DeletedAt:          util.TSToTimePtr(r.DeletedAt),
		Sequential:         r.Sequential,
		Code:               r.Code,
		StayPeriodEnvelope: util.FormatRange(r.StayPeriodEnvelope),
		ExpiresAt:          util.TSToTimePtr(r.ExpiresAt),
	}
}

// itemToResponse converts store.OperationsReservationItem to ItemResponse.
func itemToResponse(i *store.OperationsReservationItem) *ItemResponse {
	return &ItemResponse{
		ID:               i.ID,
		PropertyID:       i.PropertyID,
		ReservationID:    i.ReservationID,
		BookedRoomTypeID: i.BookedRoomTypeID,
		AssignedRoomID:   util.NullUUIDToPtr(i.AssignedRoomID),
		GuestID:          util.NullUUIDToPtr(i.GuestID),
		RatePlanID:       util.NullUUIDToPtr(i.RatePlanID),
		StayPeriod:       util.FormatRange(i.StayPeriod),
		BaseRatePence:    i.BaseRatePence,
		AdultsCount:      i.AdultsCount,
		ChildrenCount:    i.ChildrenCount,
		Status:           ItemStatus(i.Status),
		Version:          i.Version,
		DoNotMove:        i.DoNotMove,
		CreatedAt:        util.TSToTime(i.CreatedAt),
		UpdatedAt:        util.TSToTime(i.UpdatedAt),
		DeletedAt:        util.TSToTimePtr(i.DeletedAt),
	}
}

// guestToResponse converts store.IdentityGuest to GuestResponse.
func guestToResponse(g *store.IdentityGuest) *GuestResponse {
	return &GuestResponse{
		ID:         g.ID,
		PropertyID: g.PropertyID,
		FirstName:  g.FirstName,
		LastName:   g.LastName,
		Email:      util.NullText(g.Email),
		Phone:      util.NullText(g.PhoneNumber),
	}
}

// computeEnvelope finds the min arrival and max departure across all items (ADR-020).
func computeEnvelope(items []CreateItemInput) (time.Time, time.Time) {
	var envelopeLower, envelopeUpper time.Time
	for i, item := range items {
		arrival := item.ArrivalDate.Time
		departure := item.DepartureDate.Time
		if i == 0 || arrival.Before(envelopeLower) {
			envelopeLower = arrival
		}
		if i == 0 || departure.After(envelopeUpper) {
			envelopeUpper = departure
		}
	}
	return envelopeLower, envelopeUpper
}

// computeExpiresAt determines the hold expiration timestamp based on source.
func computeExpiresAt(source ReservationSource, status ReservationStatus, now time.Time) pgtype.Timestamptz {
	if status != StatusHold {
		return pgtype.Timestamptz{}
	}
	var ttlSeconds int32 = 1800 // 30 min default
	if source == SourceWebsite {
		ttlSeconds = 900 // 15 min
	}
	return pgtype.Timestamptz{Time: now.Add(time.Duration(ttlSeconds) * time.Second), Valid: true}
}

// resolvePrimaryGuest either uses the provided guest ID or creates a new guest from inline data.
func resolvePrimaryGuest(ctx context.Context, qtx *store.Queries, input *CreateReservationInput, propertyID uuid.UUID) (uuid.UUID, error) {
	if input.PrimaryGuestID != nil {
		return *input.PrimaryGuestID, nil
	}
	if input.Guest != nil {
		createdGuest, err := qtx.CreateGuest(ctx, &store.CreateGuestParams{
			PropertyID:  propertyID,
			FirstName:   input.Guest.FirstName,
			LastName:    input.Guest.LastName,
			Email:       pgtype.Text{String: input.Guest.Email, Valid: input.Guest.Email != ""},
			PhoneNumber: pgtype.Text{String: input.Guest.Phone, Valid: input.Guest.Phone != ""},
		})
		if err != nil {
			return uuid.Nil, err
		}
		return createdGuest.ID, nil
	}
	return uuid.Nil, nil
}

// insertAllItems creates reservation items, auto-pins rooms, inserts ledger rows and booked daily rates.
func insertAllItems(
	ctx context.Context,
	qtx *store.Queries,
	input *CreateReservationInput,
	initial struct {
		ReservationStatus ReservationStatus
		ItemStatus        ItemStatus
	},
	propertyID uuid.UUID,
	reservationID uuid.UUID,
) ([]ItemResponse, error) {
	itemResponses := make([]ItemResponse, 0, len(input.Items))
	for _, item := range input.Items {
		ir, err := insertSingleItem(ctx, qtx, item, initial, propertyID, reservationID)
		if err != nil {
			return nil, err
		}
		itemResponses = append(itemResponses, ir)
	}
	return itemResponses, nil
}

// insertSingleItem creates one reservation item, pins room, inserts ledger + rates.
func insertSingleItem(
	ctx context.Context,
	qtx *store.Queries,
	item CreateItemInput,
	initial struct {
		ReservationStatus ReservationStatus
		ItemStatus        ItemStatus
	},
	propertyID uuid.UUID,
	reservationID uuid.UUID,
) (ItemResponse, error) {
	arrival := item.ArrivalDate.Time
	departure := item.DepartureDate.Time
	baseRatePence := int32(10000) // Placeholder until rate lookup (Phase 4)

	ri, err := qtx.CreateReservationItem(ctx, &store.CreateReservationItemParams{
		PropertyID:       propertyID,
		ReservationID:    reservationID,
		BookedRoomTypeID: item.RoomTypeID,
		AssignedRoomID:   uuid.NullUUID{UUID: util.PtrUUID(item.AssignedRoomID), Valid: item.AssignedRoomID != nil},
		GuestID:          uuid.NullUUID{UUID: uuid.Nil, Valid: false},
		RatePlanID:       uuid.NullUUID{UUID: util.PtrUUID(item.RatePlanID), Valid: item.RatePlanID != nil},
		StayPeriod:       util.ToRange(arrival, departure),
		BaseRatePence:    baseRatePence,
		AdultsCount:      int32(item.AdultsCount),
		ChildrenCount:    int32(item.ChildrenCount),
		Status:           store.OperationsReservationItemStatus(initial.ItemStatus),
		DoNotMove:        false,
	})
	if err != nil {
		return ItemResponse{}, fmt.Errorf("create item: %w", err)
	}

	// Auto-pin room or use assigned
	var roomID uuid.UUID
	if item.AssignedRoomID != nil {
		roomID = *item.AssignedRoomID
	} else {
		dates := util.NightsBetween(arrival, departure)
		pgDates := make([]pgtype.Date, len(dates))
		for j, d := range dates {
			pgDates[j] = pgtype.Date{Time: d, Valid: true}
		}
		pinned, err := qtx.SelectRoomForAutoPin(ctx, &store.SelectRoomForAutoPinParams{
			PropertyID: propertyID,
			RoomTypeID: uuid.NullUUID{UUID: item.RoomTypeID, Valid: true},
			Dates:      pgDates,
		})
		if err != nil {
			return ItemResponse{}, fmt.Errorf("auto-pin room: %w", err)
		}
		roomID = pinned
	}

	// Bulk insert ledger rows
	dates := util.NightsBetween(arrival, departure)
	n := len(dates)
	roomIDs := make([]uuid.UUID, n)
	resIDs := make([]uuid.UUID, n)
	itemIDs := make([]uuid.UUID, n)
	propIDs := make([]uuid.UUID, n)
	statuses := make([]string, n)
	ledgerDates := make([]pgtype.Date, n)

	for j, d := range dates {
		roomIDs[j] = roomID
		resIDs[j] = reservationID
		itemIDs[j] = ri.ID
		propIDs[j] = propertyID
		statuses[j] = "sold"
		ledgerDates[j] = pgtype.Date{Time: d, Valid: true}
	}

	if err := qtx.BulkInsertLedgerRows(ctx, &store.BulkInsertLedgerRowsParams{
		PropertyIds:        propIDs,
		RoomIds:            roomIDs,
		ReservationIds:     resIDs,
		ReservationItemIds: itemIDs,
		CalendarDates:      ledgerDates,
		Statuses:           statuses,
	}); err != nil {
		return ItemResponse{}, fmt.Errorf("insert ledger rows: %w", err)
	}

	// Bulk insert booked daily rates — only when rate plan is set. FK requires a valid uuid.
	if item.RatePlanID != nil {
		rateDates := make([]pgtype.Date, n)
		ratePropIDs := make([]uuid.UUID, n)
		rateItemIDs := make([]uuid.UUID, n)
		ratePlanIDs := make([]uuid.UUID, n)
		basePrices := make([]int32, n)
		for j, d := range dates {
			rateDates[j] = pgtype.Date{Time: d, Valid: true}
			ratePropIDs[j] = propertyID
			rateItemIDs[j] = ri.ID
			ratePlanIDs[j] = *item.RatePlanID
			basePrices[j] = baseRatePence
		}
		if err := qtx.BulkInsertBookedDailyRates(ctx, &store.BulkInsertBookedDailyRatesParams{
			PropertyIds:        ratePropIDs,
			ReservationItemIds: rateItemIDs,
			CalendarDates:      rateDates,
			RatePlanIds:        ratePlanIDs,
			BasePricePences:    basePrices,
		}); err != nil {
			return ItemResponse{}, fmt.Errorf("insert booked daily rates: %w", err)
		}
	}

	return *itemToResponse(&ri), nil
}

// notifyReservationChange emits a NOTIFY on reservation_changes channel.
// Caller must fail the tx on error so the event cannot be lost.
func notifyReservationChange(ctx context.Context, qtx *store.Queries, action string, reservationID uuid.UUID) error {
	payload, err := json.Marshal(struct {
		Action        string    `json:"action"`
		ReservationID uuid.UUID `json:"reservation_id"`
	}{Action: action, ReservationID: reservationID})
	if err != nil {
		return fmt.Errorf("marshal notify payload: %w", err)
	}
	return qtx.NotifyChannel(ctx, &store.NotifyChannelParams{
		Channel: "reservation_changes",
		Payload: string(payload),
	})
}
