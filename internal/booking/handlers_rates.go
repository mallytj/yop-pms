package booking

// Core Requirements: R-RES-RATE-001, R-RES-RATE-002, R-RES-RATE-003

import (
	"net/http"

	httputil "github.com/lexxcode1/yop-pms/internal/platform/httputil"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
	// Import for swagger
	_ "github.com/lexxcode1/yop-pms/internal/store"
	// Import for swagger
	_ "github.com/lexxcode1/yop-pms/internal/platform/apierror"
)

// GetBookedRates handles GET /reservations/{id}/items/{item_id}/booked-rates.
//
// @Summary      Get booked daily rates
// @Description  Returns booked daily rates for a reservation item.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        id             path    string  true  "Reservation UUID"
// @Param        item_id        path    string  true  "Item UUID"
// @Success      200            {array}   store.PricingBookedDailyRate
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
// @Router       /v1/reservations/{id}/items/{item_id}/booked-rates [get]
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
//
// @Summary      Update booked daily rates
// @Description  Overwrites booked daily rates for an item. Not yet implemented — use AdjustRate instead.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header    string          true  "Property UUID"
// @Param        If-Match       header    string          true  "Optimistic lock version"
// @Param        id             path      string          true  "Reservation UUID"
// @Param        item_id        path      string          true  "Item UUID"
// @Param        body           body      RateAdjustInput true  "Rate adjustments"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or request body"
// @Failure      412            {object}  apierror.APIError  "Version mismatch (If-Match)"
// @Failure      501            {object}  apierror.APIError  "Feature not implemented"
// @Router       /v1/reservations/{id}/items/{item_id}/booked-rates [patch]
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
//
// @Summary      Adjust daily rates
// @Description  Applies percentage or fixed adjustments to booked rates.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header    string          true  "Property UUID"
// @Param        If-Match       header    string          true  "Optimistic lock version"
// @Param        id             path      string          true  "Reservation UUID"
// @Param        item_id        path      string          true  "Item UUID"
// @Param        body           body      RateAdjustInput true  "Rate adjustments"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or request body"
// @Failure      412            {object}  apierror.APIError  "Version mismatch (If-Match)"
// @Router       /v1/reservations/{id}/items/{item_id}/adjust-rate [post]
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
//
// @Summary      Approve rate adjustments
// @Description  Approves pending rate adjustments for a reservation item.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        If-Match       header  string  true  "Optimistic lock version"
// @Param        id             path    string  true  "Reservation UUID"
// @Param        item_id        path    string  true  "Item UUID"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
// @Failure      412            {object}  apierror.APIError  "Version mismatch (If-Match)"
// @Router       /v1/reservations/{id}/items/{item_id}/booked-rates/approve [post]
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
