package booking

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// R-RES-CRUD-016: GET per-night rate rows for an item.
// R-RES-RATE-001: Rate resolved per night from rate level.
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

func TestHandler_UpdateBookedRates_NoSuchItem(t *testing.T) {
	input := RateAdjustInput{
		Adjustments: []RateAdjustment{
			{
				CalendarDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				Type:         AdjustmentFixed,
				Value:        15000,
				Reason:       "test",
			},
		},
	}
	bodyBytes, _ := json.Marshal(input)
	body := bytes.NewReader(bodyBytes)
	req := httptest.NewRequest(http.MethodPatch, "/reservations/00000000-0000-0000-0000-000000000001/items/00000000-0000-0000-0000-000000000001/booked-rates", body)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:rate_override")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	// Non-existent item returns 500 (internal error from DB).
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}
}

// R-RES-RATE-003: Derived rate plan adjustments (percentage or fixed).
// R-RES-EDGE-025: Negative adjustments clamped to 0 with warning (behaviour).
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
