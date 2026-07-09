package booking

// Converters and shared utility functions used across create.go, lifecycle.go, and update.go.

import (
	"context"
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
		Source:             ReservationSource(r.Source),
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
		Source:             ReservationSource(r.Source),
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

// insertItemLedger bulk-inserts ledger rows for specific nights.
// Used by UpdateItem when a stay is lengthened.
// TODO Add booked daily rates
func insertItemLedger(
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

	var baseRatePences []int32
	if ratePlanID.Valid {
		baseRatePences = make([]int32, n)
		for j, d := range dates {
			price, err := qtx.GetBaseRate(ctx, &store.GetBaseRateParams{
				PropertyID: propertyID,
				RoomTypeID: roomTypeID,
				RatePlanID: ratePlanID.UUID,
				DayOfWeek:  int32(d.Weekday()),
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

	return nil
}

// reactivateItemInventory resolves a room for a reactivated item and re-inserts
// ledger rows + booked daily rates.
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

	var roomID uuid.UUID
	if item.AssignedRoomID.Valid {
		pgDates := make([]pgtype.Date, len(dates))
		for j, d := range dates {
			pgDates[j] = pgtype.Date{Time: d, Valid: true}
		}
		conflicts, err := qtx.ConflictCheckOnLedger(ctx, &store.ConflictCheckOnLedgerParams{
			RoomID: item.AssignedRoomID.UUID, Dates: pgDates, PropertyID: propertyID,
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

	return insertItemLedger(ctx, qtx, item.ID, item.ReservationID,
		propertyID, dates, roomID, item.RatePlanID, item.BookedRoomTypeID)
}

// recomputeEnvelope recalculates stay_period_envelope from all items.
func recomputeEnvelope(ctx context.Context, qtx *store.Queries, log *slog.Logger, reservationID, propertyID uuid.UUID) error {
	items, err := qtx.GetReservationItems(ctx, &store.GetReservationItemsParams{
		ReservationID: reservationID, PropertyID: propertyID,
	})
	if err != nil {
		return fmt.Errorf("get items for envelope: %w", err)
	}

	var lower, upper time.Time
	for i, item := range items {
		if i == 0 || item.StayPeriod.Lower.Time.Before(lower) {
			lower = item.StayPeriod.Lower.Time
		}
		if i == 0 || item.StayPeriod.Upper.Time.After(upper) {
			upper = item.StayPeriod.Upper.Time
		}
	}

	if lower.IsZero() || upper.IsZero() {
		log.Debug("recomputeEnvelope: no valid items, skipping envelope update", "reservation_id", reservationID)
		return nil
	}

	res, err := qtx.GetReservation(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("get reservation for envelope: %w", err)
	}

	_, err = qtx.UpdateReservationMetadata(ctx, &store.UpdateReservationMetadataParams{
		ID:                 reservationID,
		Version:            res.Version,
		StayPeriodEnvelope: util.ToRange(lower, upper),
		PrimaryGuestID:     res.PrimaryGuestID,
	})
	return err
}
