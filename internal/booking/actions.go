package booking

// Core Requirements: R-RES-CRUD-004, R-RES-CRUD-005, R-RES-CRUD-006, R-RES-LIFECYCLE-001, ADR-015

import (
	"context"
	"errors"
	"fmt"
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

// CheckinReservation checks in all items on a reservation. All items must have assigned_room_id.
func (s *Service) CheckinReservation(ctx context.Context, id uuid.UUID) (*BatchResult, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*BatchResult, error) {
		items, err := qtx.GetReservationItems(ctx, &store.GetReservationItemsParams{
			ReservationID: id, PropertyID: propertyID,
		})
		if err != nil {
			return nil, fmt.Errorf("get items: %w", err)
		}

		var result BatchResult
		for _, item := range items {
			ir := BatchResultItem{ItemID: item.ID.String()}
			if item.Status != store.OperationsReservationItemStatusBooked {
				ir.Status = "failed"
				ir.Error = &BatchError{Code: "INVALID_STATUS", Message: "item must be booked"}
				result.Results = append(result.Results, ir)
				continue
			}
			if !item.AssignedRoomID.Valid {
				ir.Status = "failed"
				ir.Error = &BatchError{Code: "UNASSIGNED_ITEMS", Message: "missing assigned_room_id"}
				result.Results = append(result.Results, ir)
				continue
			}

			updated, updateErr := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
				ID: item.ID, Version: item.Version,
				BookedRoomTypeID: uuid.NullUUID{},
				Status:           store.OperationsReservationItemStatusCheckedIn,
				AssignedRoomID:   item.AssignedRoomID,
				StayPeriod:       item.StayPeriod,
				RatePlanID:       item.RatePlanID,
				AdultsCount:      item.AdultsCount,
				ChildrenCount:    item.ChildrenCount,
			})
			if updateErr != nil {
				ir.Status = "failed"
				ir.Error = &BatchError{Code: "UPDATE_FAILED", Message: updateErr.Error()}
			} else {
				ir.Status = "ok"
				ir.Item = itemToResponse(&updated)
			}
			result.Results = append(result.Results, ir)
		}

		// Only run rollup if there were items to process
		if len(items) > 0 {
			if _, err := rollupAndNotify(ctx, qtx, id, propertyID); err != nil {
				return nil, err
			}
		}
		return &result, nil
	})
}

// CheckinItem checks in a single reservation item.
func (s *Service) CheckinItem(ctx context.Context, itemID uuid.UUID) (*ItemResponse, error) {
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

		if item.Status != store.OperationsReservationItemStatusBooked {
			return nil, ErrInvalidTransition.WithMessage("item must be booked")
		}
		if !item.AssignedRoomID.Valid {
			return nil, ErrUnassignedItems
		}

		version := helpers.GetIfMatchVersion(ctx)
		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			Status:           store.OperationsReservationItemStatusCheckedIn,
			AssignedRoomID:   item.AssignedRoomID,
			StayPeriod:       item.StayPeriod,
			RatePlanID:       item.RatePlanID,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("checkin item: %w", err)
		}

		if _, err := rollupAndNotify(ctx, qtx, item.ReservationID, propertyID); err != nil {
			return nil, err
		}
		return itemToResponse(&updated), nil
	})
}

// CheckoutReservation checks out all checked-in items.
func (s *Service) CheckoutReservation(ctx context.Context, id uuid.UUID) (*BatchResult, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}

	return db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (*BatchResult, error) {
		items, err := qtx.GetReservationItems(ctx, &store.GetReservationItemsParams{
			ReservationID: id, PropertyID: propertyID,
		})
		if err != nil {
			return nil, fmt.Errorf("get items: %w", err)
		}

		var result BatchResult
		for _, item := range items {
			ir := BatchResultItem{ItemID: item.ID.String()}
			switch item.Status {
			case store.OperationsReservationItemStatusCheckedOut,
				store.OperationsReservationItemStatusCancelled,
				store.OperationsReservationItemStatusArchived:
				ir.Status = "ok"
			case store.OperationsReservationItemStatusCheckedIn:
				updated, updateErr := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
					ID: item.ID, Version: item.Version,
					BookedRoomTypeID: uuid.NullUUID{},
					Status:           store.OperationsReservationItemStatusCheckedOut,
					AssignedRoomID:   item.AssignedRoomID,
					StayPeriod:       item.StayPeriod,
					RatePlanID:       item.RatePlanID,
					AdultsCount:      item.AdultsCount,
					ChildrenCount:    item.ChildrenCount,
				})
				if updateErr != nil {
					ir.Status = "failed"
					ir.Error = &BatchError{Code: "UPDATE_FAILED", Message: updateErr.Error()}
				} else {
					ir.Status = "ok"
					ir.Item = itemToResponse(&updated)
				}
			default:
				ir.Status = "failed"
				ir.Error = &BatchError{Code: "INVALID_STATUS", Message: "item must be checked in"}
			}
			result.Results = append(result.Results, ir)
		}

		if len(items) > 0 {
			if _, err := rollupAndNotify(ctx, qtx, id, propertyID); err != nil {
				return nil, err
			}
		}
		return &result, nil
	})
}

// CheckoutItem checks out a single reservation item.
func (s *Service) CheckoutItem(ctx context.Context, itemID uuid.UUID) (*ItemResponse, error) {
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

		if item.Status != store.OperationsReservationItemStatusCheckedIn {
			return nil, ErrInvalidTransition.WithMessage("item must be checked in")
		}

		version := helpers.GetIfMatchVersion(ctx)
		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			Status:           store.OperationsReservationItemStatusCheckedOut,
			AssignedRoomID:   item.AssignedRoomID,
			StayPeriod:       item.StayPeriod,
			RatePlanID:       item.RatePlanID,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("checkout item: %w", err)
		}

		if _, err := rollupAndNotify(ctx, qtx, item.ReservationID, propertyID); err != nil {
			return nil, err
		}
		return itemToResponse(&updated), nil
	})
}

// CancelReservation cancels a reservation. Hold → cancelled directly. Confirmed → cancelled with fee audit.
func (s *Service) CancelReservation(ctx context.Context, id uuid.UUID, input CancelInput) (*ReservationResponse, error) {
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

		if err := ValidateReservationTransition(ReservationStatus(res.Status), StatusCancelled); err != nil {
			return nil, err
		}

		s.log.Info("cancelling reservation", "reservation_id", id, "reason", input.ReasonCode, "fee_pence", input.FeePence, "waive_fee", input.WaiveFee)
		version := helpers.GetIfMatchVersion(ctx)
		if err := qtx.CancelReservationItems(ctx, &store.CancelReservationItemsParams{
			ReservationID: id, PropertyID: propertyID,
		}); err != nil {
			return nil, fmt.Errorf("cancel items: %w", err)
		}
		if err := qtx.DeleteLedgerForReservation(ctx, &store.DeleteLedgerForReservationParams{
			ReservationID: uuid.NullUUID{UUID: id, Valid: true}, PropertyID: propertyID,
		}); err != nil {
			return nil, fmt.Errorf("delete ledger: %w", err)
		}

		rows, err := qtx.UpdateReservationStatus(ctx, &store.UpdateReservationStatusParams{
			ID: id, Version: version, Status: store.OperationsReservationStatusCancelled,
		})
		if err != nil {
			return nil, fmt.Errorf("update status: %w", err)
		}
		if rows == 0 {
			return nil, ErrVersionMismatch
		}

		// Re-fetch to get updated row
		// @Ai - can we not just use returning * in SQLC rather than get it? or is it more
		// for the items included etc
		// Might be better to just return a 200 that its completed and not return the
		// reservation, or prhaps just the id?
		updated, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("re-fetch: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "cancelled", id); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}
		return reservationFromRow(&updated), nil
	})
}

// CancelItem cancels a single reservation item.
func (s *Service) CancelItem(ctx context.Context, itemID uuid.UUID, input CancelInput) (*ItemResponse, error) {
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

		if err := ValidateItemTransition(ItemStatus(item.Status), ItemStatusCancelled); err != nil {
			return nil, err
		}

		s.log.Info("cancelling item", "item_id", itemID, "reason", input.ReasonCode, "fee_pence", input.FeePence, "waive_fee", input.WaiveFee)
		version := helpers.GetIfMatchVersion(ctx)
		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			Status:           store.OperationsReservationItemStatusCancelled,
			AssignedRoomID:   item.AssignedRoomID,
			StayPeriod:       item.StayPeriod,
			RatePlanID:       item.RatePlanID,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("cancel item: %w", err)
		}

		if err := qtx.DeleteLedgerRowsByItem(ctx, &store.DeleteLedgerRowsByItemParams{
			ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true}, PropertyID: propertyID,
		}); err != nil {
			return nil, fmt.Errorf("delete ledger: %w", err)
		}

		if _, err := rollupAndNotify(ctx, qtx, item.ReservationID, propertyID); err != nil {
			return nil, err
		}
		return itemToResponse(&updated), nil
	})
}

// MarkNoShow marks an item as no-show. Requires stay period to have started.
func (s *Service) MarkNoShow(ctx context.Context, itemID uuid.UUID) (*ItemResponse, error) {
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

		if err := ValidateItemTransition(ItemStatus(item.Status), ItemStatusNoShow); err != nil {
			return nil, err
		}
		if time.Now().Before(item.StayPeriod.Lower.Time) {
			return nil, ErrInvalidDates.WithMessage("cannot mark no-show before arrival")
		}

		version := helpers.GetIfMatchVersion(ctx)
		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{},
			Status:           store.OperationsReservationItemStatusNoShow,
			AssignedRoomID:   item.AssignedRoomID,
			StayPeriod:       item.StayPeriod,
			RatePlanID:       item.RatePlanID,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("no-show: %w", err)
		}

		if err := qtx.DeleteLedgerRowsByItemFromDate(ctx, &store.DeleteLedgerRowsByItemFromDateParams{
			ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true}, PropertyID: propertyID,
			FromDate: pgtype.Date{Time: time.Now().Truncate(24 * time.Hour), Valid: true},
		}); err != nil {
			return nil, fmt.Errorf("delete future ledger: %w", err)
		}

		if _, err := rollupAndNotify(ctx, qtx, item.ReservationID, propertyID); err != nil {
			return nil, err
		}
		return itemToResponse(&updated), nil
	})
}

// ReactivateReservation reactivates a cancelled reservation back to confirmed.
func (s *Service) ReactivateReservation(ctx context.Context, id uuid.UUID) (*ReservationResponse, error) {
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

		if err := ValidateReservationTransition(ReservationStatus(res.Status), StatusConfirmed); err != nil {
			return nil, err
		}

		version := helpers.GetIfMatchVersion(ctx)
		rows, err := qtx.UpdateReservationStatus(ctx, &store.UpdateReservationStatusParams{
			ID: id, Version: version, Status: store.OperationsReservationStatusConfirmed,
		})
		if err != nil {
			return nil, fmt.Errorf("reactivate: %w", err)
		}
		if rows == 0 {
			return nil, ErrVersionMismatch
		}

		if err := qtx.ReactivateReservationItems(ctx, &store.ReactivateReservationItemsParams{
			ReservationID: id, PropertyID: propertyID,
		}); err != nil {
			return nil, fmt.Errorf("reactivate items: %w", err)
		}

		updated, err := qtx.GetReservation(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("re-fetch after reactivate: %w", err)
		}

		if err := notifyReservationChange(ctx, qtx, "reactivated", id); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}
		return reservationFromRow(&updated), nil
	})
}

// ShortenStay shortens an item's stay. Internal — called by UpdateItemStayPeriod when checked_in.
func (s *Service) ShortenStay(ctx context.Context, qtx *store.Queries, item store.OperationsReservationItem, newDeparture time.Time) error {
	if !item.StayPeriod.Lower.Time.Before(newDeparture) {
		return ErrInvalidDates.WithMessage("new departure must be after arrival")
	}

	oldDates := util.NightsBetween(item.StayPeriod.Lower.Time, item.StayPeriod.Upper.Time)
	newDates := util.NightsBetween(item.StayPeriod.Lower.Time, newDeparture)

	removeSet := make(map[string]bool, len(oldDates))
	for _, d := range oldDates {
		removeSet[d.Format("2006-01-02")] = true
	}
	for _, d := range newDates {
		delete(removeSet, d.Format("2006-01-02"))
	}

	var pgDates []pgtype.Date
	for dateStr := range removeSet {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			pgDates = append(pgDates, pgtype.Date{Time: t, Valid: true})
		}
	}

	if len(pgDates) > 0 {
		if err := qtx.SoftDeleteBookedRatesNotInPeriod(ctx, &store.SoftDeleteBookedRatesNotInPeriodParams{
			ReservationItemID: item.ID, PropertyID: item.PropertyID, Dates: pgDates,
		}); err != nil {
			return fmt.Errorf("soft delete rates: %w", err)
		}
	}

	// Delete future ledger rows to release inventory
	if err := qtx.DeleteLedgerRowsByItemFromDate(ctx, &store.DeleteLedgerRowsByItemFromDateParams{
		ReservationItemID: uuid.NullUUID{UUID: item.ID, Valid: true}, PropertyID: item.PropertyID,
		FromDate: pgtype.Date{Time: newDeparture, Valid: true},
	}); err != nil {
		return fmt.Errorf("delete future ledger: %w", err)
	}

	return nil
}

// rollupAndNotify runs ADR-015 rollup and emits reservation_changes notification.
func rollupAndNotify(ctx context.Context, qtx *store.Queries, reservationID, propertyID uuid.UUID) (string, error) {
	rollupStatus, err := qtx.RollupReservationStatus(ctx, reservationID)
	if err != nil {
		return "", fmt.Errorf("rollup: %w", err)
	}
	if rollupStatus == "" {
		return "", nil
	}

	res, err := qtx.GetReservation(ctx, reservationID)
	if err != nil {
		return "", fmt.Errorf("get reservation for rollup: %w", err)
	}

	rows, err := qtx.UpdateReservationStatus(ctx, &store.UpdateReservationStatusParams{
		ID: reservationID, Version: res.Version,
		Status: store.OperationsReservationStatus(rollupStatus),
	})
	if err != nil {
		return "", fmt.Errorf("apply rollup: %w", err)
	}
	if rows == 0 {
		return "", ErrVersionMismatch
	}

	if err := notifyReservationChange(ctx, qtx, "rollup", reservationID); err != nil {
		return "", fmt.Errorf("notify: %w", err)
	}
	return rollupStatus, nil
}
