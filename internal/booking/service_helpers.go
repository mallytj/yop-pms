package booking

// Converter helpers and shared utility functions for the booking domain.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/platform/util"
	"github.com/lexxcode1/yop-pms/internal/store"
)

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

// computeExpiresAt determines the hold expiration timestamp based on source and
// property settings. Falls back to sensible defaults (30 min / 15 min) if settings
// are unavailable.
func computeExpiresAt(ctx context.Context, qtx *store.Queries, propertyID uuid.UUID, source ReservationSource, status ReservationStatus, now time.Time) pgtype.Timestamptz {
	if status != StatusHold {
		return pgtype.Timestamptz{}
	}

	// Try to load per-property TTL from settings; fall back to hardcoded defaults.
	defaultTTL := int32(1800) // 30 min default
	if source == SourceWebsite {
		defaultTTL = 900 // 15 min
	}

	ttlSeconds := defaultTTL
	settings, err := qtx.GetPropertySettings(ctx, propertyID)
	if err == nil {
		if source == SourceWebsite && settings.WebsiteHoldTtlSeconds > 0 {
			ttlSeconds = settings.WebsiteHoldTtlSeconds
		} else if source == SourceInternal && settings.InternalHoldTtlSeconds > 0 {
			ttlSeconds = settings.InternalHoldTtlSeconds
		}
	}

	return pgtype.Timestamptz{Time: now.Add(time.Duration(ttlSeconds) * time.Second), Valid: true}
}

// resolvePrimaryGuest either uses the provided guest ID or creates a new guest from inline data.
// Returns uuid.NullUUID so that hold reservations can be created without a guest (nullable FK).
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

	// Validate occupancy against room type min/max (R-RES-VALID-010).
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

	// Pre-compute nights list once
	dates := util.NightsBetween(arrival, departure)
	n := len(dates)

	// Rate lookup per night: daily_price_grid > seasonal_rates > base_rates > default
	var baseRatePences []int32
	switch {
	case n == 0:
		baseRatePences = []int32{10000}
		n = 1
	case item.RatePlanID != nil:
		baseRatePences = make([]int32, n)
		for j, d := range dates {
			price, err := qtx.GetResolvedNightlyRate(ctx, &store.GetResolvedNightlyRateParams{
				PropertyID:   propertyID,
				RoomTypeID:   item.RoomTypeID,
				RatePlanID:   *item.RatePlanID,
				CalendarDate: pgtype.Date{Time: d, Valid: true},
				DayOfWeek:    int32(d.Weekday()),
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

	// Auto-pin room or use assigned
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

	// Bulk insert ledger rows
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
			basePrices[j] = baseRatePences[j]
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

// insertItemLedgerAndRates bulk-inserts ledger rows and booked daily rates
// for specific nights. Used by UpdateItem when a stay is lengthened.
func insertItemLedgerAndRates(
	ctx context.Context,
	qtx *store.Queries,
	itemID uuid.UUID,
	reservationID uuid.UUID,
	propertyID uuid.UUID,
	dates []time.Time,
	roomID uuid.UUID,
	ratePlanID uuid.NullUUID,
	roomTypeID uuid.UUID,
) error {
	n := len(dates)
	if n == 0 {
		return nil
	}

	// Resolve nightly rates only if rate plan is set
	var baseRatePences []int32
	if ratePlanID.Valid {
		baseRatePences = make([]int32, n)
		for j, d := range dates {
			price, err := qtx.GetResolvedNightlyRate(ctx, &store.GetResolvedNightlyRateParams{
				PropertyID:   propertyID,
				RoomTypeID:   roomTypeID,
				RatePlanID:   ratePlanID.UUID,
				CalendarDate: pgtype.Date{Time: d, Valid: true},
				DayOfWeek:    int32(d.Weekday()),
			})
			if err != nil {
				return fmt.Errorf("resolve rate for %s: %w", d.Format("2006-01-02"), err)
			}
			baseRatePences[j] = price
		}
	} else {
		baseRatePences = make([]int32, n)
		for j := range baseRatePences {
			baseRatePences[j] = 10000
		}
	}

	// Bulk insert ledger rows
	roomIDs := make([]uuid.UUID, n)
	resIDs := make([]uuid.UUID, n)
	itemIDs := make([]uuid.UUID, n)
	propIDs := make([]uuid.UUID, n)
	statuses := make([]string, n)
	ledgerDates := make([]pgtype.Date, n)

	for j, d := range dates {
		roomIDs[j] = roomID
		resIDs[j] = reservationID
		itemIDs[j] = itemID
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
		return fmt.Errorf("insert added ledger: %w", err)
	}

	// Bulk insert booked daily rates
	if ratePlanID.Valid {
		rateDates := make([]pgtype.Date, n)
		ratePropIDs := make([]uuid.UUID, n)
		rateItemIDs := make([]uuid.UUID, n)
		ratePlanIDs := make([]uuid.UUID, n)
		basePrices := make([]int32, n)
		for j, d := range dates {
			rateDates[j] = pgtype.Date{Time: d, Valid: true}
			ratePropIDs[j] = propertyID
			rateItemIDs[j] = itemID
			ratePlanIDs[j] = ratePlanID.UUID
			basePrices[j] = baseRatePences[j]
		}
		if err := qtx.BulkInsertBookedDailyRates(ctx, &store.BulkInsertBookedDailyRatesParams{
			PropertyIds:        ratePropIDs,
			ReservationItemIds: rateItemIDs,
			CalendarDates:      rateDates,
			RatePlanIds:        ratePlanIDs,
			BasePricePences:    basePrices,
		}); err != nil {
			return fmt.Errorf("insert added rates: %w", err)
		}
	}

	return nil
}

// reactivateItemInventory resolves a room for a reactivated item and re-inserts
// ledger rows + booked daily rates. Returns an error if no room is available.
// Used by ReactivateReservation (R-RES-CRUD-006).
func reactivateItemInventory(
	ctx context.Context,
	qtx *store.Queries,
	item store.OperationsReservationItem,
	log *slog.Logger,
) error {
	propertyID := item.PropertyID
	dates := util.NightsBetween(item.StayPeriod.Lower.Time, item.StayPeriod.Upper.Time)
	if len(dates) == 0 {
		return nil
	}

	// Resolve room: use assigned if set, otherwise auto-pin.
	var roomID uuid.UUID
	if item.AssignedRoomID.Valid {
		// Verify the assigned room is still available.
		pgDates := make([]pgtype.Date, len(dates))
		for j, d := range dates {
			pgDates[j] = pgtype.Date{Time: d, Valid: true}
		}
		conflicts, err := qtx.ConflictCheckOnLedger(ctx, &store.ConflictCheckOnLedgerParams{
			RoomID:     item.AssignedRoomID.UUID,
			Dates:      pgDates,
			PropertyID: propertyID,
		})
		if err != nil {
			return fmt.Errorf("conflict check on assigned room: %w", err)
		}
		if len(conflicts) > 0 {
			return ErrRoomNotAvailable.WithMessage(
				fmt.Sprintf("assigned room not available for reactivation on %d date(s)", len(conflicts)),
			)
		}
		roomID = item.AssignedRoomID.UUID
	} else {
		pgDates := make([]pgtype.Date, len(dates))
		for j, d := range dates {
			pgDates[j] = pgtype.Date{Time: d, Valid: true}
		}
		pinned, err := qtx.SelectRoomForAutoPin(ctx, &store.SelectRoomForAutoPinParams{
			PropertyID: propertyID,
			RoomTypeID: uuid.NullUUID{UUID: item.BookedRoomTypeID, Valid: true},
			Dates:      pgDates,
		})
		if err != nil {
			return fmt.Errorf("auto-pin room for reactivation: %w", err)
		}
		roomID = pinned
	}

	// Soft-delete any existing booked_daily_rates for this item so the
	// fresh insert doesn't conflict with the partial UNIQUE index (M10).
	// Cancel preserves rates as a financial snapshot but leaves deleted_at=NULL;
	// reactivation issues a fresh set, so the old rows must be marked deleted first.
	if err := qtx.SoftDeleteBookedRatesNotInPeriod(ctx, &store.SoftDeleteBookedRatesNotInPeriodParams{
		ReservationItemID: item.ID,
		PropertyID:        propertyID,
		Dates:             []pgtype.Date{}, // empty = soft-delete all
	}); err != nil {
		return fmt.Errorf("soft-delete old rates before reactivation: %w", err)
	}

	return insertItemLedgerAndRates(ctx, qtx, item.ID, item.ReservationID,
		propertyID, dates, roomID, item.RatePlanID, item.BookedRoomTypeID)
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
