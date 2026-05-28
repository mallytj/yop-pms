package booking

// Core Requirements: R-RES-CRUD-001, R-RES-CRUD-002, R-RES-CRUD-003, R-RES-CRUD-010

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	httputil "github.com/lexxcode1/yop-pms/internal/platform/httputil"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
	"github.com/lexxcode1/yop-pms/internal/platform/validation"
)

// Create handles POST /reservations.
//
// @Summary      Create reservation
// @Description  Create a new reservation with one or more room items.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header    string                  true  "Property UUID"
// @Param        body           body      CreateReservationInput  true  "Reservation payload"
// @Success      201            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid request body or X-Property-ID"
// @Failure      409            {object}  apierror.APIError  "Resource conflict (e.g. duplicate, version mismatch)"
// @Failure      422            {object}  apierror.APIError  "Validation failed"
// @Failure      501            {object}  apierror.APIError  "Feature not implemented"
// @Router       /v1/reservations [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateReservationInput
	if apiErr := platformjson.ReadJSON(r, &input); apiErr != nil {
		platformjson.WriteError(w, r, apiErr)
		return
	}
	if errs := validation.Struct(input, "operations.reservations"); len(errs) > 0 {
		platformjson.WriteError(w, r, apierror.ErrUnprocessable.WithMessage(errs[0].Error()))
		return
	}

	include := ParseIncludeFlags(r)
	res, err := h.svc.CreateReservation(r.Context(), &input, include)
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}

	platformjson.WriteJSON(w, http.StatusCreated, res)
}

// Get handles GET /reservations/{id}.
//
// @Summary      Get reservation
// @Description  Fetch a single reservation by ID.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header    string  true  "Property UUID"
// @Param        id             path      string  true  "Reservation UUID"
// @Param        include        query     string  false "Comma-separated: items,guest,none"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or X-Property-ID"
// @Failure      404            {object}  apierror.APIError  "Reservation not found"
// @Router       /v1/reservations/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUIDParam(r, "id")
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	include := ParseIncludeFlags(r)
	res, svcErr := h.svc.GetReservation(r.Context(), id, include)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}

// List handles GET /reservations.
//
// @Summary      List reservations
// @Description  Cursor-paginated list of reservations for the property.
// @Tags         Reservations
// @Produce      json
// @Param        X-Property-ID  header  string  true  "Property UUID"
// @Param        status         query   string  false "Filter by status"
// @Param        limit          query   int     false "Page size (default 50)"
// @Success      200            {array}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid X-Property-ID or query params"
// @Router       /v1/reservations [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := ListParams{}

	if s := q.Get("status"); s != "" {
		params.Status = &s
	}

	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			params.Limit = int32(n)
		}
	}

	if raw := q.Get("cursor_date"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			params.CursorDate = &t
		}
	}

	if raw := q.Get("cursor_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			params.CursorID = &id
		}
	}

	result, err := h.svc.ListReservations(r.Context(), params)
	if err != nil {
		platformjson.WriteError(w, r, err)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, result)
}

// UpdateMetadata handles PATCH /reservations/{id}.
//
// @Summary      Update reservation metadata
// @Description  Partial update of reservation-level fields. Requires If-Match version header.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header    string               true  "Property UUID"
// @Param        If-Match       header    string               true  "Optimistic lock version"
// @Param        id             path      string               true  "Reservation UUID"
// @Param        body           body      UpdateMetadataInput  true  "Fields to update"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or request body"
// @Failure      412            {object}  apierror.APIError  "Version mismatch (If-Match)"
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

	// PATCH semantics: optional pointer fields. Only validate what we can.
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
//
// @Summary      Add item to reservation
// @Description  Adds a new room item to an existing reservation. Implemented in Phase 7.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header    string       true  "Property UUID"
// @Param        If-Match       header    string       true  "Optimistic lock version"
// @Param        id             path      string       true  "Reservation UUID"
// @Param        body           body      AddItemInput true  "Item to add"
// @Success      201            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or request body"
// @Failure      422            {object}  apierror.APIError  "Validation failed"
// @Failure      501            {object}  apierror.APIError  "Feature not implemented"
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
//
// @Summary      Update reservation item
// @Description  Updates an item's stay period or room type. Implemented in Phase 7.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header    string          true  "Property UUID"
// @Param        If-Match       header    string          true  "Optimistic lock version"
// @Param        id             path      string          true  "Reservation UUID"
// @Param        item_id        path      string          true  "Item UUID"
// @Param        body           body      CreateItemInput true  "Item fields"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or request body"
// @Failure      412            {object}  apierror.APIError  "Version mismatch (If-Match)"
// @Failure      501            {object}  apierror.APIError  "Feature not implemented"
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
//
// @Summary      Assign room to item
// @Description  Assigns a specific room to a reservation item. Implemented in Phase 7.
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        X-Property-ID  header    string          true  "Property UUID"
// @Param        If-Match       header    string          true  "Optimistic lock version"
// @Param        id             path      string          true  "Reservation UUID"
// @Param        item_id        path      string          true  "Item UUID"
// @Param        body           body      AssignRoomInput true  "Room assignment"
// @Success      200            {object}  ReservationResponse
// @Failure      400            {object}  apierror.APIError  "Invalid ID or request body"
// @Failure      412            {object}  apierror.APIError  "Version mismatch (If-Match)"
// @Failure      501            {object}  apierror.APIError  "Feature not implemented"
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
	res, svcErr := h.svc.AssignRoom(r.Context(), itemID, input)
	if svcErr != nil {
		platformjson.WriteError(w, r, svcErr)
		return
	}
	platformjson.WriteJSON(w, http.StatusOK, res)
}
