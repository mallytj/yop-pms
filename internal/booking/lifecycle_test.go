package booking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexxcode1/yop-pms/internal/platform/types"
)

// R-RES-CRUD-018: POST /confirm — hold→confirmed staff path via HTTP.
func TestHandler_Confirm_OK(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/reservations/%s/confirm", res.ID), nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:confirm")
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
	if got.Status != StatusConfirmed {
		t.Errorf("status = %s, want confirmed", got.Status)
	}
}

// R-RES-CRUD-005: Cancel endpoint exists; 404 for unknown id.
func TestHandler_Cancel_NotFound(t *testing.T) {
	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPost, "/reservations/00000000-0000-0000-0000-000000000001/cancel", body)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:cancel")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rr.Code, rr.Body.String())
	}
}

// R-RES-CRUD-006: Reactivate endpoint exists; 404 for unknown id.
func TestHandler_Reactivate_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/reservations/00000000-0000-0000-0000-000000000001/reactivate", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:reactivate")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_CheckinReservation_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/reservations/00000000-0000-0000-0000-000000000001/checkin", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:checkin")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	// Returns 200 with empty batch result — no matching items is not an error
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_CheckoutReservation_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/reservations/00000000-0000-0000-0000-000000000001/checkout", nil)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:checkout")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	// Returns 200 with empty batch result — no matching items is not an error
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_CancelItem_NotFound(t *testing.T) {
	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPost, "/reservations/00000000-0000-0000-0000-000000000001/items/00000000-0000-0000-0000-000000000001/cancel", body)
	req.Header.Set("X-Property-ID", testPropertyID.String())
	req.Header.Set("X-User-Permissions", "reservations:cancel")
	req.Header.Set("If-Match", "1")

	rr := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rr.Code, rr.Body.String())
	}
}
