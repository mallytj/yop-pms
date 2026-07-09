package booking

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// Shared test infrastructure — initialised by TestMain in service_test.go.
var (
	testPool       *pgxpool.Pool
	testQueries    *store.Queries
	testRedis      *redis.Client
	testSvc        *Service
	testPropertyID uuid.UUID
	testMu         sync.Mutex
	testDateBase   int32 = 7
)

// ctxWithProperty returns a context containing the test property ID.
func ctxWithProperty(ctx context.Context) context.Context {
	return helpers.SetPropertyIDInCtx(ctx, testPropertyID)
}

// backdateReservation sets updated_at on a reservation and its items
// to appear past the archive threshold (default 365 days).
func backdateReservation(t *testing.T, ctx context.Context, reservationID uuid.UUID) {
	t.Helper()
	if _, err := testPool.Exec(ctx,
		`UPDATE operations.reservations SET updated_at = NOW() - INTERVAL '400 days' WHERE id = $1`,
		reservationID); err != nil {
		t.Fatalf("backdate reservation: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`UPDATE operations.reservation_items SET updated_at = NOW() - INTERVAL '400 days' WHERE reservation_id = $1`,
		reservationID); err != nil {
		t.Fatalf("backdate items: %v", err)
	}
}

// getGuestID returns the ID of the seeded test guest.
func getGuestID(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM identity.guests WHERE email = 'jane@example.com'`).Scan(&id)
	if err != nil {
		t.Fatal("get guest id:", err)
	}
	return id
}

// getRoomTypeID returns the ID of the first seeded room type.
func getRoomTypeID(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM inventory.room_types LIMIT 1`).Scan(&id)
	if err != nil {
		t.Fatal("get room type id:", err)
	}
	return id
}

// getRoomID returns the ID of the first seeded room.
func getRoomID(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM inventory.rooms ORDER BY name ASC LIMIT 1`).Scan(&id)
	if err != nil {
		t.Fatal("get room id:", err)
	}
	return id
}

// roomIDPtr returns a pointer to the first room. Used in struct
// literals where AssignedRoomID is *uuid.UUID.
func roomIDPtr(t *testing.T) *uuid.UUID {
	id := getRoomID(t)
	return &id
}

// getRatePlanID returns a pointer to the first seeded rate plan.
func getRatePlanID(t *testing.T) *uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`SELECT id FROM pricing.rate_plans LIMIT 1`).Scan(&id)
	if err != nil {
		t.Fatal("get rate plan id:", err)
	}
	return &id
}

// nextTestDate returns a non-overlapping (arrival, departure) pair for tests.
func nextTestDate(t *testing.T) (time.Time, time.Time) {
	t.Helper()
	testMu.Lock()
	days := testDateBase
	testDateBase += 7
	testMu.Unlock()
	arrival := time.Now().Truncate(24 * time.Hour).Add(time.Duration(days) * 24 * time.Hour)
	return arrival, arrival.Add(3 * 24 * time.Hour)
}

// backdateForCheckin shifts a reservation's items + ledger + booked_daily_rates
// backwards in time so the stay_period crosses NOW(), enabling checkin / overstay
// flows in tests. hoursBefore = how many hours ago the stay starts.
// daysSpan = total span in days for the resulting window.
//
// All affected ledger rows have their calendar_date shifted by the same delta so
// the EXCLUDE constraint and ledger UNIQUE remain satisfied. version bumps on
// the item so subsequent service calls don't trip optimistic locks.
func backdateForCheckin(ctx context.Context, reservationID uuid.UUID, hoursBefore, daysSpan int) error {
	// Encode intervals as bigint seconds — pgx doesn't auto-encode time.Duration.
	beforeSec := int64(hoursBefore) * 3600
	spanSec := int64(daysSpan) * 24 * 3600

	tx, err := testPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`SELECT set_config('app.current_property_id', $1, true)`,
		testPropertyID.String()); err != nil {
		return err
	}

	// Compute delta from existing lower(stay_period) to (now - beforeSec).
	var deltaSeconds int64
	if err := tx.QueryRow(ctx,
		`SELECT EXTRACT(EPOCH FROM ((now() - ($2::bigint || ' seconds')::interval) - lower(stay_period)))::bigint
		   FROM operations.reservation_items
		  WHERE reservation_id = $1
		  LIMIT 1`, reservationID, beforeSec).Scan(&deltaSeconds); err != nil {
		return err
	}

	// Move ledger + rates FIRST (so EXCLUDE on the items table doesn't fire
	// on the intermediate stay_period vs existing ledger pin).
	if _, err := tx.Exec(ctx,
		`UPDATE inventory.room_inventory_ledger
		    SET calendar_date = calendar_date + ($2::bigint || ' seconds')::interval
		  WHERE reservation_item_id IN (
		      SELECT id FROM operations.reservation_items WHERE reservation_id = $1
		  )`, reservationID, deltaSeconds); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE operations.reservation_items
		    SET stay_period = tstzrange(
		            now() - ($2::bigint || ' seconds')::interval,
		            now() - ($2::bigint || ' seconds')::interval + ($3::bigint || ' seconds')::interval,
		            '[)'),
		        version = version + 1
		  WHERE reservation_id = $1`, reservationID, beforeSec, spanSec); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// currentItemVersion reads the current optimistic-lock version of an item.
func currentItemVersion(ctx context.Context, t *testing.T, itemID uuid.UUID) int32 {
	t.Helper()
	var v int32
	if err := testPool.QueryRow(ctx,
		`SELECT version FROM operations.reservation_items WHERE id = $1`, itemID).Scan(&v); err != nil {
		t.Fatalf("get item version: %v", err)
	}
	return v
}

// cleanupTestReservations deletes transient reservation data while preserving seed data
// (properties, room types, guests, rate plans). Called via t.Cleanup.
// Uses a mutex to prevent concurrent cleanup races between test goroutines.
func cleanupTestReservations() {
	testMu.Lock()
	defer testMu.Unlock()

	ctx := context.Background()
	conn, err := testPool.Acquire(ctx)
	if err != nil {
		log.Printf("cleanup acquire: %v", err)
		return
	}
	defer conn.Release()

	// audit_logs trigger reads current_setting('app.current_property_id').
	// Must be set on the same connection that performs the DELETEs.
	// Disable triggers during teardown — audit_logs trigger references
	// NEW.property_id on DELETE (schema bug), which would otherwise abort cleanup.
	if _, err := conn.Exec(ctx, "SET session_replication_role = 'replica'"); err != nil {
		log.Printf("cleanup disable triggers: %v", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "SET session_replication_role = 'origin'")
	}()

	for _, tbl := range []string{
		"inventory.room_inventory_ledger",
		"operations.reservation_items",
		"operations.reservations",
	} {
		if _, err := conn.Exec(ctx, "DELETE FROM "+tbl); err != nil {
			log.Printf("cleanup %s: %v", tbl, err)
		}
	}
}
