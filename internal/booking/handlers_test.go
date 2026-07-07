package booking

// HTTP handler smoke tests — covers auth, middleware, routing, and error paths.
// Happy-path business logic is tested at the service layer (service_test.go et al).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	yopMw "github.com/lexxcode1/yop-pms/internal/platform/middleware"
	"github.com/lexxcode1/yop-pms/internal/platform/types"
)

func newTestHandler() http.Handler {
	r := chi.NewRouter()
	r.Use(yopMw.StubAuth)
	r.Route("/reservations", Routes(testSvc, yopMw.RequireIfMatch))
	return r
}

// ── Create ──────────────────────────────────────────────────────────────────

func TestHandler_Create_MissingPropertyID(t *testing.T) {
	body := []byte(`{"source":"internal","items":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/reservations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

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

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body: %s", rr.Code, rr.Body.String())
	}
}

// ── Read ────────────────────────────────────────────────────────────────────

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

// ── Version guard ───────────────────────────────────────────────────────────

func TestHandler_Confirm_VersionMismatch(t *testing.T) {
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
			Items: []CreateItemInput{{
				RoomTypeID:     getRoomTypeID(t),
				RatePlanID:     getRatePlanID(t),
				ArrivalDate:    types.ISO8601Date{Time: arrival},
				DepartureDate:  types.ISO8601Date{Time: departure},
				AssignedRoomID: roomIDPtr(t),
				AdultsCount:    1,
			}},
		},
		IncludeFlags{},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/reservations/%s/confirm", res.ID), nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:confirm")
	req.Header.Set("If-Match", "9999")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionFailed {
		t.Errorf("status = %d, want 412; body: %s", rr.Code, rr.Body.String())
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
			Items: []CreateItemInput{{
				RoomTypeID:    getRoomTypeID(t),
				RatePlanID:    getRatePlanID(t),
				ArrivalDate:   types.ISO8601Date{Time: arrival},
				DepartureDate: types.ISO8601Date{Time: departure},
				AdultsCount:   1,
			}},
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
	req.Header.Set("If-Match", "9999")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionFailed {
		t.Errorf("status = %d, want 412; body: %s", rr.Code, rr.Body.String())
	}
}

// ── Lifecycle error paths ──────────────────────────────────────────────────

func TestHandler_Cancel_NotFound(t *testing.T) {
	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/reservations/%s/cancel", uuid.New()), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:cancel")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Reactivate_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/reservations/%s/reactivate", uuid.New()), nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:reactivate")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_CancelItem_NotFound(t *testing.T) {
	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/reservations/%s/items/%s/cancel", uuid.New(), uuid.New()), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:cancel")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rr.Code, rr.Body.String())
	}
}

// ── Availability ────────────────────────────────────────────────────────────

func TestHandler_Availability_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/reservations/availability", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_Availability_InvalidRoomTypeID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/reservations/availability?room_type_id=bad-uuid&start_date=2026-06-01&end_date=2026-06-03", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
}
