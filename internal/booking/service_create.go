package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/util"
	"github.com/lexxcode1/yop-pms/internal/platform/worker"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// service_create.go — Reservation creation service:
//   - CreateReservation, ConfirmReservation
//   - Helpers: createReservationInTx, resolvePrimaryGuest, computeEnvelope, computeExpiresAt,
//     insertAllItems, insertSingleItem, notifyReservationChange

// CreateReservation creates a new reservation with items, ledger rows,
// The initial status depends on source and walkin flag (see SourceToInitialStatus map).
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

	res, txErr := db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
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
	if txErr != nil {
		return nil, txErr
	}

	// Invalidate availability cache after successful creation
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	s.invalidateAvailabilityAfterCreate(ctx, propertyID, input)

	return res, nil
}

func (s *Service) invalidateAvailabilityAfterCreate(ctx context.Context, propertyID uuid.UUID, input *CreateReservationInput) {
	seen := make(map[uuid.UUID]bool)
	for _, item := range input.Items {
		if seen[item.RoomTypeID] {
			continue
		}
		seen[item.RoomTypeID] = true
		s.InvalidateAvailabilityCache(ctx, propertyID, item.RoomTypeID, item.ArrivalDate.Time, item.DepartureDate.Time)
	}
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

	envLower, envUpper := computeEnvelope(input.Items)
	expiresAt := computeExpiresAt(ctx, qtx, propertyID, input.Source, initial.ReservationStatus, now)

	primaryGuestID, err := resolvePrimaryGuest(ctx, qtx, input, propertyID)
	if err != nil {
		return nil, fmt.Errorf("resolve guest: %w", err)
	}

	res, err := qtx.CreateReservation(ctx, &store.CreateReservationParams{
		PropertyID:         propertyID,
		PrimaryGuestID:     primaryGuestID.UUID,
		Source:             store.OperationsReservationSource(input.Source),
		Notes:              pgtype.Text{String: input.Notes, Valid: input.Notes != ""},
		Status:             store.OperationsReservationStatus(initial.ReservationStatus),
		StayPeriodEnvelope: util.ToRange(envLower, envUpper),
		ExpiresAt:          expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create reservation: %w", err)
	}

	itemResponses, err := insertAllItems(ctx, qtx, input, initial, propertyID, res.ID)
	if err != nil {
		return nil, err
	}

	if err := notifyReservationChange(ctx, qtx, "created", res.ID); err != nil {
		return nil, fmt.Errorf("notify reservation_changes: %w", err)
	}

	if initial.ReservationStatus != StatusHold {
		if err := worker.Enqueue(ctx, qtx, worker.EventConfirmationEmail, worker.ConfirmationEmailPayload{
			ReservationID: res.ID.String(),
		}); err != nil {
			s.log.Warn("enqueue confirmation email", "error", err, "reservation_id", res.ID)
		}
	}

	response := reservationToResponse(&res)

	if include.IncludeItems() {
		response.Items = itemResponses
	}

	expandInclude(ctx, qtx, response, include, propertyID, res.ID, primaryGuestID, s.log)
	return response, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Service: ConfirmReservation
// ─────────────────────────────────────────────────────────────────────────────

// ConfirmReservation transitions a hold reservation to confirmed.
// Requires the reservation to be in 'hold' status and have a guest attached.
// If-Match version is extracted from context (set by RequireIfMatch middleware).
func (s *Service) ConfirmReservation(ctx context.Context, id uuid.UUID, include IncludeFlags) (*ReservationResponse, error) {
	version := helpers.GetIfMatchVersion(ctx)

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		resRow, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get reservation: %w", err)
		}

		if ReservationStatus(resRow.Status) == StatusConfirmed {
			response := reservationFromRow(&resRow)
			expandInclude(ctx, qtx, response, include, resRow.PropertyID, id, uuid.NullUUID{UUID: resRow.PrimaryGuestID, Valid: true}, s.log)
			return response, nil
		}
		if err := ValidateReservationTransition(
			ReservationStatus(resRow.Status),
			StatusConfirmed,
		); err != nil {
			return nil, err
		}

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

		updatedRow, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("re-fetch reservation: %w", err)
		}

		response := reservationFromRow(&updatedRow)
		expandInclude(ctx, qtx, response, include, resRow.PropertyID, id, uuid.NullUUID{UUID: resRow.PrimaryGuestID, Valid: true}, s.log)

		if err := notifyReservationChange(ctx, qtx, "confirmed", id); err != nil {
			return nil, fmt.Errorf("notify reservation_changes: %w", err)
		}

		if err := worker.Enqueue(ctx, qtx, worker.EventConfirmationEmail, worker.ConfirmationEmailPayload{
			ReservationID: id.String(),
		}); err != nil {
			s.log.Warn("enqueue confirmation email", "error", err, "reservation_id", id)
		}

		return response, nil
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

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

// computeExpiresAt determines the hold expiration timestamp based on source and property settings.
func computeExpiresAt(ctx context.Context, qtx *store.Queries, propertyID uuid.UUID, source ReservationSource, status ReservationStatus, now time.Time) pgtype.Timestamptz {
	if status != StatusHold {
		return pgtype.Timestamptz{}
	}

	defaultTTL := int32(1800) // 30 min default
	if source == SourceWebsite {
		defaultTTL = 900 // 15 min
	}

	ttlSeconds := defaultTTL

	return pgtype.Timestamptz{Time: now.Add(time.Duration(ttlSeconds) * time.Second), Valid: true}
}

// resolvePrimaryGuest either uses the provided guest ID or creates a new guest from inline data.
func resolvePrimaryGuest(ctx context.Context, qtx *store.Queries, input *CreateReservationInput, propertyID uuid.UUID) (uuid.NullUUID, error) {
	if input.PrimaryGuestID != nil {
		return uuid.NullUUID{UUID: *input.PrimaryGuestID, Valid: true}, nil
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
			return uuid.NullUUID{}, err
		}
		return uuid.NullUUID{UUID: createdGuest.ID, Valid: true}, nil
	}
	return uuid.NullUUID{}, nil
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

	totalGuests := int32(item.AdultsCount + item.ChildrenCount)
	limits, err := qtx.GetRoomTypeOccupancy(ctx, &store.GetRoomTypeOccupancyParams{
		ID: item.RoomTypeID, PropertyID: propertyID,
	})
	if err != nil {
		return ItemResponse{}, fmt.Errorf("get room type occupancy: %w", err)
	}
	if totalGuests < limits.MinOccupancy || (limits.MaxOccupancy > 0 && totalGuests > limits.MaxOccupancy) {
		return ItemResponse{}, fmt.Errorf(
			"occupancy %d is outside room type limits [%d, %d]", totalGuests, limits.MinOccupancy, limits.MaxOccupancy,
		)
	}

	dates := util.NightsBetween(arrival, departure)
	n := len(dates)

	var baseRatePences []int32
	switch {
	case n == 0:
		baseRatePences = []int32{10000}
		n = 1
	case item.RatePlanID != nil:
		baseRatePences = make([]int32, n)
		for j, d := range dates {
			price, err := qtx.GetBaseRate(ctx, &store.GetBaseRateParams{
				PropertyID: propertyID,
				RoomTypeID: item.RoomTypeID,
				RatePlanID: *item.RatePlanID,
				DayOfWeek:  int32(d.Weekday()),
			})
			if err != nil {
				return ItemResponse{}, fmt.Errorf("resolve rate for %s: %w", d.Format("2006-01-02"), err)
			}
			baseRatePences[j] = price
		}
	default:
		baseRatePences = make([]int32, n)
		for j := range baseRatePences {
			baseRatePences[j] = 10000
		}
	}

	ri, err := qtx.CreateReservationItem(ctx, &store.CreateReservationItemParams{
		PropertyID:       propertyID,
		ReservationID:    reservationID,
		BookedRoomTypeID: item.RoomTypeID,
		AssignedRoomID:   uuid.NullUUID{UUID: util.PtrUUID(item.AssignedRoomID), Valid: item.AssignedRoomID != nil},
		GuestID:          uuid.NullUUID{UUID: uuid.Nil, Valid: false},
		RatePlanID:       uuid.NullUUID{UUID: util.PtrUUID(item.RatePlanID), Valid: item.RatePlanID != nil},
		StayPeriod:       util.ToRange(arrival, departure),
		BaseRatePence:    baseRatePences[0],
		AdultsCount:      int32(item.AdultsCount),
		ChildrenCount:    int32(item.ChildrenCount),
		Status:           store.OperationsReservationItemStatus(initial.ItemStatus),
		DoNotMove:        false,
	})
	if err != nil {
		return ItemResponse{}, fmt.Errorf("create item: %w", err)
	}

	var roomID uuid.UUID
	if item.AssignedRoomID != nil {
		roomID = *item.AssignedRoomID
	} else {
		pgDates := make([]pgtype.Date, n)
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
			basePrices[j] = baseRatePences[j]
		}
	}

	return *itemToResponse(&ri), nil
}

// notifyReservationChange emits a NOTIFY on reservation_changes channel.
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
