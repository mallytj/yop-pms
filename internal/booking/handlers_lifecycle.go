package booking

// Core Requirements: R-RES-CRUD-004, R-RES-CRUD-005, R-RES-CRUD-006, R-RES-LIFECYCLE-001

import (
	"net/http"

	// Blank import for swagger
	_ "github.com/lexxcode1/yop-pms/internal/platform/apierror"
	httputil "github.com/lexxcode1/yop-pms/internal/platform/httputil"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
)

// Confirm handles POST /reservations/{id}/confirm.
//
// @Summary      Confirm reservation
// @Description  Transitions a hold reservation to confirmed status.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        If-Match       header  string  true  "Optimistic lock version"
// @Param        id             path    string  true  "Reservation UUID"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      409            {object}  apierror.APIError  "Invalid state transition"
// @Failure      412            {object}  apierror.APIError  "Version mismatch (If-Match)"
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
//
// @Summary      Cancel reservation
// @Description  Cancels a reservation with reason code and optional fee.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header  string      true  "Property UUID"
// @Param        If-Match       header  string      true  "Optimistic lock version"
// @Param        id             path    string      true  "Reservation UUID"
// @Param        body           body    CancelInput false "Cancellation options"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or request body"
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
	res, svcErr := h.svc.CancelReservation(r.Context(), id, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// Reactivate handles POST /reservations/{id}/reactivate.
//
// @Summary      Reactivate reservation
// @Description  Reactivates a cancelled reservation back to confirmed.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        If-Match       header  string  true  "Optimistic lock version"
// @Param        id             path    string  true  "Reservation UUID"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
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
//
// @Summary      Check in all items
// @Description  Checks in all items on a reservation. Returns 207 if partial.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        id             path    string  true  "Reservation UUID"
// @Success      200            {object}  BatchResult
// @Success      207            {object}  BatchResult
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
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
//
// @Summary      Check in single item
// @Description  Checks in a single reservation item.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        If-Match       header  string  true  "Optimistic lock version"
// @Param        id             path    string  true  "Reservation UUID"
// @Param        item_id        path    string  true  "Item UUID"
// @Success      200            {object}  ItemResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
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
//
// @Summary      Check out all items
// @Description  Checks out all items on a reservation. Returns 207 if partial.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        id             path    string  true  "Reservation UUID"
// @Success      200            {object}  BatchResult
// @Success      207            {object}  BatchResult
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
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
//
// @Summary      Check out single item
// @Description  Checks out a single reservation item.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        If-Match       header  string  true  "Optimistic lock version"
// @Param        id             path    string  true  "Reservation UUID"
// @Param        item_id        path    string  true  "Item UUID"
// @Success      200            {object}  ItemResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
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
//
// @Summary      Mark item as no-show
// @Description  Marks a reservation item as no-show. Requires stay period to have started.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        If-Match       header  string  true  "Optimistic lock version"
// @Param        id             path    string  true  "Reservation UUID"
// @Param        item_id        path    string  true  "Item UUID"
// @Success      200            {object}  ItemResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
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
//
// @Summary      Cancel single item
// @Description  Cancels a single reservation item with reason code and optional fee.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header  string      true  "Property UUID"
// @Param        If-Match       header  string      true  "Optimistic lock version"
// @Param        id             path    string      true  "Reservation UUID"
// @Param        item_id        path    string      true  "Item UUID"
// @Param        body           body    CancelInput false "Cancellation options"
// @Success      200            {object}  ItemResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or request body"
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
