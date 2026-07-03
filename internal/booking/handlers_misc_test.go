package booking

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// R-RES-AVAIL-001: Availability check endpoint.
// R-RES-AVAIL-007: Per-date remaining count returned.
func TestHandler_Availability_OK(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)

	rtID := getRoomTypeID(t)
	start := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	end := time.Now().Add(72 * time.Hour).Format("2006-01-02")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/reservations/availability?room_type_id=%s&start_date=%s&end_date=%s", rtID, start, end), nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var result []DateAvailability
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected at least one date in availability response")
	}
}

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

func TestHandler_GetFolio_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/reservations/00000000-0000-0000-0000-000000000001/folios/00000000-0000-0000-0000-000000000001", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:read")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body: %s", rr.Code, rr.Body.String())
	}

	var result map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "not_implemented" {
		t.Errorf("status = %q, want not_implemented", result["status"])
	}
}

func TestHandler_CancellationQuote_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/reservations/00000000-0000-0000-0000-000000000001/cancellation-quote", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:read")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body: %s", rr.Code, rr.Body.String())
	}

	var result CancellationQuoteResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Status != "not_implemented" {
		t.Errorf("status = %q, want not_implemented", result.Status)
	}
}
