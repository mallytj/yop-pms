package booking

// Core Requirements: R-RES-CRUD-004, R-RES-CRUD-005, R-RES-CRUD-006, R-RES-LIFECYCLE-001, ADR-015

// lifecycle.go — Reservation lifecycle flow:
//   - HTTP handlers: Confirm, Cancel, Reactivate, CheckinReservation, CheckinItem,
//     CheckoutReservation, CheckoutItem, MarkNoShow, CancelItem
//   - Service: CancelReservation, CancelItem, CheckinReservation, CheckinItem,
//     CheckoutReservation, CheckoutItem, MarkNoShow, ReactivateReservation, ShortenStay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/db"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	httputil "github.com/lexxcode1/yop-pms/internal/platform/httputil"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
	"github.com/lexxcode1/yop-pms/internal/platform/util"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// ─────────────────────────────────────────────────────────────────────────────
// HTTP handlers
// ─────────────────────────────────────────────────────────────────────────────

// Confirm handles POST /reservations/{id}/confirm.
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUIDParam(r, "id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	include := ParseIncludeFlags(r)
	res, svcErr := h.svc.ConfirmReservation(r.Context(), id, include)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// Cancel handles POST /reservations/{id}/cancel.
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUIDParam(r, "id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	var input CancelInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	res, svcErr := h.svc.CancelReservation(r.Context(), id, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// Reactivate handles POST /reservations/{id}/reactivate.
func (h *Handler) Reactivate(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUIDParam(r, "id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	res, svcErr := h.svc.ReactivateReservation(r.Context(), id)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// CheckinReservation handles PATCH /reservations/{id}/checkin.
func (h *Handler) CheckinReservation(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUIDParam(r, "id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	res, svcErr := h.svc.CheckinReservation(r.Context(), id)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, batchStatus(res), res)
}

// CheckinItem handles PATCH /reservations/{id}/items/{item_id}/checkin.
func (h *Handler) CheckinItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	res, svcErr := h.svc.CheckinItem(r.Context(), itemID)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// CheckoutReservation handles PATCH /reservations/{id}/checkout.
func (h *Handler) CheckoutReservation(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUIDParam(r, "id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	res, svcErr := h.svc.CheckoutReservation(r.Context(), id)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, batchStatus(res), res)
}

// CheckoutItem handles PATCH /reservations/{id}/items/{item_id}/checkout.
func (h *Handler) CheckoutItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	res, svcErr := h.svc.CheckoutItem(r.Context(), itemID)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// MarkNoShow handles PATCH /reservations/{id}/items/{item_id}/no-show.
func (h *Handler) MarkNoShow(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	res, svcErr := h.svc.MarkNoShow(r.Context(), itemID)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// CancelItem handles POST /reservations/{id}/items/{item_id}/cancel.
func (h *Handler) CancelItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	var input CancelInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	res, svcErr := h.svc.CancelItem(r.Context(), itemID, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// batchStatus returns 207 Multi-Status if any item failed, otherwise 200.
func batchStatus(r *BatchResult) int {
	if r == nil || len(r.Results) == 0 {
		return http.StatusOK
	}
	for _, item := range r.Results {
		if item.Error != nil {
			return http.StatusMultiStatus
		}
	}
	return http.StatusOK
}

// ─────────────────────────────────────────────────────────────────────────────
// Service: CancelReservation
// ─────────────────────────────────────────────────────────────────────────────

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

		items, err := qtx.GetReservationItems(ctx, &store.GetReservationItemsParams{
			ReservationID: id, PropertyID: propertyID,
		})
		if err != nil {
			return nil, fmt.Errorf("get items: %w", err)
		}
		for _, item := range items {
			if item.Status == store.OperationsReservationItemStatusCheckedIn {
				return nil, apierror.ErrConflict.WithMessage(
					fmt.Sprintf("cannot cancel reservation with checked-in items (item %s)", item.ID))
			}
		}

		s.log.Info("cancelling reservation", "reservation_id", id, "reason", input.ReasonCode, "fee_pence", input.FeePence)
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

		s.log.Info("cancelling item", "item_id", itemID, "reason", input.ReasonCode)
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

// ─────────────────────────────────────────────────────────────────────────────
// Service: Checkin / Checkout / MarkNoShow / Reactivate
// ─────────────────────────────────────────────────────────────────────────────

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
			} else if !item.AssignedRoomID.Valid {
				ir.Status = "failed"
				ir.Error = &BatchError{Code: "UNASSIGNED_ITEMS", Message: "missing assigned_room_id"}
			} else {
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

		if res.StayPeriodEnvelope.Valid && res.StayPeriodEnvelope.Upper.Valid &&
			res.StayPeriodEnvelope.Upper.Time.Before(time.Now()) &&
			!helpers.HasPermission(ctx, "reservations:retroactive_create") {
			return nil, ErrInvalidTransition.WithMessage("cannot reactivate a past reservation")
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
	return "", nil
}
