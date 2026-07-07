package booking

// Worker tests mapped to R-RES-WORKER-XXX in docs/requirements/reservations.md §12.
// Each test function references the requirement ID in its doc comment for rtmcheck.

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/types"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// R-RES-WORKER-002 / R-RES-INTEG-008: Archival sweep archives terminal reservations
// past the property's archive threshold.
func TestWorker_ArchivalSweep(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())

	guestID := getGuestID(t)
	arrival, departure := nextTestDate(t)
	input := &CreateReservationInput{
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
	}
	res, err := testSvc.CreateReservation(ctx, input, IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Cancel to make it terminal.
	cancelCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	cancelled, err := testSvc.CancelReservation(cancelCtx, res.ID, CancelInput{ReasonCode: "test"})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Backdate so it appears past the archive threshold (default 365 days).
	backdateReservation(t, ctx, res.ID)

	// Run archival on this reservation directly — use version from cancelled response.
	w := NewWorkers(testPool, testQueries, slog.Default())
	if err := w.archiveReservationTx(ctx, store.OperationsReservation{
		ID:         res.ID,
		PropertyID: testPropertyID,
		Version:    cancelled.Version,
	}); err != nil {
		t.Fatalf("archiveReservationTx: %v", err)
	}

	// Verify reservation is archived.
	var status string
	if err := testPool.QueryRow(
		ctx,
		`SELECT status::text FROM operations.reservations WHERE id = $1`, res.ID,
	).Scan(&status); err != nil {
		t.Fatalf("query reservation: %v", err)
	}
	if status != string(StatusArchived) {
		t.Errorf("status = %q, want archived", status)
	}
}

// R-RES-WORKER-003: No-show reminder finds overdue check-ins and emits
// staff_alerts NOTIFY. No status change — staff acts manually (§4.4).
func TestWorker_NoShowReminder(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())

	guestID := getGuestID(t)
	roomTypeID := getRoomTypeID(t)
	ratePlanID := getRatePlanID(t)

	// Create a reservation with items still in booked status but past arrival.
	pastArrival := time.Now().Add(-48 * time.Hour)
	pastDeparture := time.Now().Add(-24 * time.Hour)

	resID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())
	stayRange := pgtype.Range[pgtype.Timestamptz]{
		Lower:     pgtype.Timestamptz{Time: pastArrival, Valid: true},
		Upper:     pgtype.Timestamptz{Time: pastDeparture, Valid: true},
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive,
		Valid:     true,
	}

	if _, err := testPool.Exec(
		ctx,
		`INSERT INTO operations.reservations (id, property_id, primary_guest_id, source, status, stay_period_envelope)
		 VALUES ($1, $2, $3, 'internal', 'confirmed', tstzrange($4::timestamptz, $5::timestamptz, '[)'))`,
		resID, testPropertyID, guestID, pastArrival, pastDeparture,
	); err != nil {
		t.Fatalf("insert reservation: %v", err)
	}
	if _, err := testPool.Exec(
		ctx,
		`INSERT INTO operations.reservation_items (id, property_id, reservation_id, booked_room_type_id, rate_plan_id, stay_period, status, adults_count)
		 VALUES ($1, $2, $3, $4, $5, $6, 'booked', 1)`,
		itemID, testPropertyID, resID, roomTypeID, *ratePlanID, stayRange,
	); err != nil {
		t.Fatalf("insert item: %v", err)
	}

	// Run a single no-show check tick.
	w := NewWorkers(testPool, testQueries, slog.Default())
	w.checkNoShows(ctx)

	// Verify items are still in booked status — the worker only notifies.
	var status string
	if err := testPool.QueryRow(
		ctx,
		`SELECT status::text FROM operations.reservation_items WHERE id = $1`, itemID,
	).Scan(&status); err != nil {
		t.Fatalf("query item: %v", err)
	}
	if status != string(ItemStatusBooked) {
		t.Errorf("status = %q, want booked (worker should not mutate)", status)
	}
}

// R-RES-WORKER-005: Overstay sweep transitions checked_in items past
// departure + grace to overstay status.
func TestWorker_OverstaySweep(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Cleanup(cleanupTestReservations)
	ctx := ctxWithProperty(context.Background())

	guestID := getGuestID(t)
	roomTypeID := getRoomTypeID(t)
	roomID := getRoomID(t)
	ratePlanID := getRatePlanID(t)

	// Create a reservation with an item already checked_in and past departure.
	pastArrival := time.Now().Add(-48 * time.Hour)
	pastDeparture := time.Now().Add(-2 * time.Hour) // 2 hours past → beyond default 0-min grace

	resID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())
	stayRange := pgtype.Range[pgtype.Timestamptz]{
		Lower:     pgtype.Timestamptz{Time: pastArrival, Valid: true},
		Upper:     pgtype.Timestamptz{Time: pastDeparture, Valid: true},
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive,
		Valid:     true,
	}

	if _, err := testPool.Exec(
		ctx,
		`INSERT INTO operations.reservations (id, property_id, primary_guest_id, source, status, stay_period_envelope)
		 VALUES ($1, $2, $3, 'internal', 'checked_in', tstzrange($4::timestamptz, $5::timestamptz, '[)'))`,
		resID, testPropertyID, guestID, pastArrival, pastDeparture,
	); err != nil {
		t.Fatalf("insert reservation: %v", err)
	}
	if _, err := testPool.Exec(
		ctx,
		`INSERT INTO operations.reservation_items (id, property_id, reservation_id, booked_room_type_id, assigned_room_id, rate_plan_id, stay_period, status, adults_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'checked_in', 1)`,
		itemID, testPropertyID, resID, roomTypeID, roomID, *ratePlanID, stayRange,
	); err != nil {
		t.Fatalf("insert item: %v", err)
	}

	// Run overstay sweep.
	w := NewWorkers(testPool, testQueries, slog.Default())
	w.processOverstays(ctx)

	// Verify item is now in overstay status.
	var status string
	if err := testPool.QueryRow(
		ctx,
		`SELECT status::text FROM operations.reservation_items WHERE id = $1`, itemID,
	).Scan(&status); err != nil {
		t.Fatalf("query item: %v", err)
	}
	if status != string(ItemStatusOverstay) {
		t.Errorf("status = %q, want overstay", status)
	}
}
