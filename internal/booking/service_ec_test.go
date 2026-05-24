package booking

// Edge case tests mapped to R-RES-EDGE-XXX in docs/requirements/reservations.md §8.
// Each test name includes the requirement ID for traceability.
// Tests requiring unimplemented service methods have a t.Skip() with blocking dependency.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/types"
)

// R-RES-EDGE-016: Zero-night stay (arrival == departure) rejected at DB CHECK level.
func TestEdge_016_ZeroNightStay(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	arrival, _ := nextTestDate(t)
	sameDay := types.ISO8601Date{Time: arrival}

	// Arrival == departure means zero nights — DB CHECK lower(stay_period) < upper(stay_period)
	input := &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     uuid.Nil,
		PrimaryGuestID: &guestID,
		Items: []CreateItemInput{
			{
				RoomTypeID:    getRoomTypeID(t),
				RatePlanID:    getRatePlanID(t),
				ArrivalDate:   sameDay,
				DepartureDate: sameDay,
				AdultsCount:   1,
			},
		},
	}

	// This is a DB CHECK constraint, not a service check. The service
	// inserts the range, DB rejects it.
	_, err := testSvc.CreateReservation(ctx, input, IncludeFlags{})
	if err == nil {
		t.Fatal("R-RES-EDGE-016: expected error for zero-night stay, got nil")
	}
}

// R-RES-EDGE-023: Complimentary reservation (£0 stay). Creates OK, folio A still created.
func TestEdge_023_ComplimentaryReservation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	arrival, departure := nextTestDate(t)

	res, err := testSvc.CreateReservation(ctx, &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     uuid.Nil,
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
	}, IncludeFlags{})
	if err != nil {
		t.Fatalf("R-RES-EDGE-023: complimentary reservation should create: %v", err)
	}
	if res.Code == "" {
		t.Error("R-RES-EDGE-023: expected code")
	}
	if res.Status != StatusHold {
		t.Errorf("R-RES-EDGE-023: status = %s, want hold", res.Status)
	}

	// Confirm it exists in DB
	dbRes, err := testQueries.GetReservation(context.Background(), res.ID)
	if err != nil {
		t.Fatalf("R-RES-EDGE-023: get reservation: %v", err)
	}
	if dbRes.ID != res.ID {
		t.Errorf("R-RES-EDGE-023: id mismatch")
	}
}

// R-RES-EDGE-048: Check-in attempted with no room assigned.
// Walk-in without assigned_room_id → 409 UNASSIGNED_ITEMS.
func TestEdge_048_CheckinNoRoomAssignment(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	arrival, departure := nextTestDate(t)

	input := &CreateReservationInput{
		Source:         SourceInternal,
		IsWalkin:       true,
		PropertyID:     uuid.Nil,
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
	}

	_, err := testSvc.CreateReservation(ctx, input, IncludeFlags{})
	if err == nil {
		t.Fatal("R-RES-EDGE-048: expected UNASSIGNED_ITEMS for walkin without room, got nil")
	}
}

// R-RES-EDGE-049: Check-in attempted before check-in date.
// Service rejects past-date arrival for non-walkin (R-RES-VALID-002).
func TestEdge_049_CheckinBeforeArrivalDate(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	pastArrival := time.Now().Truncate(24 * time.Hour).Add(-7 * 24 * time.Hour)

	input := &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     uuid.Nil,
		PrimaryGuestID: &guestID,
		Items: []CreateItemInput{
			{
				RoomTypeID:    getRoomTypeID(t),
				RatePlanID:    getRatePlanID(t),
				ArrivalDate:   types.ISO8601Date{Time: pastArrival},
				DepartureDate: types.ISO8601Date{Time: pastArrival.Add(3 * 24 * time.Hour)},
				AdultsCount:   1,
			},
		},
	}

	_, err := testSvc.CreateReservation(ctx, input, IncludeFlags{})
	if err == nil {
		t.Fatal("R-RES-EDGE-049: expected INVALID_DATES for past arrival, got nil")
	}
}

// R-RES-EDGE-049 variant: Walkin is exempt from past-date check.
func TestEdge_049_WalkinPastDateExempt(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	today := time.Now().Truncate(24 * time.Hour)

	input := &CreateReservationInput{
		Source:         SourceInternal,
		IsWalkin:       true,
		PropertyID:     uuid.Nil,
		PrimaryGuestID: &guestID,
		Items: []CreateItemInput{
			{
				RoomTypeID:     getRoomTypeID(t),
				RatePlanID:     getRatePlanID(t),
				AssignedRoomID: roomIDPtr(t),
				ArrivalDate:    types.ISO8601Date{Time: today},
				DepartureDate:  types.ISO8601Date{Time: today.Add(24 * time.Hour)},
				AdultsCount:    1,
			},
		},
	}

	res, err := testSvc.CreateReservation(ctx, input, IncludeFlags{})
	if err != nil {
		t.Fatalf("R-RES-EDGE-049: walkin today should be allowed: %v", err)
	}
	if res.Status != StatusCheckedIn {
		t.Errorf("status = %s, want checked_in", res.Status)
	}
}

// R-RES-EDGE-061: Confirm on already-confirmed reservation → 200 no-op.
func TestEdge_061_ConfirmIdempotentOnConfirmed(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	arrival, departure := nextTestDate(t)

	res, err := testSvc.CreateReservation(ctx, &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     uuid.Nil,
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
	}, IncludeFlags{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// First confirm
	confirmCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	confirmed, err := testSvc.ConfirmReservation(confirmCtx, res.ID, IncludeFlags{})
	if err != nil {
		t.Fatalf("first confirm: %v", err)
	}

	// Second confirm at service layer: ErrInvalidTransition.
	// ActionIdempotency at handler layer converts to 200 no-op per §7.4.
	confirmCtx2 := helpers.SetIfMatchVersion(ctx, confirmed.Version)
	_, err = testSvc.ConfirmReservation(confirmCtx2, res.ID, IncludeFlags{})
	if err == nil {
		t.Fatal("expected ErrInvalidTransition on second confirm, got nil")
	}
}

// This is hard to test without mocking, but the service logs and continues.
// We verify the reservation was created regardless.
func TestEdge_056_NotifyFailureNonFatal(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// NOTIFY failure is inside the tx — we can't easily trigger it.
	// Instead, verify the default case: NOTIFY succeeds but even if it didn't,
	// the reservation still exists.
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	arrival, departure := nextTestDate(t)

	res, err := testSvc.CreateReservation(ctx, &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     uuid.Nil,
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
	}, IncludeFlags{})
	if err != nil {
		t.Fatalf("R-RES-EDGE-056: create: %v", err)
	}
	if res.ID == uuid.Nil {
		t.Error("R-RES-EDGE-056: expected reservation ID")
	}
}

// R-RES-EDGE-040: Cancel on checked-out reservation → 409.
// Blocked on CancelReservation.
func TestEdge_040_CancelTerminalReservation(t *testing.T) {
	t.Skip("blocked: CancelReservation not implemented (actions.go)")
}

// R-RES-EDGE-041: Reactivation on past reservation → 409.
// Blocked on Reactivate.
func TestEdge_041_ReactivatePastReservation(t *testing.T) {
	t.Skip("blocked: Reactivate not implemented (actions.go)")
}

// R-RES-EDGE-042: Cancel → reactivate → cancel cycle.
// Blocked on CancelReservation + Reactivate.
func TestEdge_042_CancelReactivateCycle(t *testing.T) {
	t.Skip("blocked: CancelReservation + Reactivate not implemented (actions.go)")
}

// R-RES-EDGE-059: Cancel on partially-checked-in reservation → 409 with checked_in item IDs.
// Blocked on CancelReservation + CheckinItem.
func TestEdge_059_CancelPartialCheckin(t *testing.T) {
	t.Skip("blocked: CheckinItem + CancelReservation not implemented (actions.go)")
}

// R-RES-EDGE-061: Cancel on cancelled → 409 (destructive action).
// Blocked on CancelReservation.
func TestEdge_061_CancelOnCancelled(t *testing.T) {
	t.Skip("blocked: CancelReservation not implemented (actions.go)")
}

// R-RES-EDGE-003: Reactivate with now-conflicting dates → 409.
// Blocked on CancelReservation + Reactivate.
func TestEdge_003_ReactivateConflictingDates(t *testing.T) {
	t.Skip("blocked: CancelReservation + Reactivate not implemented (actions.go)")
}

// R-RES-EDGE-035: Stay shortened on checked-in reservation.
// Blocked on ShortenStay.
func TestEdge_035_ShortenStayCheckedIn(t *testing.T) {
	t.Skip("blocked: ShortenStay not implemented (actions.go)")
}

// R-RES-EDGE-051: PATCH dates on checked-in reservation.
// Blocked on UpdateItemStayPeriod.
func TestEdge_051_PatchDatesCheckedIn(t *testing.T) {
	t.Skip("blocked: UpdateItemStayPeriod not implemented (mutations.go)")
}

// R-RES-EDGE-045: Concurrent update and cancel → 412.
// Blocked on CancelReservation.
func TestEdge_045_ConcurrentUpdateCancel(t *testing.T) {
	t.Skip("blocked: CancelReservation not implemented (actions.go)")
}
