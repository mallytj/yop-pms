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

// cleanupTestReservations deletes transient reservation data while preserving seed data
// (properties, room types, guests, rate plans). Called via t.Cleanup.
// Uses a mutex to prevent concurrent cleanup races between test goroutines.
func cleanupTestReservations() {
	testMu.Lock()
	defer testMu.Unlock()

	ctx := context.Background()
	for _, tbl := range []string{
		"finance.folio_transactions",
		"inventory.room_inventory_ledger",
		"pricing.booked_daily_rates",
		"finance.folios",
		"operations.reservation_items",
		"operations.reservations",
	} {
		if _, err := testPool.Exec(ctx, "DELETE FROM "+tbl); err != nil {
			log.Printf("cleanup %s: %v", tbl, err)
		}
	}
}
