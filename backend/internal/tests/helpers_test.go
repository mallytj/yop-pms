package integration_tests

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// withPropertyCtx returns a context with the given property ID injected,
// which is required by all service methods that call ExecuteTx / SetCurrentPropertyID.
func withPropertyCtx(propertyID uuid.UUID) context.Context {
	return hf.SetIDInCtx(context.Background(), hf.PropertyIDKey, propertyID)
}

// withResItemCtx returns a context carrying both a property ID and a reservation
// item ID, as required by reservationService.UpdateItem.
func withResItemCtx(propertyID, resItemID uuid.UUID) context.Context {
	ctx := withPropertyCtx(propertyID)
	return hf.SetIDInCtx(ctx, hf.ReservationItemIDKey, resItemID)
}

// ---- DB seed helpers (raw SQL, same approach as the DB adapter tests) ----

func seedProperty(t *testing.T, ctx context.Context) uuid.UUID {
	t.Helper()
	licID := seedLicence(t, ctx)
	var id uuid.UUID
	err := testDB.QueryRow(ctx,
		`INSERT INTO operations.properties (licence_id, name, address, timezone)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		licID,
		"Integration Test Property "+uuid.New().String()[:6],
		"1 Test Lane",
		"UTC",
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedLicence(t *testing.T, ctx context.Context) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	key := fmt.Sprintf("YOP-%05d", rand.Intn(100000))
	err := testDB.QueryRow(ctx,
		`INSERT INTO operations.licences (licence_key, organisation_name, contact_email, is_active)
		 VALUES ($1, $2, $3, true) RETURNING id`,
		key, "Test Org", "test@example.com",
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedRoomType(t *testing.T, ctx context.Context, propertyID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx,
		`INSERT INTO inventory.room_types (property_id, name, code, std_occupancy, min_occupancy, max_occupancy)
		 VALUES ($1, $2, $3, 2, 1, 2) RETURNING id`,
		propertyID,
		"Type "+uuid.New().String()[:6],
		fmt.Sprintf("T%05X", rand.Intn(0xFFFFF)),
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedRoom(t *testing.T, ctx context.Context, propertyID, roomTypeID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx,
		`INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1, $2, $3) RETURNING id`,
		propertyID, roomTypeID,
		"Room "+uuid.New().String()[:6],
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedGuest(t *testing.T, ctx context.Context, propertyID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx,
		`INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number)
		 VALUES ($1, 'Integration', 'Guest', $2, '07700000000') RETURNING id`,
		propertyID,
		"guest_"+uuid.New().String()[:8]+"@example.com",
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedReservation(t *testing.T, ctx context.Context, propertyID, guestID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx,
		`INSERT INTO operations.reservations (property_id, primary_guest_id, source, notes, status)
		 VALUES ($1, $2, 'direct', 'integration test', 'confirmed') RETURNING id`,
		propertyID, guestID,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedRatePlan(t *testing.T, ctx context.Context, propertyID uuid.UUID, isActive bool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx,
		`INSERT INTO pricing.rate_plans (property_id, name, code, description, is_active, currency_code)
		 VALUES ($1, $2, $3, 'integration test', $4, 'GBP') RETURNING id`,
		propertyID,
		"Rate Plan "+uuid.New().String()[:6],
		fmt.Sprintf("R%05X", rand.Intn(0xFFFFF)),
		isActive,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedReservationItem(t *testing.T, ctx context.Context, propertyID, reservationID, roomTypeID, ratePlanID uuid.UUID, checkIn, checkOut time.Time) uuid.UUID {
	t.Helper()
	stayPeriod := hf.ToPgTstzRange(checkIn, checkOut)
	var id uuid.UUID
	err := testDB.QueryRow(ctx,
		`INSERT INTO operations.reservation_items
		   (property_id, reservation_id, booked_room_type_id, rate_plan_id, stay_period, base_rate_pence, adults_count, children_count, status)
		 VALUES ($1, $2, $3, $4, $5, 15000, 2, 0, 'booked') RETURNING id`,
		propertyID, reservationID, roomTypeID, ratePlanID, stayPeriod,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// seedBaseRate inserts a row into pricing.base_rates for every day of the week.
// This is required by GetRatesForRange which joins on base_rates.
func seedBaseRate(t *testing.T, ctx context.Context, propertyID, roomTypeID, ratePlanID uuid.UUID, pricePence int32) {
	t.Helper()
	for dow := 0; dow <= 6; dow++ {
		_, err := testDB.Exec(ctx,
			`INSERT INTO pricing.base_rates (property_id, room_type_id, rate_plan_id, day_of_week, base_price_pence, min_los_restriction, max_los_restriction)
			 VALUES ($1, $2, $3, $4, $5, 1, 30)`,
			propertyID, roomTypeID, ratePlanID, dow, pricePence,
		)
		require.NoError(t, err, "seeding base_rate for dow=%d", dow)
	}
}

// seedDailyPriceGrid inserts an override row for a specific calendar date.
func seedDailyPriceGrid(t *testing.T, ctx context.Context, propertyID, roomTypeID, ratePlanID uuid.UUID, calendarDate string, pricePence int32) {
	t.Helper()
	_, err := testDB.Exec(ctx,
		`INSERT INTO pricing.daily_price_grid (property_id, room_type_id, rate_plan_id, calendar_date, base_price_pence, min_los_restriction, max_los_restriction, is_available)
		 VALUES ($1, $2, $3, $4, $5, 1, 30, true)`,
		propertyID, roomTypeID, ratePlanID,
		calendarDate, pricePence,
	)
	require.NoError(t, err)
}

// mustPgDate converts a "YYYY-MM-DD" string to pgtype.Date.
func mustPgDate(s string) pgtype.Date {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic("bad date: " + s)
	}
	return pgtype.Date{Time: t, Valid: true}
}

// ctx0 returns a plain background context (no IDs injected), useful as a
// base for DB seed calls that don't require RLS context.
func ctx0() context.Context { return context.Background() }

// zeroUUID returns a uuid.Nil — a named constant that communicates intent
// ("no ID / missing ID") more clearly than uuid.Nil inline.
func zeroUUID() uuid.UUID { return uuid.Nil }
