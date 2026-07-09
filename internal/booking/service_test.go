package booking

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/types"
	"github.com/lexxcode1/yop-pms/internal/store"
)

func TestMain(m *testing.M) {
	code := runTests(m)
	os.Exit(code)
}

func runTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pgContainer, err := postgres.Run(
		ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("pms_test"),
		postgres.WithUsername("admin"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		log.Printf("start postgres: %v", err)
		return 1
	}
	defer func() {
		if err := pgContainer.Terminate(context.Background()); err != nil {
			log.Printf("terminate postgres: %v", err)
		}
	}()

	connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")

	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Printf("open sql: %v", err)
		return 1
	}
	if err := goose.Up(sqlDB, "../../migrations"); err != nil {
		log.Printf("migrations: %v", err)
		sqlDB.Close()
		return 1
	}
	sqlDB.Close()

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		log.Printf("connect pool: %v", err)
		return 1
	}
	defer testPool.Close()

	testQueries = store.New(testPool)

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("* Ready to accept connections").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		log.Printf("start redis: %v", err)
		return 1
	}
	defer func() {
		if err := redisContainer.Terminate(context.Background()); err != nil {
			log.Printf("terminate redis: %v", err)
		}
	}()

	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		log.Printf("redis host: %v", err)
		return 1
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		log.Printf("redis port: %v", err)
		return 1
	}
	testRedis = redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort.Port(),
	})
	defer testRedis.Close()

	testSvc = NewService(testPool, testQueries, testRedis, slog.Default())

	if err := seedTestData(ctx); err != nil {
		log.Printf("seed: %v", err)
		return 1
	}

	return m.Run()
}

func seedTestData(ctx context.Context) error {
	var licID uuid.UUID
	if err := testPool.QueryRow(ctx,
		`INSERT INTO operations.licences (licence_key, organisation_name, contact_email)
		 VALUES ($1,$2,$3) RETURNING id`,
		"YOP-12345", "Test Org", "test@test.com").Scan(&licID); err != nil {
		return fmt.Errorf("seed licence: %w", err)
	}

	if err := testPool.QueryRow(ctx,
		`INSERT INTO operations.properties (name, licence_id, address, timezone)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		"Test Property", licID, "123 Test St", "Europe/London").Scan(&testPropertyID); err != nil {
		return fmt.Errorf("seed property: %w", err)
	}

	var rtID uuid.UUID
	if err := testPool.QueryRow(ctx,
		`INSERT INTO inventory.room_types (property_id, name, code, std_occupancy, max_occupancy)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		testPropertyID, "Double Room", "DBL", 2, 4).Scan(&rtID); err != nil {
		return fmt.Errorf("seed room type: %w", err)
	}

	var rpID uuid.UUID
	if err := testPool.QueryRow(ctx,
		`INSERT INTO pricing.rate_plans (property_id, name, code, currency_code)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		testPropertyID, "Standard Rate", "BAR", "GBP").Scan(&rpID); err != nil {
		return fmt.Errorf("seed rate plan: %w", err)
	}

	// Seed base rate for all days of week (£100.00 = 10000 pence).
	for dow := 0; dow <= 6; dow++ {
		if _, err := testPool.Exec(ctx,
			`INSERT INTO pricing.base_rates (property_id, room_type_id, rate_plan_id, day_of_week, base_price_pence)
			 VALUES ($1,$2,$3,$4,$5)`,
			testPropertyID, rtID, rpID, dow, 10000); err != nil {
			return fmt.Errorf("seed base rate dow=%d: %w", dow, err)
		}
	}

	for _, name := range []string{"101", "102", "103"} {
		if _, err := testPool.Exec(ctx,
			`INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1,$2,$3)`,
			testPropertyID, rtID, name); err != nil {
			return fmt.Errorf("seed room %s: %w", name, err)
		}
	}

	if _, err := testPool.Exec(ctx,
		`INSERT INTO identity.guests (property_id, first_name, last_name, email)
		 VALUES ($1,$2,$3,$4)`,
		testPropertyID, "Jane", "Doe", "jane@example.com"); err != nil {
		return fmt.Errorf("seed guest: %w", err)
	}

	return nil
}

// --- Tests ---

// R-RES-CRUD-001: Create reservation with primary guest + items.
// R-RES-CRUD-007: Reservation code RES-XXXXXX sequential per property.
// R-RES-CRUD-009: ≥1 reservation_item + booked_daily_rates atomic with insert.
// R-RES-CRUD-010: Creation creates Folio A.
// R-RES-CRUD-011: Each item gets a room_inventory_ledger entry.
// R-RES-CRUD-013: Lifecycle: hold default for internal source.
// R-RES-AVAIL-009: Auto-pin at hold creation.
func TestCreate_Hold(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
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
				AdultsCount:   2,
			},
		},
	}

	res, err := testSvc.CreateReservation(ctx, input, IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}

	if res.Status != StatusHold {
		t.Errorf("status = %s, want hold", res.Status)
	}
	if res.Code == "" {
		t.Error("code is empty")
	}
	if len(res.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(res.Items))
	}
	if res.Items[0].Status != ItemStatusBooked {
		t.Errorf("item status = %s, want booked", res.Items[0].Status)
	}
}

// R-RES-CRUD-013: Lifecycle: walkin bypasses to checked_in.
// R-RES-CRUD-014: Walk-in requires room assigned, creates in checked_in today only.
// R-RES-VALID-011: Walk-in must be created today, room assigned immediately.
func TestCreate_Walkin(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	arrival, departure := nextTestDate(t)

	guestID := getGuestID(t)
	input := &CreateReservationInput{
		Source:         SourceInternal,
		IsWalkin:       true,
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
	}

	res, err := testSvc.CreateReservation(ctx, input, IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("walkin: %v", err)
	}

	if res.Status != StatusCheckedIn {
		t.Errorf("status = %s, want checked_in", res.Status)
	}
	if res.Items[0].Status != ItemStatusCheckedIn {
		t.Errorf("item status = %s, want checked_in", res.Items[0].Status)
	}
}

// R-RES-VALID-002: lower(stay_period) not before today (no retroactive_create perm).
func TestCreate_PastDate_Rejected(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	arrival := time.Now().Truncate(24 * time.Hour).Add(-7 * 24 * time.Hour)

	input := &CreateReservationInput{
		Source:         SourceInternal,
		PropertyID:     testPropertyID,
		PrimaryGuestID: &guestID,
		Items: []CreateItemInput{
			{
				RoomTypeID:    getRoomTypeID(t),
				RatePlanID:    getRatePlanID(t),
				ArrivalDate:   types.ISO8601Date{Time: arrival},
				DepartureDate: types.ISO8601Date{Time: arrival.Add(2 * 24 * time.Hour)},
				AdultsCount:   1,
			},
		},
	}

	_, err := testSvc.CreateReservation(ctx, input, IncludeFlags{})
	if err == nil {
		t.Fatal("expected error for past date, got nil")
	}
}

// R-RES-CRUD-014: Walk-in requires room assigned on every item.
// R-RES-VALID-011: Walk-in must have room assigned immediately.
func TestCreate_WalkinNoRoom_Rejected(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	arrival, departure := nextTestDate(t)

	input := &CreateReservationInput{
		Source:         SourceInternal,
		IsWalkin:       true,
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

	_, err := testSvc.CreateReservation(ctx, input, IncludeFlags{})
	if err == nil {
		t.Fatal("expected error for walkin without room, got nil")
	}
}

// R-RES-CRUD-015: POST accepts existing primary_guest_id.
func TestCreate_GuestExpansion(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)

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
				AdultsCount:   2,
			},
		},
	}

	res, err := testSvc.CreateReservation(ctx, input, IncludeFlags{Guest: true})
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}

	if res.Guest == nil {
		t.Fatal("guest expansion missing")
	}
	if res.Guest.Email != "jane@example.com" {
		t.Errorf("guest email = %s, want jane@example.com", res.Guest.Email)
	}
}

// R-RES-CRUD-015: POST accepts inline guest payload created in same tx.
// R-RES-GROOM-001: Primary guest required at creation (inline OK).
func TestCreate_InlineGuest(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	arrival, departure := nextTestDate(t)

	input := &CreateReservationInput{
		Source:     SourceInternal,
		PropertyID: testPropertyID,
		Guest: &GuestInlinePayload{
			FirstName: "John",
			LastName:  "Smith",
			Email:     fmt.Sprintf("john-%s@example.com", uuid.New().String()[:8]),
		},
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

	res, err := testSvc.CreateReservation(ctx, input, IncludeFlags{Guest: true})
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}

	if res.Guest == nil {
		t.Fatal("guest expansion missing")
	}
	if res.Guest.FirstName != "John" {
		t.Errorf("first_name = %s, want John", res.Guest.FirstName)
	}
}

func TestCreate_IncludeNone(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	arrival, departure := nextTestDate(t)
	guestID := getGuestID(t)

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
				AdultsCount:   2,
			},
		},
	}

	res, err := testSvc.CreateReservation(ctx, input, IncludeFlags{None: true})
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}

	if len(res.Items) != 0 {
		t.Errorf("got %d items with include=none, want 0", len(res.Items))
	}
}

// R-RES-CRUD-018: POST /confirm — hold→confirmed staff path.
// R-RES-CRUD-013: Lifecycle: hold→confirmed for internal.
func TestConfirm_HoldToConfirmed(t *testing.T) {
	ctx := ctxWithProperty(context.Background())
	t.Cleanup(func() { cleanupTestReservations() })
	guestID := getGuestID(t)
	arrival, departure := nextTestDate(t)

	createInput := &CreateReservationInput{
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
				AdultsCount:    2,
			},
		},
	}

	res, err := testSvc.CreateReservation(ctx, createInput, IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	confirmCtx := helpers.SetIfMatchVersion(ctx, res.Version)
	confirmed, err := testSvc.ConfirmReservation(confirmCtx, res.ID, IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	if confirmed.Status != StatusConfirmed {
		t.Errorf("status = %s, want confirmed", confirmed.Status)
	}
	if len(confirmed.Items) != 1 {
		t.Errorf("got %d items, want 1", len(confirmed.Items))
	}
}
