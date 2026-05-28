package booking

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_GetBookedRates_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/reservations/00000000-0000-0000-0000-000000000001/items/00000000-0000-0000-0000-000000000001/booked-rates", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:read")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	// Returns 200 with null body (empty result slice marshals to null in JSON)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_UpdateBookedRates_NotImplemented(t *testing.T) {
	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPatch, "/reservations/00000000-0000-0000-0000-000000000001/items/00000000-0000-0000-0000-000000000001/booked-rates", body)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:rate_override")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_AdjustRate_OK(t *testing.T) {
	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPost, "/reservations/00000000-0000-0000-0000-000000000001/items/00000000-0000-0000-0000-000000000001/adjust-rate", body)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:rate_override")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	// AdjustRate with empty adjustments succeeds
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_ApproveAdjustments_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/reservations/00000000-0000-0000-0000-000000000001/items/00000000-0000-0000-0000-000000000001/booked-rates/approve", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:rate_override")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	// Approve with no pending adjustments returns 200
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}
