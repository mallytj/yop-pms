package booking

// Miscellaneous handlers: availability surface, folio stub, cancellation quote stub.

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	httputil "github.com/lexxcode1/yop-pms/internal/platform/httputil"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
)

// Availability handles GET /reservations/availability.
//
// @Summary      Check room type availability
// @Description  Returns per-night availability for a room type. Results cached in Redis (TTL 60s).
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true   "Property UUID"
// @Param        room_type_id   query   string  true   "Room type UUID"
// @Param        start_date     query   string  true   "Start date (YYYY-MM-DD)"
// @Param        end_date       query   string  true   "End date (YYYY-MM-DD)"
// @Success      200            {array}  DateAvailability
// @Failure      400            {object}  apierror.APIError  "Invalid query params"
// @Router       /v1/reservations/availability [get]
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

// TODO: add /folios for all and the ability to get mutliple in one request

// GetFolio handles GET /reservations/{id}/folios/{folio_id}.
//
// @Summary      Get folio
// @Description  Returns folio data. Stub — finance PR implements this.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        id             path    string  true  "Reservation UUID"
// @Param        folio_id       path    string  true  "Folio UUID"
// @Success      200            {object}  map[string]string
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
// @Failure      501            {object}  apierror.APIError  "Feature not implemented"
// @Router       /v1/reservations/{id}/folios/{folio_id} [get]
func (h *Handler) GetFolio(w http.ResponseWriter, _ *http.Request) {
	platformjson.WriteJSON(w, http.StatusNotImplemented, map[string]string{"status": "not_implemented"})
}

// CancellationQuote handles GET /reservations/{id}/cancellation-quote.
//
// @Summary      Get cancellation quote
// @Description  Returns cancellation fee estimate. Stub — finance PR implements this.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        id             path    string  true  "Reservation UUID"
// @Success      200            {object}  CancellationQuoteResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID"
// @Failure      501            {object}  apierror.APIError  "Feature not implemented"
// @Router       /v1/reservations/{id}/cancellation-quote [get]
func (h *Handler) CancellationQuote(w http.ResponseWriter, _ *http.Request) {
	platformjson.WriteJSON(w, http.StatusNotImplemented, CancellationQuoteResponse{
		FeePence: nil,
		Status:   "not_implemented",
	})
}
