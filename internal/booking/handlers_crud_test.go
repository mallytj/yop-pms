package booking

// HTTP handler integration tests for CRUD endpoints.
// Tests run against the real DB + Redis from TestMain (testcontainers).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexxcode1/yop-pms/internal/platform/types"

	"github.com/go-chi/chi/v5"
	yopMw "github.com/lexxcode1/yop-pms/internal/platform/middleware"
)

// newTestHandler builds a chi router with StubAuth + booking routes backed by testSvc.
// Mounted at /reservations (matching the production path segment).
func newTestHandler() http.Handler {
	r := chi.NewRouter()
	r.Use(yopMw.StubAuth)
	r.Route("/reservations", Routes(testSvc, yopMw.RequireIfMatch))
	return r
}

func TestHandler_Create_MissingPropertyID(t *testing.T) {
	body := []byte(`{"source":"internal","items":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/reservations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Property-ID header

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/reservations", bytes.NewReader([]byte(`not-json`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:create")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Create_MissingPermission(t *testing.T) {
	body := []byte(`{"source":"internal","items":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/reservations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	// No reservations:create permission

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Create_OK(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	rtID := getRoomTypeID(t)
	rpID := getRatePlanID(t)

	body := map[string]any{
		"source":           "internal",
		"property_id":      testPropertyID,
		"primary_guest_id": guestID,
		"items": []map[string]any{
			{
				"room_type_id":   rtID,
				"rate_plan_id":   rpID,
				"arrival_date":   arrival.Format("2006-01-02"),
				"departure_date": departure.Format("2006-01-02"),
				"adults_count":   1,
			},
		},
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/reservations?include=items", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:create")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rr.Code, rr.Body.String())
	}

	var resp ReservationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != StatusHold {
		t.Errorf("status = %s, want hold", resp.Status)
	}
	if resp.Code == "" {
		t.Error("code is empty")
	}
}

func TestHandler_Get_OK(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	// Create a reservation via service to get a known ID.
	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	res, err := testSvc.CreateReservation(
		ctxWithProperty(context.Background()),
		&CreateReservationInput{
			Source:         SourceInternal,
			PropertyID:     testPropertyID,
			PrimaryGuestID: &guestID,
			Items: []CreateItemInput{
				{
					RoomTypeID:    getRoomTypeID(t),
					RatePlanID:    getRatePlanID(t),
					ArrivalDate:   types.ISO8601Date{Time: arrival},
					DepartureDate: types.ISO8601Date{Time: departure},
					AdultsCount:   1,
				},
			},
		},
		IncludeFlags{},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/reservations/%s", res.ID), nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:read")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var got ReservationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != res.ID {
		t.Errorf("id = %s, want %s", got.ID, res.ID)
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/reservations/00000000-0000-0000-0000-000000000001", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:read")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_List_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/reservations", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:read")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	// Response must be a JSON array (possibly empty).
	var got []ReservationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("expected array response: %v; body: %s", err, rr.Body.String())
	}
}

func TestHandler_Confirm_VersionMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	// Create reservation (version=1).
	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	res, err := testSvc.CreateReservation(
		ctxWithProperty(context.Background()),
		&CreateReservationInput{
			Source:         SourceInternal,
			PropertyID:     testPropertyID,
			PrimaryGuestID: &guestID,
			Items: []CreateItemInput{
				{
					RoomTypeID:     getRoomTypeID(t),
					RatePlanID:     getRatePlanID(t),
					ArrivalDate:    types.ISO8601Date{Time: arrival},
					DepartureDate:  types.ISO8601Date{Time: departure},
					AssignedRoomID: roomIDPtr(t),
					AdultsCount:    1,
				},
			},
		},
		IncludeFlags{},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/reservations/%s/confirm", res.ID), nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:confirm")
	req.Header.Set("If-Match", "9999") // wrong version

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionFailed {
		t.Errorf("status = %d, want 412; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_UpdateMetadata_OK(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	res, err := testSvc.CreateReservation(
		ctxWithProperty(context.Background()),
		&CreateReservationInput{
			Source:         SourceInternal,
			PropertyID:     testPropertyID,
			PrimaryGuestID: &guestID,
			Items: []CreateItemInput{
				{
					RoomTypeID:    getRoomTypeID(t),
					RatePlanID:    getRatePlanID(t),
					ArrivalDate:   types.ISO8601Date{Time: arrival},
					DepartureDate: types.ISO8601Date{Time: departure},
					AdultsCount:   1,
				},
			},
		},
		IncludeFlags{},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	body := map[string]any{
		"notes": "Updated notes",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/reservations/%s", res.ID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:update")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var got ReservationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Notes != "Updated notes" {
		t.Errorf("notes = %q, want %q", got.Notes, "Updated notes")
	}
}

func TestHandler_UpdateMetadata_VersionMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	res, err := testSvc.CreateReservation(
		ctxWithProperty(context.Background()),
		&CreateReservationInput{
			Source:         SourceInternal,
			PropertyID:     testPropertyID,
			PrimaryGuestID: &guestID,
			Items: []CreateItemInput{
				{
					RoomTypeID:    getRoomTypeID(t),
					RatePlanID:    getRatePlanID(t),
					ArrivalDate:   types.ISO8601Date{Time: arrival},
					DepartureDate: types.ISO8601Date{Time: departure},
					AdultsCount:   1,
				},
			},
		},
		IncludeFlags{},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	body := map[string]any{"notes": "should fail"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/reservations/%s", res.ID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:update")
	req.Header.Set("If-Match", "9999") // wrong version

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionFailed {
		t.Errorf("status = %d, want 412; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_AddItem_NotImplemented(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	res, err := testSvc.CreateReservation(
		ctxWithProperty(context.Background()),
		&CreateReservationInput{
			Source:         SourceInternal,
			PropertyID:     testPropertyID,
			PrimaryGuestID: &guestID,
			Items: []CreateItemInput{
				{
					RoomTypeID:    getRoomTypeID(t),
					RatePlanID:    getRatePlanID(t),
					ArrivalDate:   types.ISO8601Date{Time: arrival},
					DepartureDate: types.ISO8601Date{Time: departure},
					AdultsCount:   1,
				},
			},
		},
		IncludeFlags{},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	body := map[string]any{
		"room_type_id":   getRoomTypeID(t),
		"arrival_date":   arrival.Format("2006-01-02"),
		"departure_date": departure.Format("2006-01-02"),
		"adults_count":   1,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/reservations/%s/items", res.ID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:add_item")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_UpdateItem_NotImplemented(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	res, err := testSvc.CreateReservation(
		ctxWithProperty(context.Background()),
		&CreateReservationInput{
			Source:         SourceInternal,
			PropertyID:     testPropertyID,
			PrimaryGuestID: &guestID,
			Items: []CreateItemInput{
				{
					RoomTypeID:    getRoomTypeID(t),
					RatePlanID:    getRatePlanID(t),
					ArrivalDate:   types.ISO8601Date{Time: arrival},
					DepartureDate: types.ISO8601Date{Time: departure},
					AdultsCount:   1,
				},
			},
		},
		IncludeFlags{Items: true},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	itemID := res.Items[0].ID
	body := map[string]any{
		"room_type_id":   getRoomTypeID(t),
		"arrival_date":   arrival.Format("2006-01-02"),
		"departure_date": departure.Format("2006-01-02"),
		"adults_count":   2,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/reservations/%s/items/%s", res.ID, itemID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:update_item")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_AssignRoom_NotImplemented(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	res, err := testSvc.CreateReservation(
		ctxWithProperty(context.Background()),
		&CreateReservationInput{
			Source:         SourceInternal,
			PropertyID:     testPropertyID,
			PrimaryGuestID: &guestID,
			Items: []CreateItemInput{
				{
					RoomTypeID:    getRoomTypeID(t),
					RatePlanID:    getRatePlanID(t),
					ArrivalDate:   types.ISO8601Date{Time: arrival},
					DepartureDate: types.ISO8601Date{Time: departure},
					AdultsCount:   1,
				},
			},
		},
		IncludeFlags{Items: true},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	itemID := res.Items[0].ID
	body := map[string]any{
		"room_id": getRoomID(t),
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/reservations/%s/items/%s/assign-room", res.ID, itemID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:assign_room")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501; body: %s", rr.Code, rr.Body.String())
	}
}
