package booking

// Core Requirements: R-RES-CRUD-003, R-RES-CRUD-010, R-RES-CRUD-011, R-RES-CRUD-012

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/util"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// UpdateItem updates reservation item fields and returns the reservation response.
// When the stay period or room assignment changes, this incrementally updates
// inventory ledger rows and booked daily rates — only the rows that actually
// changed, not a full delete+reinsert.
func (s *Service) UpdateItem(ctx context.Context, itemID uuid.UUID, input CreateItemInput) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		version := helpers.GetIfMatchVersion(ctx)

		// Detect room type change.
		// TODO(bookings): Room type / rate plan changes return early, dropping
		// other PATCH fields (dates, room assignment). Inline the updates
		// within this transaction instead of delegating to separate txs.
		// Ignore zero UUID — PATCH bodies often omit room_type_id.
		if input.RoomTypeID != uuid.Nil && input.RoomTypeID != item.BookedRoomTypeID {
			if _, err := s.UpdateItemRoomType(ctx, itemID, input.RoomTypeID, false /* retainPrice */, ""); err != nil {
				return nil, err
			}
			return s.fetchAndExpandReservation(ctx, qtx, item.ReservationID, propertyID)
		}

		// Detect rate plan change (includes capacity check in UpdateItemRatePlan).
		if input.RatePlanID != nil && (!item.RatePlanID.Valid || item.RatePlanID.UUID != *input.RatePlanID) {
			if _, err := s.UpdateItemRatePlan(ctx, itemID, *input.RatePlanID, false /* retainPrice */); err != nil {
				return nil, err
			}
			return s.fetchAndExpandReservation(ctx, qtx, item.ReservationID, propertyID)
		}

		arrival := input.ArrivalDate.Time
		departure := input.DepartureDate.Time
		if !departure.After(arrival) {
			return nil, ErrInvalidDates.WithMessage("departure must be after arrival")
		}

		oldNights := util.NightsBetween(item.StayPeriod.Lower.Time, item.StayPeriod.Upper.Time)
		newNights := util.NightsBetween(arrival, departure)
		datesChanged := !arrival.Equal(item.StayPeriod.Lower.Time) || !departure.Equal(item.StayPeriod.Upper.Time)
		roomChanged := input.AssignedRoomID != nil && (item.AssignedRoomID.UUID != *input.AssignedRoomID || !item.AssignedRoomID.Valid)

		newStayPeriod := util.ToRange(arrival, departure)

		ratePlanID := item.RatePlanID
		if input.RatePlanID != nil {
			ratePlanID = uuid.NullUUID{UUID: *input.RatePlanID, Valid: true}
		}
		newRoomID := item.AssignedRoomID
		if input.AssignedRoomID != nil {
			newRoomID = uuid.NullUUID{UUID: *input.AssignedRoomID, Valid: true}
		}

		// Determine effective room ID (old or new) for ledger operations.
		effectiveRoomID := item.AssignedRoomID.UUID
		if input.AssignedRoomID != nil {
			effectiveRoomID = *input.AssignedRoomID
		}

		// --- Handle date changes incrementally ---
		if datesChanged {
			// Rates: soft-delete any that fall outside the new range.
			// Keeps existing rates for dates still in range untouched.
			if err := qtx.SoftDeleteBookedRatesNotInPeriod(ctx, &store.SoftDeleteBookedRatesNotInPeriodParams{
				ReservationItemID: itemID,
				PropertyID:        propertyID,
				Dates:             util.DatesToPGDates(newNights),
			}); err != nil {
				return nil, fmt.Errorf("soft-delete removed rates: %w", err)
			}

			removedNights := util.RemovedDates(oldNights, newNights)
			addedNights := util.AddedDates(oldNights, newNights)

			// Ledger: handle removed nights.
			// If removal is from the end only (shortening departure), use DeleteFromDate.
			// Otherwise (removal at start or both ends), delete all and re-insert.
			complexShift := len(removedNights) > 0 && len(addedNights) > 0

			switch {
			case complexShift:
				// Both removed and added — full replace to avoid date-ordering issues.
				if err := qtx.DeleteLedgerRowsByItem(ctx, &store.DeleteLedgerRowsByItemParams{
					ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true},
					PropertyID:        propertyID,
				}); err != nil {
					return nil, fmt.Errorf("delete all ledger: %w", err)
				}
				if err := insertItemLedgerAndRates(ctx, qtx, itemID, item.ReservationID, propertyID,
					newNights, effectiveRoomID, ratePlanID, item.BookedRoomTypeID); err != nil {
					return nil, err
				}
			case len(removedNights) > 0:
				// Removal at end only (shorten) — use FromDate.
				if err := qtx.DeleteLedgerRowsByItemFromDate(ctx, &store.DeleteLedgerRowsByItemFromDateParams{
					ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true},
					PropertyID:        propertyID,
					FromDate:          pgtype.Date{Time: removedNights[0], Valid: true},
				}); err != nil {
					return nil, fmt.Errorf("delete removed ledger: %w", err)
				}
			case len(addedNights) > 0:
				// Added nights only (lengthen) — insert ledger + rates.
				if err := insertItemLedgerAndRates(ctx, qtx, itemID, item.ReservationID, propertyID,
					addedNights, effectiveRoomID, ratePlanID, item.BookedRoomTypeID); err != nil {
					return nil, err
				}
			}

			// Recompute stay_period_envelope
			if err := recomputeEnvelope(ctx, qtx, s.log, item.ReservationID, propertyID); err != nil {
				return nil, err
			}
		}

		// --- Handle room assignment change without date change ---
		if !datesChanged && roomChanged {
			if err := qtx.UpdateLedgerRowRoom(ctx, &store.UpdateLedgerRowRoomParams{
				ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true},
				PropertyID:        propertyID,
				NewRoomID:         *input.AssignedRoomID,
			}); err != nil {
				return nil, fmt.Errorf("update ledger room: %w", err)
			}
		}

		if _, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{UUID: item.BookedRoomTypeID, Valid: true},
			StayPeriod:       newStayPeriod,
			RatePlanID:       ratePlanID,
			AssignedRoomID:   newRoomID,
			Status:           item.Status,
			AdultsCount:      int32(input.AdultsCount),
			ChildrenCount:    int32(input.ChildrenCount),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update item: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "item_updated", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}
		return s.fetchAndExpandReservation(ctx, qtx, item.ReservationID, propertyID)
	})
}

// UpdateItemStayPeriod changes an item's arrival and/or departure dates.
// For checked_in items, calls ShortenStay to handle future ledger/rates.
func (s *Service) UpdateItemStayPeriod(ctx context.Context, itemID uuid.UUID, arrival, departure time.Time) (*ItemResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ItemResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		if !departure.After(arrival) {
			return nil, ErrInvalidDates.WithMessage("departure must be after arrival")
		}

		newStayPeriod := util.ToRange(arrival, departure)
		version := helpers.GetIfMatchVersion(ctx)

		if item.Status == store.OperationsReservationItemStatusCheckedIn {
			if err := requirePostCheckinPermission(ctx, item); err != nil {
				return nil, err
			}
			// Arrival date cannot change after check-in — it would corrupt ledger
			// housekeeping in ShortenStay which uses the original arrival.
			// Compare calendar dates, not exact timestamps (clock skew between Go and DB).
			arrivalDay := arrival.Truncate(24 * time.Hour)
			itemArrivalDay := item.StayPeriod.Lower.Time.Truncate(24 * time.Hour)
			if !arrivalDay.Equal(itemArrivalDay) {
				return nil, ErrInvalidDates.WithMessage("cannot change arrival date for checked-in items")
			}
			if err := s.ShortenStay(ctx, qtx, item, departure); err != nil {
				return nil, err
			}
		}

		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			StayPeriod:       newStayPeriod,
			AssignedRoomID:   item.AssignedRoomID,
			RatePlanID:       item.RatePlanID,
			Status:           item.Status,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update stay period: %w", err)
		}

		// Update envelope on the reservation
		if err := recomputeEnvelope(ctx, qtx, s.log, item.ReservationID, propertyID); err != nil {
			return nil, err
		}

		if err := notifyReservationChange(ctx, qtx, "item_updated", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		return itemToResponse(&updated), nil
	})
}

// AssignRoom assigns a room to an item. Respects do-not-move flag —
// override requires reservations:override_dnm permission + recorded reason.
func (s *Service) AssignRoom(ctx context.Context, itemID uuid.UUID, input AssignRoomInput) (*ItemResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ItemResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		if item.DoNotMove {
			if !helpers.HasPermission(ctx, "reservations:override_dnm") {
				return nil, ErrDoNotMove
			}
			if input.OverrideDnmReason == "" {
				return nil, ErrDoNotMove.WithMessage("override_dnm_reason is required when overriding do-not-move")
			}
			s.log.Info("DNM override", "item_id", itemID, "reason", input.OverrideDnmReason)
		}
		// Post-checkin room move requires additional permission.
		if err := requirePostCheckinPermission(ctx, item); err != nil {
			return nil, err
		}

		version := helpers.GetIfMatchVersion(ctx)
		newRoomID := uuid.NullUUID{UUID: input.RoomID, Valid: true}

		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			AssignedRoomID:   newRoomID,
			StayPeriod:       item.StayPeriod,
			RatePlanID:       item.RatePlanID,
			Status:           item.Status,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("assign room: %w", err)
		}

		// Update ledger rows to reflect new room
		if err := qtx.UpdateLedgerRowRoom(ctx, &store.UpdateLedgerRowRoomParams{
			ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true}, PropertyID: propertyID, NewRoomID: input.RoomID,
		}); err != nil {
			return nil, fmt.Errorf("update ledger: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "room_assigned", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		return itemToResponse(&updated), nil
	})
}

// UpdateItemRoomType changes an item's room type. Optionally retains the original price.
// overrideDnmReason is required when do_not_move is set and caller has reservations:override_dnm.
func (s *Service) UpdateItemRoomType(ctx context.Context, itemID uuid.UUID, newRoomTypeID uuid.UUID, retainPrice bool, overrideDnmReason string) (*ItemResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ItemResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		if item.DoNotMove {
			if !helpers.HasPermission(ctx, "reservations:override_dnm") {
				return nil, ErrDoNotMove
			}
			if overrideDnmReason == "" {
				return nil, ErrDoNotMove.WithMessage("override_dnm_reason is required when overriding do-not-move")
			}
			s.log.Info("DNM override on room type change", "item_id", itemID, "reason", overrideDnmReason)
		}
		// Post-checkin room type change requires additional permission.
		if err := requirePostCheckinPermission(ctx, item); err != nil {
			return nil, err
		}

		version := helpers.GetIfMatchVersion(ctx)

		// Preserve prices if retainPrice=true
		var updated store.OperationsReservationItem
		if retainPrice {
			updated, err = qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
				ID: itemID, Version: version,
				BookedRoomTypeID: uuid.NullUUID{UUID: newRoomTypeID, Valid: true},
				StayPeriod:       item.StayPeriod,
				RatePlanID:       item.RatePlanID,
				AssignedRoomID:   item.AssignedRoomID,
				Status:           item.Status,
				AdultsCount:      item.AdultsCount,
				ChildrenCount:    item.ChildrenCount,
			})
		} else {
			// Clear rate plan so new rates can be generated for the new room type.
			updated, err = qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
				ID: itemID, Version: version,
				BookedRoomTypeID: uuid.NullUUID{UUID: newRoomTypeID, Valid: true},
				StayPeriod:       item.StayPeriod,
				RatePlanID:       uuid.NullUUID{},
				AssignedRoomID:   item.AssignedRoomID,
				Status:           item.Status,
				AdultsCount:      item.AdultsCount,
				ChildrenCount:    item.ChildrenCount,
			})
		}
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update room type: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "room_type_changed", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		return itemToResponse(&updated), nil
	})
}

// UpdateItemRatePlan changes an item's rate plan. Checks daily room capacity.
func (s *Service) UpdateItemRatePlan(ctx context.Context, itemID uuid.UUID, newRatePlanID uuid.UUID, retainPrice bool) (*ItemResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ItemResponse, error) {
		item, err := qtx.GetReservationItem(ctx, itemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get item: %w", err)
		}

		// Post-checkin rate plan change requires additional permission.
		if err := requirePostCheckinPermission(ctx, item); err != nil {
			return nil, err
		}

		// Check daily room capacity for the new rate plan (R-RES-RATE-005 / M12).
		// GetRatePlanCapacity uses 3-tier inheritance: daily_price_grid > seasonal_rates > base_rates.
		dates := util.NightsBetween(item.StayPeriod.Lower.Time, item.StayPeriod.Upper.Time)
		if len(dates) > 0 {
			for _, d := range dates {
				// maxCapacity == 0 means unlimited (SQL returns 0 for NULL).
				maxCapacity, err := qtx.GetRatePlanCapacity(ctx, &store.GetRatePlanCapacityParams{
					RatePlanID:   newRatePlanID,
					CalendarDate: pgtype.Date{Time: d, Valid: true},
					PropertyID:   propertyID,
				})
				if err != nil {
					return nil, fmt.Errorf("check rate plan capacity: %w", err)
				}
				if maxCapacity > 0 {
					used, err := qtx.CountRatePlanUsage(ctx, &store.CountRatePlanUsageParams{
						RatePlanID:    uuid.NullUUID{UUID: newRatePlanID, Valid: true},
						CalendarDate:  pgtype.Date{Time: d, Valid: true},
						PropertyID:    propertyID,
						ExcludeItemID: itemID,
					})
					if err != nil {
						return nil, fmt.Errorf("count rate plan usage: %w", err)
					}
					if used >= maxCapacity {
						return nil, ErrRatePlanCapacity.WithMessage(
							fmt.Sprintf("rate plan capacity of %d exceeded on %s", maxCapacity, d.Format("2006-01-02")),
						)
					}
				}
			}
		}

		version := helpers.GetIfMatchVersion(ctx)
		ratePlanID := uuid.NullUUID{UUID: newRatePlanID, Valid: true}

		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			RatePlanID:       ratePlanID,
			StayPeriod:       item.StayPeriod,
			AssignedRoomID:   item.AssignedRoomID,
			Status:           item.Status,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update rate plan: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "rate_plan_changed", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		return itemToResponse(&updated), nil
	})
}

// AddItem adds a new item to an existing non-terminal reservation.
func (s *Service) AddItem(ctx context.Context, id uuid.UUID, input AddItemInput) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*ReservationResponse, error) {
		res, err := qtx.GetReservation(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apierror.ErrNotFound
			}
			return nil, fmt.Errorf("get reservation: %w", err)
		}

		if IsTerminalReservationStatus(ReservationStatus(res.Status)) {
			return nil, ErrTerminal
		}

		itemResp, err := insertSingleItem(
			ctx, qtx, input.CreateItemInput,
			struct {
				ReservationStatus ReservationStatus
				ItemStatus        ItemStatus
			}{ReservationStatus: StatusConfirmed, ItemStatus: ItemStatusBooked},
			propertyID, id,
		)
		if err != nil {
			return nil, fmt.Errorf("insert item: %w", err)
		}

		// Update stay_period_envelope to include new item's range
		if err := recomputeEnvelope(ctx, qtx, s.log, id, propertyID); err != nil {
			return nil, err
		}

		if err := notifyReservationChange(ctx, qtx, "item_added", id); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}

		updatedRes, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get updated reservation: %w", err)
		}
		response := reservationFromRow(&updatedRes)
		response.Items = append(response.Items, itemResp)
		return response, nil
	})
}

// requirePostCheckinPermission returns ErrMissingPermission if the item is checked_in
// and the caller lacks reservations:post_checkin_mutate.
func requirePostCheckinPermission(ctx context.Context, item store.OperationsReservationItem) error {
	if ItemStatus(item.Status) == ItemStatusCheckedIn && !helpers.HasPermission(ctx, "reservations:post_checkin_mutate") {
		return ErrMissingPermission.WithMessage("reservations:post_checkin_mutate required for checked-in items")
	}
	return nil
}

// fetchAndExpandReservation fetches a reservation by ID and returns an expanded response.
// Used by UpdateItem to avoid repeating the fetch→hydrate→expand pattern.
func (s *Service) fetchAndExpandReservation(ctx context.Context, qtx *store.Queries, reservationID, propertyID uuid.UUID) (*ReservationResponse, error) {
	resRow, err := qtx.GetReservation(ctx, reservationID)
	if err != nil {
		return nil, fmt.Errorf("get reservation: %w", err)
	}
	resp := reservationFromRow(&resRow)
	expandInclude(ctx, qtx, resp, IncludeFlags{Items: true}, propertyID, reservationID, uuid.NullUUID{UUID: resRow.PrimaryGuestID, Valid: true}, s.log)
	return resp, nil
}

// recomputeEnvelope recalculates stay_period_envelope from all items.
// Uses the reservation's own version (fetched inside) to avoid version mismatch
// when called from item-level mutation paths where context has the item's version.
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
