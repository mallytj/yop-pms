package booking

// Core Requirements: R-RES-CRUD-003, R-RES-CRUD-010, R-RES-CRUD-011, R-RES-CRUD-012,
// R-RES-RATE-001, R-RES-RATE-002, R-RES-RATE-003, ADR-021

// update.go — Reservation mutation and miscellaneous handlers:
//   - HTTP handlers: UpdateMetadata, AddItem, UpdateItem, AssignRoom,
//     Availability

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	httputil "github.com/lexxcode1/yop-pms/internal/platform/httputil"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
	"github.com/lexxcode1/yop-pms/internal/platform/validation"

	_ "github.com/lexxcode1/yop-pms/internal/store" // swag: resolve PricingBookedDailyRate
)

// UpdateMetadata handles PATCH /reservations/{id}.
// UpdateMetadata handles PATCH /{id}.
//
// @Summary      Update reservation metadata
// @Description  Patch reservation-level fields: notes, travel_agent_id, group_id, primary_guest_id.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header               string               true  "Property UUID"
// @Param        id             path                 string               true  "Reservation UUID"
// @Param        If-Match       header               string               true  "Version for optimistic concurrency"
// @Param        body           body                 UpdateMetadataInput  true  "Fields to update"
// @Success      200            {object}             ReservationResponse
// @Failure      400            {object}             apierror.APIError    "Invalid ID or X-Property-ID"
// @Failure      404            {object}             apierror.APIError    "Reservation not found"
// @Failure      409            {object}             apierror.APIError    "Version mismatch"
// @Failure      422            {object}             apierror.APIError    "Validation failed"
// @Router       /v1/reservations/{id} [patch]
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
// AddItem handles POST /{id}/items.
//
// @Summary      Add item to reservation
// @Description  Add a new room item to an existing reservation.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header    string        true  "Property UUID"
// @Param        id             path      string        true  "Reservation UUID"
// @Param        If-Match       header    string        true  "Version for optimistic concurrency"
// @Param        body           body      AddItemInput  true  "Item payload"
// @Success      201            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}  apierror.APIError  "Reservation not found"
// @Failure      409            {object}  apierror.APIError  "Version mismatch or terminal state"
// @Failure      422            {object}  apierror.APIError  "Validation failed"
// @Router       /v1/reservations/{id}/items [post]
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
// UpdateItem handles PATCH /{id}/items/{item_id}.
//
// @Summary      Update reservation item
// @Description  Update stay period, room type, rate plan, guest counts of a reservation item.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header             string            true  "Property UUID"
// @Param        id             path               string            true  "Reservation UUID"
// @Param        item_id        path               string            true  "Item UUID"
// @Param        If-Match       header             string            true  "Version for optimistic concurrency"
// @Param        body           body               CreateItemInput   true  "Updated item fields"
// @Success      200            {object}           ReservationResponse
// @Failure      400            {object}           apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}           apierror.APIError  "Item not found"
// @Failure      409            {object}           apierror.APIError  "Version mismatch"
// @Failure      422            {object}           apierror.APIError  "Validation failed"
// @Router       /v1/reservations/{id}/items/{item_id} [patch]
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
// AssignRoom handles PATCH /{id}/items/{item_id}/assign-room.
//
// @Summary      Assign room to item
// @Description  Assign or reassign a physical room to a reservation item.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header             string            true  "Property UUID"
// @Param        id             path               string            true  "Reservation UUID"
// @Param        item_id        path               string            true  "Item UUID"
// @Param        If-Match       header             string            true  "Version for optimistic concurrency"
// @Param        body           body               AssignRoomInput   true  "Room assignment payload"
// @Success      200            {object}           ItemResponse
// @Failure      400            {object}           apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}           apierror.APIError  "Item not found"
// @Failure      409            {object}           apierror.APIError  "Version mismatch or DNM conflict"
// @Failure      422            {object}           apierror.APIError  "Validation failed"
// @Router       /v1/reservations/{id}/items/{item_id}/assign-room [patch]
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
	if errs := validation.Struct(input, "operations.reservation_items"); len(errs) > 0 {
		platformjson.WriteError(w, r, apierror.ErrUnprocessable.WithMessage(errs[0].Error()))
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
// HTTP handlers: availability
// ─────────────────────────────────────────────────────────────────────────────

// Availability handles GET /reservations/availability.
// Availability handles GET /availability.
//
// @Summary      Check room type availability
// @Description  Check date-range availability for a room type.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header    string  true  "Property UUID"
// @Param        room_type_id   query     string  true  "Room type UUID"
// @Param        start_date     query     string  true  "Start date (YYYY-MM-DD)"
// @Param        end_date       query     string  true  "End date (YYYY-MM-DD)"
// @Success      200            {array}   DateAvailability
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
