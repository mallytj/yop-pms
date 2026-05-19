package validation

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// validReservation returns a test reservation with every required field populated.
func validReservation() testReservation {
	return testReservation{
		Source:     "website",
		PropertyID: uuid.New(),
		Status:     "hold",
		Version:    1,
	}
}

type testReservation struct {
	Source     string    `json:"source"`
	PropertyID uuid.UUID `json:"property_id"`
	Notes      string    `json:"notes"`
	Status     string    `json:"status"`
	Version    int       `json:"version"`
}

// validItem returns a test item with every required field populated.
func validItem() testItem {
	return testItem{
		BookedRoomTypeID: uuid.New(),
		AdultsCount:      2,
		ChildrenCount:    0,
		StayPeriod:       "[2026-06-01,2026-06-04)",
		Status:           "booked",
		BaseRatePence:    10000,
		ReservationID:    uuid.New().String(),
	}
}

type testItem struct {
	BookedRoomTypeID uuid.UUID `json:"booked_room_type_id"`
	AdultsCount      int       `json:"adults_count"`
	ChildrenCount    int       `json:"children_count"`
	StayPeriod       string    `json:"stay_period"`
	Status           string    `json:"status"`
	BaseRatePence    int       `json:"base_rate_pence"`
	ReservationID    string    `json:"reservation_id"`
}

type testCreateInput struct {
	Source     string     `json:"source"`
	PropertyID uuid.UUID  `json:"property_id"`
	Notes      string     `json:"notes"`
	Status     string     `json:"status"`
	Version    int        `json:"version"`
	Items      []testItem `json:"items" constraints:"operations.reservation_items"`
}

type testBookedRate struct {
	CalendarDate       string `json:"calendar_date"`
	BasePricePence     int    `json:"base_price_pence"`
	FinalPricePence    int    `json:"final_price_pence"`
	ReservationItemID  string `json:"reservation_item_id"`
	AdjustmentApproved bool   `json:"adjustment_approved"`
	Type               string `json:"type"`   // maps to adjustment.jsonb.type
	Value              int    `json:"value"`  // maps to adjustment.jsonb.value
	Reason             string `json:"reason"` // maps to adjustment.jsonb.reason
}

func TestStruct_RequiredString(t *testing.T) {
	input := validReservation()
	input.Source = ""
	errs := Struct(input, "operations.reservations")
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "source")
	require.Contains(t, errs[0].Error(), "required")
}

func TestStruct_RequiredUUID(t *testing.T) {
	input := validReservation()
	input.PropertyID = uuid.Nil
	errs := Struct(input, "operations.reservations")
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "property_id")
}

func TestStruct_MaxLength(t *testing.T) {
	input := validReservation()
	input.Notes = string(make([]byte, 3000))
	errs := Struct(input, "operations.reservations")
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "notes")
	require.Contains(t, errs[0].Error(), "2500")
}

func TestStruct_MaxLengthOK(t *testing.T) {
	input := validReservation()
	input.Notes = string(make([]byte, 2500))
	errs := Struct(input, "operations.reservations")
	require.Nil(t, errs)
}

func TestStruct_IntMin(t *testing.T) {
	input := validItem()
	input.AdultsCount = 0
	errs := Struct(input, "operations.reservation_items")
	var found bool
	for _, e := range errs {
		if strings.Contains(e.Error(), "adults_count") && strings.Contains(e.Error(), "at least 1") {
			found = true
			break
		}
	}
	require.True(t, found, "expected adults_count min=1 error, got: %v", errs)
}

func TestStruct_IntMinOK(t *testing.T) {
	input := validItem()
	input.AdultsCount = 2
	errs := Struct(input, "operations.reservation_items")
	require.Nil(t, errs)
}

func TestStruct_Pattern(t *testing.T) {
	// auth.users has pattern ^[a-zA-Z0-9_]+$ on username
	type testUser struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	input := testUser{
		Username: "bad user!",
		Email:    "a@b.com",
		Role:     "staff",
	}
	errs := Struct(input, "auth.users")
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "username")
}

func TestStruct_PatternOK(t *testing.T) {
	type testUser struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	input := testUser{
		Username: "good_user_123",
		Email:    "a@b.com",
		Role:     "staff",
	}
	errs := Struct(input, "auth.users")
	require.Nil(t, errs)
}

func TestStruct_NestedSlice(t *testing.T) {
	good := validItem()
	bad := validItem()
	bad.AdultsCount = 0
	input := testCreateInput{
		Source:     "website",
		PropertyID: uuid.New(),
		Status:     "hold",
		Version:    1,
		Items:      []testItem{bad, good},
	}
	errs := Struct(input, "operations.reservations")
	var found bool
	for _, e := range errs {
		if strings.Contains(e.Error(), "items[0]") && strings.Contains(e.Error(), "adults_count") {
			found = true
			break
		}
	}
	require.True(t, found, "expected items[0] adults_count error, got: %v", errs)
}

func TestStruct_JSONBSubFields(t *testing.T) {
	// pricing.booked_daily_rates has adjustment.jsonb with type, value, reason all required
	input := testBookedRate{
		CalendarDate:       "2026-06-01",
		BasePricePence:     10000,
		FinalPricePence:    10000,
		ReservationItemID:  uuid.New().String(),
		AdjustmentApproved: true,
		Type:               "",
		Value:              0,
		Reason:             "",
	}
	errs := Struct(input, "pricing.booked_daily_rates")
	require.Len(t, errs, 3)
	for _, e := range errs {
		require.Contains(t, e.Error(), "required")
	}
}

func TestStruct_JSONBSubFieldsOK(t *testing.T) {
	input := testBookedRate{
		CalendarDate:       "2026-06-01",
		BasePricePence:     10000,
		FinalPricePence:    10000,
		ReservationItemID:  uuid.New().String(),
		AdjustmentApproved: true,
		Type:               "fixed",
		Value:              -5000,
		Reason:             "goodwill discount",
	}
	errs := Struct(input, "pricing.booked_daily_rates")
	require.Nil(t, errs)
}

func TestStruct_NoConstraints(t *testing.T) {
	errs := Struct(struct{ X string }{"hello"}, "nonexistent.table")
	require.Nil(t, errs)
}

func TestStruct_PointerField(t *testing.T) {
	type withPtr struct {
		Source     string     `json:"source"`
		PropertyID *uuid.UUID `json:"property_id"`
	}
	t.Run("nil pointer is required error", func(t *testing.T) {
		input := withPtr{Source: "website", PropertyID: nil}
		errs := Struct(input, "operations.reservations")
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), "property_id")
	})
	t.Run("non-nil pointer is ok", func(t *testing.T) {
		id := uuid.New()
		input := withPtr{Source: "website", PropertyID: &id}
		errs := Struct(input, "operations.reservations")
		require.Nil(t, errs)
	})
}

func TestStruct(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		v        any
		tableKey string
		want     []error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Struct(tt.v, tt.tableKey)
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("Struct() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isZero(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		fv   reflect.Value
		kind reflect.Kind
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isZero(tt.fv, tt.kind)
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("isZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
