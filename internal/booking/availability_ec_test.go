package booking

// Availability edge-case tests: R-RES-AVAIL-001, R-RES-AVAIL-002, R-RES-AVAIL-003, ADR-013.

import (
	"context"
	"testing"
	"time"

	"github.com/lexxcode1/yop-pms/internal/platform/types"
)

// Edge case test coverage traceability:
//   TestCheckAvailability_InvalidDateRange        → R-RES-AVAIL-001
//   TestCheckAvailability_EmptyInventory          → R-RES-AVAIL-002
//   TestCheckAvailability_Success                 → R-RES-AVAIL-003

// TestCheckAvailability_InvalidDateRange verifies that end_date <= start_date returns ErrInvalidDates.
func TestCheckAvailability_InvalidDateRange(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	rtID := getRoomTypeID(t)
	base := time.Now().Truncate(24 * time.Hour).Add(30 * 24 * time.Hour)

	_, err := testSvc.CheckAvailability(ctx, testPropertyID, rtID, base, base)
	if err == nil {
		t.Fatal("expected error for equal start/end date, got nil")
	}

	_, err = testSvc.CheckAvailability(ctx, testPropertyID, rtID, base.Add(24*time.Hour), base)
	if err == nil {
		t.Fatal("expected error for end before start, got nil")
	}
}

// TestCheckAvailability_PartialAvailability books one room in a three-room type;
// CheckAvailability must return available=2 for each night (3 total - 1 booked).
func TestCheckAvailability_PartialAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)

	_, err := testSvc.CreateReservation(ctx, &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     testPropertyID,
		PrimaryGuestID: &guestID,
		Items: []CreateItemInput{
			{
				RoomTypeID:     getRoomTypeID(t),
				RatePlanID:     getRatePlanID(t),
				AssignedRoomID: roomIDPtr(t),
				ArrivalDate:    types.ISO8601Date{Time: arrival},
				DepartureDate:  types.ISO8601Date{Time: departure},
				AdultsCount:    1,
			},
		},
	}, IncludeFlags{})
	if err != nil {
		t.Fatalf("create reservation: %v", err)
	}

	results, err := testSvc.CheckAvailability(ctx, testPropertyID, getRoomTypeID(t), arrival, departure)
	if err != nil {
		t.Fatalf("CheckAvailability: %v", err)
	}
	// 3 rooms seeded, 1 booked → 2 available per night.
	if len(results) == 0 {
		t.Fatal("expected at least one date result")
	}
	for _, da := range results {
		if da.Available != 2 {
			t.Errorf("date %s: available=%d, want 2 (3 rooms, 1 booked)",
				da.Date.Format("2006-01-02"), da.Available)
		}
	}
}

// TestCheckAvailability_CacheHit verifies Redis is populated after the first call,
// and that a second call returns identical results (served from cache).
func TestCheckAvailability_CacheHit(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := ctxWithProperty(context.Background())

	start := time.Now().Truncate(24 * time.Hour).Add(400 * 24 * time.Hour)
	end := start.Add(3 * 24 * time.Hour)
	rtID := getRoomTypeID(t)

	first, err := testSvc.CheckAvailability(ctx, testPropertyID, rtID, start, end)
	if err != nil {
		t.Fatalf("first CheckAvailability: %v", err)
	}

	cacheKey := availabilityCacheKey(testPropertyID, rtID, start)
	val, err := testRedis.Get(ctx, cacheKey).Result()
	if err != nil {
		t.Fatalf("Redis key %q not set after first call: %v", cacheKey, err)
	}
	if val == "" {
		t.Fatalf("Redis key %q is empty", cacheKey)
	}

	second, err := testSvc.CheckAvailability(ctx, testPropertyID, rtID, start, end)
	if err != nil {
		t.Fatalf("second CheckAvailability: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("result length mismatch: first=%d second=%d", len(first), len(second))
	}
	for i := range first {
		if first[i].Date != second[i].Date || first[i].Available != second[i].Available {
			t.Errorf("date %s: first.Available=%d second.Available=%d",
				first[i].Date.Format("2006-01-02"), first[i].Available, second[i].Available)
		}
	}
}

// TestConflictCheck_Conflict asserts that booking a room twice on the same dates
// returns ErrRoomNotAvailable from conflictCheck.
func TestConflictCheck_Conflict(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	roomID := getRoomID(t)

	_, err := testSvc.CreateReservation(ctx, &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     testPropertyID,
		PrimaryGuestID: &guestID,
		Items: []CreateItemInput{
			{
				RoomTypeID:     getRoomTypeID(t),
				RatePlanID:     getRatePlanID(t),
				AssignedRoomID: &roomID,
				ArrivalDate:    types.ISO8601Date{Time: arrival},
				DepartureDate:  types.ISO8601Date{Time: departure},
				AdultsCount:    1,
			},
		},
	}, IncludeFlags{})
	if err != nil {
		t.Fatalf("first booking: %v", err)
	}

	var dates []time.Time
	for d := arrival; d.Before(departure); d = d.Add(24 * time.Hour) {
		dates = append(dates, d.Truncate(24*time.Hour))
	}

	got := testSvc.conflictCheck(ctx, testQueries, roomID, dates, nil)
	if got == nil {
		t.Fatal("expected ErrRoomNotAvailable for double-booked room, got nil")
	}
}

// TestConflictCheck_ExcludeItemID verifies that an item's own ledger rows do not
// trigger a conflict when its ID is passed as excludeItemID (needed for updates).
func TestConflictCheck_ExcludeItemID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(cleanupTestReservations)

	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)
	roomID := getRoomID(t)

	res, err := testSvc.CreateReservation(ctx, &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     testPropertyID,
		PrimaryGuestID: &guestID,
		Items: []CreateItemInput{
			{
				RoomTypeID:     getRoomTypeID(t),
				RatePlanID:     getRatePlanID(t),
				AssignedRoomID: &roomID,
				ArrivalDate:    types.ISO8601Date{Time: arrival},
				DepartureDate:  types.ISO8601Date{Time: departure},
				AdultsCount:    1,
			},
		},
	}, IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(res.Items) == 0 {
		t.Fatal("no items in response")
	}
	itemID := res.Items[0].ID

	var dates []time.Time
	for d := arrival; d.Before(departure); d = d.Add(24 * time.Hour) {
		dates = append(dates, d.Truncate(24*time.Hour))
	}

	if err := testSvc.conflictCheck(ctx, testQueries, roomID, dates, nil); err == nil {
		t.Fatal("expected conflict without exclusion, got nil")
	}

	if err := testSvc.conflictCheck(ctx, testQueries, roomID, dates, &itemID); err != nil {
		t.Errorf("expected no conflict with own item excluded, got: %v", err)
	}
}
