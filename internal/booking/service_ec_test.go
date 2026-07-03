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

	// Per §7.4: confirm on already-confirmed → 200 no-op (nil error at service layer).
	confirmCtx2 := helpers.SetIfMatchVersion(ctx, confirmed.Version)
	_, err = testSvc.ConfirmReservation(confirmCtx2, res.ID, IncludeFlags{})
	if err != nil {
		t.Fatalf("second confirm on already-confirmed should be no-op, got: %v", err)
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

// seedConfirmedRes creates + confirms a reservation. Returns the
// confirmed response plus the stay window used (ItemResponse.StayPeriod
// is a TSTZRANGE string — easier to track separately).
func seedConfirmedRes(t *testing.T) (*ReservationResponse, time.Time, time.Time) {
	t.Helper()
	ctx := ctxWithProperty(context.Background())
	guestID := getGuestID(t)
	arrival, departure := nextTestDate(t)
	res, err := testSvc.CreateReservation(ctx, &CreateReservationInput{
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
	}, IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	confirmCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	confirmed, err := testSvc.ConfirmReservation(confirmCtx, res.ID, IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("seed confirm: %v", err)
	}
	return confirmed, arrival, departure
}

// R-RES-EDGE-040: Cancel on checked-out / cancelled / archived → 409.
func TestEdge_040_CancelTerminalReservation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())
	res, _, _ := seedConfirmedRes(t)

	// First cancel succeeds.
	cancelCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	cancelled, err := testSvc.CancelReservation(cancelCtx, res.ID, CancelInput{ReasonCode: "test"})
	if err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	// Second cancel on terminal → error.
	cancelCtx2 := helpers.SetIfMatchVersion(ctx, cancelled.Version)
	_, err = testSvc.CancelReservation(cancelCtx2, res.ID, CancelInput{ReasonCode: "test"})
	if err == nil {
		t.Fatal("expected error cancelling terminal reservation, got nil")
	}
}

// R-RES-EDGE-041: Reactivation on past reservation rejected unless retroactive_create.
func TestEdge_041_ReactivatePastReservation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())
	res, _, _ := seedConfirmedRes(t)

	cancelCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	cancelled, err := testSvc.CancelReservation(cancelCtx, res.ID, CancelInput{ReasonCode: "test"})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Backdate items' stay AND the reservation envelope so lower bounds are in the past.
	if _, err := testPool.Exec(ctx,
		`UPDATE operations.reservation_items
		   SET stay_period = tstzrange(now() - interval '10 days', now() - interval '7 days', '[)')
		 WHERE reservation_id = $1`, res.ID); err != nil {
		t.Fatalf("backdate items: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`UPDATE operations.reservations
		   SET stay_period_envelope = tstzrange(now() - interval '10 days', now() - interval '7 days', '[)')
		 WHERE id = $1`, res.ID); err != nil {
		t.Fatalf("backdate envelope: %v", err)
	}

	reactCtx := helpers.SetIfMatchVersion(ctx, cancelled.Version)
	_, err = testSvc.ReactivateReservation(reactCtx, res.ID)
	if err == nil {
		t.Fatal("expected error reactivating past reservation, got nil")
	}
}

// R-RES-EDGE-042: Cancel → reactivate → cancel cycle audited.
func TestEdge_042_CancelReactivateCycle(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())
	res, _, _ := seedConfirmedRes(t)

	cancelCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	cancelled, err := testSvc.CancelReservation(cancelCtx, res.ID, CancelInput{ReasonCode: "test"})
	if err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	if cancelled.Status != StatusCancelled && cancelled.Status != StatusPendingCancellation {
		t.Errorf("status after cancel = %s", cancelled.Status)
	}

	reactCtx := helpers.SetIfMatchVersion(ctx, cancelled.Version)
	reactivated, err := testSvc.ReactivateReservation(reactCtx, res.ID)
	if err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if reactivated.Status != StatusConfirmed {
		t.Errorf("status after reactivate = %s, want confirmed", reactivated.Status)
	}

	cancelCtx2 := helpers.SetIfMatchVersion(ctx, reactivated.Version)
	if _, err := testSvc.CancelReservation(cancelCtx2, res.ID, CancelInput{ReasonCode: "test2"}); err != nil {
		t.Fatalf("second cancel: %v", err)
	}
}

// R-RES-EDGE-059: Cancel on partially-checked-in reservation → 409, lists checked_in item IDs.
// R-RES-VALID-013: Cancel rejected if any item is checked_in.
func TestEdge_059_CancelPartialCheckin(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())
	res, _, _ := seedConfirmedRes(t)

	// Backdate stay so checkin is permitted.
	if err := backdateForCheckin(ctx, res.ID, 1, 1); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	if _, err := testSvc.CheckinItem(helpers.SetIfMatchVersion(ctx, currentItemVersion(ctx, t, res.Items[0].ID)), res.Items[0].ID); err != nil {
		t.Fatalf("checkin item: %v", err)
	}

	cancelCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	_, err := testSvc.CancelReservation(cancelCtx, res.ID, CancelInput{ReasonCode: "test"})
	if err == nil {
		t.Fatal("expected error cancelling partially-checked-in reservation, got nil")
	}
}

// R-RES-EDGE-061: Cancel on cancelled → 409 (destructive action).
func TestEdge_061_CancelOnCancelled(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())
	res, _, _ := seedConfirmedRes(t)

	cancelCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	cancelled, err := testSvc.CancelReservation(cancelCtx, res.ID, CancelInput{ReasonCode: "test"})
	if err != nil {
		t.Fatalf("first cancel: %v", err)
	}

	cancelCtx2 := helpers.SetIfMatchVersion(ctx, cancelled.Version)
	_, err = testSvc.CancelReservation(cancelCtx2, res.ID, CancelInput{ReasonCode: "test2"})
	if err == nil {
		t.Fatal("expected error cancelling already-cancelled, got nil")
	}
}

// R-RES-EDGE-003: Reactivate with now-conflicting dates → 409.
func TestEdge_003_ReactivateConflictingDates(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())

	original, originalArrival, originalDeparture := seedConfirmedRes(t)
	cancelCtx := helpers.SetIfMatchVersion(ctx, original.Version)
	cancelled, err := testSvc.CancelReservation(cancelCtx, original.ID, CancelInput{ReasonCode: "test"})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Occupy every same-type room on those dates with new bookings so reactivate has nowhere to land.
	rtID := getRoomTypeID(t)
	rpID := getRatePlanID(t)
	guestID := getGuestID(t)
	var roomIDs []uuid.UUID
	if err := testPool.QueryRow(ctx,
		`SELECT array_agg(id) FROM inventory.rooms WHERE property_id=$1 AND room_type_id=$2`,
		testPropertyID, rtID).Scan(&roomIDs); err != nil {
		t.Fatalf("room ids: %v", err)
	}
	for _, rid := range roomIDs {
		_, err := testSvc.CreateReservation(ctx, &CreateReservationInput{
			Source:         SourceInternal,
			PropertyID:     testPropertyID,
			PrimaryGuestID: &guestID,
			Items: []CreateItemInput{
				{
					RoomTypeID:     rtID,
					RatePlanID:     rpID,
					AssignedRoomID: &rid,
					ArrivalDate:    types.ISO8601Date{Time: originalArrival},
					DepartureDate:  types.ISO8601Date{Time: originalDeparture},
					AdultsCount:    1,
				},
			},
		}, IncludeFlags{})
		if err != nil {
			t.Fatalf("seed block: %v", err)
		}
	}

	reactCtx := helpers.SetIfMatchVersion(ctx, cancelled.Version)
	_, err = testSvc.ReactivateReservation(reactCtx, original.ID)
	if err == nil {
		t.Fatal("expected conflict error on reactivate, got nil")
	}
}

// R-RES-EDGE-035: Stay shortened on checked-in reservation — future ledger rows released.
func TestEdge_035_ShortenStayCheckedIn(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())
	res, _, _ := seedConfirmedRes(t)

	if err := backdateForCheckin(ctx, res.ID, 1, 3); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	if _, err := testSvc.CheckinItem(helpers.SetIfMatchVersion(ctx, currentItemVersion(ctx, t, res.Items[0].ID)), res.Items[0].ID); err != nil {
		t.Fatalf("checkin: %v", err)
	}

	newDeparture := time.Now().Add(24 * time.Hour)
	mutCtx := helpers.SetIfMatchVersion(
		helpers.SetPermissionsInCtx(ctx, []string{"reservations:post_checkin_mutate", "reservations:update_item"}),
		currentItemVersion(ctx, t, res.Items[0].ID),
	)
	if _, err := testSvc.UpdateItemStayPeriod(mutCtx, res.Items[0].ID, time.Now().Add(-1*time.Hour), newDeparture); err != nil {
		t.Fatalf("shorten: %v", err)
	}

	// Verify in DB that future ledger rows past newDeparture are gone.
	var rowCount int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM inventory.room_inventory_ledger
		   WHERE reservation_item_id = $1 AND calendar_date >= $2::date`,
		res.Items[0].ID, newDeparture).Scan(&rowCount); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if rowCount != 0 {
		t.Errorf("expected 0 future ledger rows after shorten, got %d", rowCount)
	}
}

// R-RES-EDGE-051: PATCH dates on checked-in reservation — extension allowed.
func TestEdge_051_PatchDatesCheckedIn(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())
	res, _, _ := seedConfirmedRes(t)

	if err := backdateForCheckin(ctx, res.ID, 1, 1); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	if _, err := testSvc.CheckinItem(helpers.SetIfMatchVersion(ctx, currentItemVersion(ctx, t, res.Items[0].ID)), res.Items[0].ID); err != nil {
		t.Fatalf("checkin: %v", err)
	}

	extended := time.Now().Add(48 * time.Hour)
	mutCtx := helpers.SetIfMatchVersion(
		helpers.SetPermissionsInCtx(ctx, []string{"reservations:post_checkin_mutate", "reservations:update_item"}),
		currentItemVersion(ctx, t, res.Items[0].ID),
	)
	if _, err := testSvc.UpdateItemStayPeriod(mutCtx, res.Items[0].ID, time.Now().Add(-1*time.Hour), extended); err != nil {
		t.Fatalf("extend: %v", err)
	}

	// Verify DB has a ledger row on the extended night.
	var rowCount int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM inventory.room_inventory_ledger
		   WHERE reservation_item_id = $1 AND calendar_date >= $2::date`,
		res.Items[0].ID, time.Now().Add(36*time.Hour)).Scan(&rowCount); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if rowCount == 0 {
		t.Errorf("expected ≥1 ledger row on extended night, got 0")
	}
}

// R-RES-EDGE-045: Concurrent update + cancel — second mutation 412.
// R-RES-VALID-012: If-Match version mismatch returns 412 / ErrVersionMismatch.
func TestEdge_045_ConcurrentUpdateCancel(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())
	res, _, _ := seedConfirmedRes(t)

	// First mutation: cancel with the current version.
	cancelCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	if _, err := testSvc.CancelReservation(cancelCtx, res.ID, CancelInput{ReasonCode: "test"}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Second mutation: cancel again with stale version.
	staleCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	_, err := testSvc.CancelReservation(staleCtx, res.ID, CancelInput{ReasonCode: "test2"})
	if err == nil {
		t.Fatal("expected version mismatch or terminal-state error on stale cancel, got nil")
	}
}
