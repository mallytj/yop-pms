package booking

// Core Requirements: R-RES-CRUD-003, R-RES-CRUD-010, R-RES-CRUD-011, R-RES-CRUD-012,
// R-RES-RATE-001, R-RES-RATE-002, R-RES-RATE-003, ADR-021

// update.go — Reservation mutation and miscellaneous endpoints:
//   - HTTP handlers: UpdateMetadata, AddItem, UpdateItem, AssignRoom,
//     Availability, GetFolio, CancellationQuote, GetBookedRates, UpdateBookedRates,
//     AdjustRate, ApproveAdjustments
//   - Service: UpdateItem, UpdateItemStayPeriod, AssignRoom, UpdateItemRoomType,
//     UpdateItemRatePlan, AddItem, UpdateBookedRates, AdjustRate, ApproveAdjustments

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
	"github.com/lexxcode1/yop-pms/internal/platform/validation"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// ─────────────────────────────────────────────────────────────────────────────
// HTTP handlers: metadata + items
// ─────────────────────────────────────────────────────────────────────────────

// UpdateMetadata handles PATCH /reservations/{id}.
func (h *Handler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUIDParam(r, "id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	var input UpdateMetadataInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	if input.Notes != nil && len(*input.Notes) > 2500 {
		platformjson.WriteError(w, r, apierror.ErrUnprocessable.WithMessage("notes must be at most 2500 characters"))
		return
	}
	res, svcErr := h.svc.UpdateMetadata(r.Context(), id, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// AddItem handles POST /reservations/{id}/items.
func (h *Handler) AddItem(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUIDParam(r, "id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	var input AddItemInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	if errs := validation.Struct(input, "operations.reservation_items"); len(errs) > 0 {
		platformjson.WriteError(w, r, apierror.ErrUnprocessable.WithMessage(errs[0].Error()))
		return
	}
	res, svcErr := h.svc.AddItem(r.Context(), id, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusCreated, res)
}

// UpdateItem handles PATCH /reservations/{id}/items/{item_id}.
func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	var input CreateItemInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	if errs := validation.Struct(input, "operations.reservation_items"); len(errs) > 0 {
		platformjson.WriteError(w, r, apierror.ErrUnprocessable.WithMessage(errs[0].Error()))
		return
	}
	res, svcErr := h.svc.UpdateItem(r.Context(), itemID, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// AssignRoom handles PATCH /reservations/{id}/items/{item_id}/assign-room.
func (h *Handler) AssignRoom(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	var input AssignRoomInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	res, svcErr := h.svc.AssignRoom(r.Context(), itemID, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP handlers: availability + misc stubs
// ─────────────────────────────────────────────────────────────────────────────

// Availability handles GET /reservations/availability.
func (h *Handler) Availability(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rtRaw := q.Get("room_type_id")
	rtID, err := uuid.Parse(rtRaw)
	if err != nil {
		platformjson.WriteError(w, r, apierror.ErrBadRequest.WithMessage("room_type_id must be a valid UUID"))
		return
	}
	startDate, apiErr := httputil.ParseDateParam(r, "start_date")
	if apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	endDate, apiErr := httputil.ParseDateParam(r, "end_date")
	if apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	propertyID := helpers.GetPropertyIDFromCtx(r.Context())
	result, svcErr := h.svc.CheckAvailability(r.Context(), propertyID, rtID, startDate, endDate)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, result)
}

// GetFolio handles GET /reservations/{id}/folios/{folio_id}.
func (h *Handler) GetFolio(w http.ResponseWriter, _ *http.Request) {
	platformjson.WriteJSON(w, http.StatusNotImplemented, map[string]string{"status": "not_implemented"})
}

// CancellationQuote handles GET /reservations/{id}/cancellation-quote.
func (h *Handler) CancellationQuote(w http.ResponseWriter, _ *http.Request) {
	platformjson.WriteJSON(w, http.StatusNotImplemented, CancellationQuoteResponse{
		FeePence: nil,
		Status:   "not_implemented",
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP handlers: rates
// ─────────────────────────────────────────────────────────────────────────────

// GetBookedRates handles GET /reservations/{id}/items/{item_id}/booked-rates.
func (h *Handler) GetBookedRates(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	res, svcErr := h.svc.GetBookedRates(r.Context(), itemID)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// UpdateBookedRates handles PATCH /reservations/{id}/items/{item_id}/booked-rates.
func (h *Handler) UpdateBookedRates(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	var input RateAdjustInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	res, svcErr := h.svc.UpdateBookedRates(r.Context(), itemID, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// AdjustRate handles POST /reservations/{id}/items/{item_id}/adjust-rate.
func (h *Handler) AdjustRate(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	var input RateAdjustInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	res, svcErr := h.svc.AdjustRate(r.Context(), itemID, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// ApproveAdjustments handles POST /reservations/{id}/items/{item_id}/booked-rates/approve.
func (h *Handler) ApproveAdjustments(w http.ResponseWriter, r *http.Request) {
	itemID, err := httputil.ParseUUIDParam(r, "item_id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	res, svcErr := h.svc.ApproveAdjustments(r.Context(), itemID)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// ─────────────────────────────────────────────────────────────────────────────
// Service: Item mutations
// ─────────────────────────────────────────────────────────────────────────────

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

		if input.RoomTypeID != uuid.Nil && input.RoomTypeID != item.BookedRoomTypeID {
			if _, err := s.UpdateItemRoomType(ctx, itemID, input.RoomTypeID, false, ""); err != nil {
				return nil, err
			}
			return s.fetchAndExpandReservation(ctx, qtx, item.ReservationID, propertyID)
		}

		if input.RatePlanID != nil && (!item.RatePlanID.Valid || item.RatePlanID.UUID != *input.RatePlanID) {
			if _, err := s.UpdateItemRatePlan(ctx, itemID, *input.RatePlanID, false); err != nil {
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

		effectiveRoomID := item.AssignedRoomID.UUID
		if input.AssignedRoomID != nil {
			effectiveRoomID = *input.AssignedRoomID
		}

		if datesChanged {
			if err := qtx.SoftDeleteBookedRatesNotInPeriod(ctx, &store.SoftDeleteBookedRatesNotInPeriodParams{
				ReservationItemID: itemID, PropertyID: propertyID,
				Dates: util.DatesToPGDates(newNights),
			}); err != nil {
				return nil, fmt.Errorf("soft-delete removed rates: %w", err)
			}

			removedNights := util.RemovedDates(oldNights, newNights)
			addedNights := util.AddedDates(oldNights, newNights)

			complexShift := len(removedNights) > 0 && len(addedNights) > 0
			switch {
			case complexShift:
				if err := qtx.DeleteLedgerRowsByItem(ctx, &store.DeleteLedgerRowsByItemParams{
					ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true}, PropertyID: propertyID,
				}); err != nil {
					return nil, fmt.Errorf("delete all ledger: %w", err)
				}
				if err := insertItemLedgerAndRates(ctx, qtx, itemID, item.ReservationID, propertyID,
					newNights, effectiveRoomID, ratePlanID, item.BookedRoomTypeID); err != nil {
					return nil, err
				}
			case len(removedNights) > 0:
				if err := qtx.DeleteLedgerRowsByItemFromDate(ctx, &store.DeleteLedgerRowsByItemFromDateParams{
					ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true}, PropertyID: propertyID,
					FromDate: pgtype.Date{Time: removedNights[0], Valid: true},
				}); err != nil {
					return nil, fmt.Errorf("delete removed ledger: %w", err)
				}
			case len(addedNights) > 0:
				if err := insertItemLedgerAndRates(ctx, qtx, itemID, item.ReservationID, propertyID,
					addedNights, effectiveRoomID, ratePlanID, item.BookedRoomTypeID); err != nil {
					return nil, err
				}
			}

			if err := recomputeEnvelope(ctx, qtx, s.log, item.ReservationID, propertyID); err != nil {
				return nil, err
			}
		}

		if !datesChanged && roomChanged {
			if err := qtx.UpdateLedgerRowRoom(ctx, &store.UpdateLedgerRowRoomParams{
				ReservationItemID: uuid.NullUUID{UUID: itemID, Valid: true}, PropertyID: propertyID, NewRoomID: *input.AssignedRoomID,
			}); err != nil {
				return nil, fmt.Errorf("update ledger room: %w", err)
			}
		}

		if _, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{UUID: item.BookedRoomTypeID, Valid: true},
			StayPeriod:       newStayPeriod, RatePlanID: ratePlanID, AssignedRoomID: newRoomID,
			Status: item.Status, AdultsCount: int32(input.AdultsCount), ChildrenCount: int32(input.ChildrenCount),
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
			ID: itemID, Version: version, BookedRoomTypeID: uuid.NullUUID{},
			StayPeriod: newStayPeriod, AssignedRoomID: item.AssignedRoomID, RatePlanID: item.RatePlanID,
			Status: item.Status, AdultsCount: item.AdultsCount, ChildrenCount: item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("update stay period: %w", err)
		}
		if err := recomputeEnvelope(ctx, qtx, s.log, item.ReservationID, propertyID); err != nil {
			return nil, err
		}
		if err := notifyReservationChange(ctx, qtx, "item_updated", item.ReservationID); err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}
		return itemToResponse(&updated), nil
	})
}

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
		if err := requirePostCheckinPermission(ctx, item); err != nil {
			return nil, err
		}
		version := helpers.GetIfMatchVersion(ctx)
		newRoomID := uuid.NullUUID{UUID: input.RoomID, Valid: true}
		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version, BookedRoomTypeID: uuid.NullUUID{},
			AssignedRoomID: newRoomID, StayPeriod: item.StayPeriod, RatePlanID: item.RatePlanID,
			Status: item.Status, AdultsCount: item.AdultsCount, ChildrenCount: item.ChildrenCount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrVersionMismatch
			}
			return nil, fmt.Errorf("assign room: %w", err)
		}
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
		if err := requirePostCheckinPermission(ctx, item); err != nil {
			return nil, err
		}
		version := helpers.GetIfMatchVersion(ctx)
		params := &store.UpdateReservationItemParams{
			ID: itemID, Version: version,
			BookedRoomTypeID: uuid.NullUUID{UUID: newRoomTypeID, Valid: true},
			StayPeriod:       item.StayPeriod,
			RatePlanID:       item.RatePlanID,
			AssignedRoomID:   item.AssignedRoomID,
			Status:           item.Status,
			AdultsCount:      item.AdultsCount,
			ChildrenCount:    item.ChildrenCount,
		}
		if !retainPrice {
			params.RatePlanID = uuid.NullUUID{}
		}
		updated, err := qtx.UpdateReservationItem(ctx, params)
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
		if err := requirePostCheckinPermission(ctx, item); err != nil {
			return nil, err
		}
		dates := util.NightsBetween(item.StayPeriod.Lower.Time, item.StayPeriod.Upper.Time)
		if len(dates) > 0 {
			for _, d := range dates {
				maxCapacity, err := qtx.GetRatePlanCapacity(ctx, &store.GetRatePlanCapacityParams{
					RatePlanID: newRatePlanID, CalendarDate: pgtype.Date{Time: d, Valid: true}, PropertyID: propertyID,
				})
				if err != nil {
					return nil, fmt.Errorf("check rate plan capacity: %w", err)
				}
				if maxCapacity > 0 {
					used, err := qtx.CountRatePlanUsage(ctx, &store.CountRatePlanUsageParams{
						RatePlanID:   uuid.NullUUID{UUID: newRatePlanID, Valid: true},
						CalendarDate: pgtype.Date{Time: d, Valid: true}, PropertyID: propertyID, ExcludeItemID: itemID,
					})
					if err != nil {
						return nil, fmt.Errorf("count rate plan usage: %w", err)
					}
					if used >= maxCapacity {
						return nil, ErrRatePlanCapacity.WithMessage(
							fmt.Sprintf("rate plan capacity of %d exceeded on %s", maxCapacity, d.Format("2006-01-02")))
					}
				}
			}
		}
		version := helpers.GetIfMatchVersion(ctx)
		updated, err := qtx.UpdateReservationItem(ctx, &store.UpdateReservationItemParams{
			ID: itemID, Version: version, BookedRoomTypeID: uuid.NullUUID{},
			RatePlanID: uuid.NullUUID{UUID: newRatePlanID, Valid: true},
			StayPeriod: item.StayPeriod, AssignedRoomID: item.AssignedRoomID,
			Status: item.Status, AdultsCount: item.AdultsCount, ChildrenCount: item.ChildrenCount,
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
		itemResp, err := insertSingleItem(ctx, qtx, input.CreateItemInput,
			struct {
				ReservationStatus ReservationStatus
				ItemStatus        ItemStatus
			}{ReservationStatus: StatusConfirmed, ItemStatus: ItemStatusBooked},
			propertyID, id)
		if err != nil {
			return nil, fmt.Errorf("insert item: %w", err)
		}
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

// ─────────────────────────────────────────────────────────────────────────────
// Service: Rate management
// ─────────────────────────────────────────────────────────────────────────────

func (s *Service) UpdateBookedRates(ctx context.Context, itemID uuid.UUID, input RateAdjustInput) (*ReservationResponse, error) {
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return nil, ErrNoPropertyContext
	}
	_, err := db.ExecuteTx(ctx, s.pool, s.q, func(qtx *store.Queries) (struct{}, error) {
		for _, adj := range input.Adjustments {
			calDate := pgtype.Date{Time: adj.CalendarDate, Valid: true}
			if adj.Type == AdjustmentPercent {
				rate, err := qtx.GetBaseRateForDate(ctx, &store.GetBaseRateForDateParams{
					ReservationItemID: itemID, CalendarDate: calDate, PropertyID: propertyID,
				})
				if err != nil {
					return struct{}{}, fmt.Errorf("get base rate for %s: %w", adj.CalendarDate.Format("2006-01-02"), err)
				}
				newPrice := max(int32(float64(rate)*(1+float64(adj.Value)/100.0)), 0)
				if err := qtx.SetBaseRateForDate(ctx, &store.SetBaseRateForDateParams{
					ReservationItemID: itemID, CalendarDate: calDate, PropertyID: propertyID, BasePricePence: newPrice,
				}); err != nil {
					return struct{}{}, fmt.Errorf("set base rate for %s: %w", adj.CalendarDate.Format("2006-01-02"), err)
				}
				continue
			}
			newPrice := max(int32(adj.Value), 0)
			if err := qtx.SetBaseRateForDate(ctx, &store.SetBaseRateForDateParams{
				ReservationItemID: itemID, CalendarDate: calDate, PropertyID: propertyID, BasePricePence: newPrice,
			}); err != nil {
				return struct{}{}, fmt.Errorf("set base rate for %s: %w", adj.CalendarDate.Format("2006-01-02"), err)
			}
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}
	item, err := s.q.GetReservationItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get item after rate update: %w", err)
	}
	return s.GetReservation(ctx, item.ReservationID, IncludeFlags{Items: true})
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func requirePostCheckinPermission(ctx context.Context, item store.OperationsReservationItem) error {
	if ItemStatus(item.Status) == ItemStatusCheckedIn && !helpers.HasPermission(ctx, "reservations:post_checkin_mutate") {
		return ErrMissingPermission.WithMessage("reservations:post_checkin_mutate required for checked-in items")
	}
	return nil
}

func (s *Service) fetchAndExpandReservation(ctx context.Context, qtx *store.Queries, reservationID, propertyID uuid.UUID) (*ReservationResponse, error) {
	resRow, err := qtx.GetReservation(ctx, reservationID)
	if err != nil {
		return nil, fmt.Errorf("get reservation: %w", err)
	}
	resp := reservationFromRow(&resRow)
	expandInclude(ctx, qtx, resp, IncludeFlags{Items: true}, propertyID, reservationID, uuid.NullUUID{UUID: resRow.PrimaryGuestID, Valid: true}, s.log)
	return resp, nil
}
