package booking

// Core Requirements: R-RES-CRUD-004, R-RES-CRUD-005, R-RES-CRUD-006, R-RES-LIFECYCLE-001, ADR-009

// lifecycle.go — Reservation lifecycle handlers:
//   - HTTP handlers: Confirm, Cancel, Reactivate, CheckinReservation, CheckinItem,
//     CheckoutReservation, CheckoutItem, MarkNoShow, CancelItem

import (
	"net/http"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	httputil "github.com/lexxcode1/yop-pms/internal/platform/httputil"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
	"github.com/lexxcode1/yop-pms/internal/platform/validation"
)

// ─────────────────────────────────────────────────────────────────────────────
// HTTP handlers
// ─────────────────────────────────────────────────────────────────────────────

// Confirm handles POST /reservations/{id}/confirm.
// Confirm handles POST /{id}/confirm.
//
// @Summary      Confirm reservation
// @Description  Transition reservation from hold to confirmed. Requires attached guest.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header    string               true  "Property UUID"
// @Param        id             path      string               true  "Reservation UUID"
// @Param        If-Match       header    string               true  "Version for optimistic concurrency"
// @Param        include        query     string               false "Comma-separated: items,guest,none"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}  apierror.APIError  "Reservation not found"
// @Failure      409            {object}  apierror.APIError  "Version mismatch or invalid transition"
// @Router       /v1/reservations/{id}/confirm [post]
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
// Cancel handles POST /{id}/cancel.
//
// @Summary      Cancel reservation
// @Description  Cancel a reservation and all its items. Rejects if any item is checked in.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header       string               true  "Property UUID"
// @Param        id             path         string               true  "Reservation UUID"
// @Param        If-Match       header       string               true  "Version for optimistic concurrency"
// @Param        body           body         CancelInput          true  "Cancellation details"
// @Success      200            {object}     ReservationResponse
// @Failure      400            {object}     apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}     apierror.APIError  "Reservation not found"
// @Failure      409            {object}     apierror.APIError  "Version mismatch or checked-in items"
// @Failure      422            {object}     apierror.APIError  "Validation failed"
// @Router       /v1/reservations/{id}/cancel [post]
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
	if errs := validation.Struct(input, "operations.reservations"); len(errs) > 0 {
		platformjson.WriteError(w, r, apierror.ErrUnprocessable.WithMessage(errs[0].Error()))
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
// Reactivate handles POST /{id}/reactivate.
//
// @Summary      Reactivate cancelled reservation
// @Description  Restore a cancelled reservation to confirmed, reactivate items and inventory.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header    string               true  "Property UUID"
// @Param        id             path      string               true  "Reservation UUID"
// @Param        If-Match       header    string               true  "Version for optimistic concurrency"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}  apierror.APIError  "Reservation not found"
// @Failure      409            {object}  apierror.APIError  "Version mismatch or invalid transition or past reservation"
// @Router       /v1/reservations/{id}/reactivate [post]
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
// CheckinReservation handles PATCH /{id}/checkin.
//
// @Summary      Check in reservation
// @Description  Batch check-in all items on a reservation. Returns per-item results.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header    string       true  "Property UUID"
// @Param        id             path      string       true  "Reservation UUID"
// @Success      200            {object}  BatchResult
// @Success      207            {object}  BatchResult  "Partial success (some items failed)"
// @Failure      400            {object}  apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}  apierror.APIError  "Reservation not found"
// @Router       /v1/reservations/{id}/checkin [patch]
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
// CheckinItem handles PATCH /{id}/items/{item_id}/checkin.
//
// @Summary      Check in single item
// @Description  Check in a single reservation item. Requires assigned room.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header       string          true  "Property UUID"
// @Param        id             path         string          true  "Reservation UUID"
// @Param        item_id        path         string          true  "Item UUID"
// @Param        If-Match       header       string          true  "Version for optimistic concurrency"
// @Success      200            {object}     ItemResponse
// @Failure      400            {object}     apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}     apierror.APIError  "Item not found"
// @Failure      409            {object}     apierror.APIError  "Version mismatch or invalid transition or unassigned"
// @Router       /v1/reservations/{id}/items/{item_id}/checkin [patch]
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
// CheckoutReservation handles PATCH /{id}/checkout.
//
// @Summary      Check out reservation
// @Description  Batch check-out all checked-in items on a reservation.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header    string       true  "Property UUID"
// @Param        id             path      string       true  "Reservation UUID"
// @Success      200            {object}  BatchResult
// @Success      207            {object}  BatchResult  "Partial success"
// @Failure      400            {object}  apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}  apierror.APIError  "Reservation not found"
// @Router       /v1/reservations/{id}/checkout [patch]
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
// CheckoutItem handles PATCH /{id}/items/{item_id}/checkout.
//
// @Summary      Check out single item
// @Description  Check out a single reservation item.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header       string          true  "Property UUID"
// @Param        id             path         string          true  "Reservation UUID"
// @Param        item_id        path         string          true  "Item UUID"
// @Param        If-Match       header       string          true  "Version for optimistic concurrency"
// @Success      200            {object}     ItemResponse
// @Failure      400            {object}     apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}     apierror.APIError  "Item not found"
// @Failure      409            {object}     apierror.APIError  "Version mismatch or invalid transition"
// @Router       /v1/reservations/{id}/items/{item_id}/checkout [patch]
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
// MarkNoShow handles PATCH /{id}/items/{item_id}/no-show.
//
// @Summary      Mark item as no-show
// @Description  Mark a reservation item as no-show. Must be on or after arrival date.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header       string          true  "Property UUID"
// @Param        id             path         string          true  "Reservation UUID"
// @Param        item_id        path         string          true  "Item UUID"
// @Param        If-Match       header       string          true  "Version for optimistic concurrency"
// @Success      200            {object}     ItemResponse
// @Failure      400            {object}     apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}     apierror.APIError  "Item not found"
// @Failure      409            {object}     apierror.APIError  "Version mismatch or before arrival"
// @Router       /v1/reservations/{id}/items/{item_id}/no-show [patch]
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
// CancelItem handles POST /{id}/items/{item_id}/cancel.
//
// @Summary      Cancel single item
// @Description  Cancel an individual reservation item.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header       string          true  "Property UUID"
// @Param        id             path         string          true  "Reservation UUID"
// @Param        item_id        path         string          true  "Item UUID"
// @Param        If-Match       header       string          true  "Version for optimistic concurrency"
// @Param        body           body         CancelInput     true  "Cancellation details"
// @Success      200            {object}     ItemResponse
// @Failure      400            {object}     apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}     apierror.APIError  "Item not found"
// @Failure      409            {object}     apierror.APIError  "Version mismatch or invalid transition"
// @Failure      422            {object}     apierror.APIError  "Validation failed"
// @Router       /v1/reservations/{id}/items/{item_id}/cancel [post]
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
