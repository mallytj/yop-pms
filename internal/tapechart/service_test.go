package tapechart

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
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

	"github.com/lexxcode1/yop-pms/internal/booking"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/platform/types"
	"github.com/lexxcode1/yop-pms/internal/store"
)

var (
	testPool       *pgxpool.Pool
	testQueries    *store.Queries
	testSvc        *Service
	testBookingSvc *booking.Service
	testPropertyID uuid.UUID
	testRoomTypeID uuid.UUID
	testRoomID     uuid.UUID
	testWindowMu   sync.Mutex
	testWindowBase int32 = 7
)

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
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
	testRedis := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort.Port(),
	})
	defer testRedis.Close()

	testSvc = NewService(testPool, testQueries, slog.Default())
	testBookingSvc = booking.NewService(testPool, testQueries, testRedis, slog.Default())

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
		"YOP-99058", "Test Org", "test@test.com").Scan(&licID); err != nil {
		return fmt.Errorf("seed licence: %w", err)
	}

	if err := testPool.QueryRow(ctx,
		`INSERT INTO operations.properties (name, licence_id, address, timezone)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		"Test Property", licID, "123 Test St", "Europe/London").Scan(&testPropertyID); err != nil {
		return fmt.Errorf("seed property: %w", err)
	}

	if err := testPool.QueryRow(ctx,
		`INSERT INTO inventory.room_types (property_id, name, code, std_occupancy, max_occupancy)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		testPropertyID, "Double Room", "DBL", 2, 4).Scan(&testRoomTypeID); err != nil {
		return fmt.Errorf("seed room type: %w", err)
	}

	var rpID uuid.UUID
	if err := testPool.QueryRow(ctx,
		`INSERT INTO pricing.rate_plans (property_id, name, code, currency_code)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		testPropertyID, "Standard Rate", "BAR", "GBP").Scan(&rpID); err != nil {
		return fmt.Errorf("seed rate plan: %w", err)
	}
	for dow := 0; dow <= 6; dow++ {
		if _, err := testPool.Exec(ctx,
			`INSERT INTO pricing.base_rates (property_id, room_type_id, rate_plan_id, day_of_week, base_price_pence)
			 VALUES ($1,$2,$3,$4,$5)`,
			testPropertyID, testRoomTypeID, rpID, dow, 10000); err != nil {
			return fmt.Errorf("seed base rate dow=%d: %w", dow, err)
		}
	}

	if err := testPool.QueryRow(ctx,
		`INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1,$2,$3) RETURNING id`,
		testPropertyID, testRoomTypeID, "101").Scan(&testRoomID); err != nil {
		return fmt.Errorf("seed room 101: %w", err)
	}
	if _, err := testPool.Exec(ctx,
		`INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1,$2,$3)`,
		testPropertyID, testRoomTypeID, "102"); err != nil {
		return fmt.Errorf("seed room 102: %w", err)
	}

	return nil
}

// seedReservation creates a walk-in reservation with one item assigned to
// testRoomID over [arrival, departure), which also populates
// room_inventory_ledger (the inventory feed) via the booking service.
func seedReservation(t *testing.T, arrival, departure time.Time) *booking.ReservationResponse {
	t.Helper()
	ctx := helpers.SetPropertyIDInCtx(context.Background(), testPropertyID)

	input := &booking.CreateReservationInput{
		Source:     booking.SourceInternal,
		IsWalkin:   true,
		PropertyID: testPropertyID,
		Guest: &booking.GuestInlinePayload{
			FirstName: "Jane",
			LastName:  "Doe",
			Email:     fmt.Sprintf("jane-%s@example.com", uuid.NewString()),
		},
		Items: []booking.CreateItemInput{
			{
				RoomTypeID:     testRoomTypeID,
				AssignedRoomID: &testRoomID,
				ArrivalDate:    types.ISO8601Date{Time: arrival},
				DepartureDate:  types.ISO8601Date{Time: departure},
				AdultsCount:    1,
			},
		},
	}

	res, err := testBookingSvc.CreateReservation(ctx, input, booking.IncludeFlags{Items: true})
	if err != nil {
		t.Fatalf("seed reservation: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM operations.reservations WHERE id = $1`, res.ID)
	})
	return res
}

// seedMaintenanceBlock inserts a maintenance block directly (no create
// endpoint exists yet in this branch).
func seedMaintenanceBlock(t *testing.T, roomID uuid.UUID, start, end time.Time, reason, blockType string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testPool.QueryRow(context.Background(),
		`INSERT INTO inventory.maintenance_blocks (property_id, room_id, block_period, reason, type)
		 VALUES ($1, $2, tstzrange($3, $4), $5, $6::inventory.maintenance_block_type)
		 RETURNING id`,
		testPropertyID, roomID, start, end, reason, blockType).Scan(&id)
	if err != nil {
		t.Fatalf("seed maintenance block: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM inventory.maintenance_blocks WHERE id = $1`, id)
	})
	return id
}

// nextWindow returns a fresh, non-overlapping [from, to) window on each
// call so concurrent/sequential tests sharing testRoomID never collide on
// the room's exclusion constraint.
func nextWindow(t *testing.T) (time.Time, time.Time) {
	t.Helper()
	testWindowMu.Lock()
	days := testWindowBase
	testWindowBase += 7
	testWindowMu.Unlock()
	from := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, int(days))
	to := from.AddDate(0, 0, 3)
	return from, to
}

func TestGetTapeChart_Shallow_ReturnsFlatBlocks(t *testing.T) {
	from, to := nextWindow(t)
	res := seedReservation(t, from, to)
	mbID := seedMaintenanceBlock(t, testRoomID, to.AddDate(0, 0, 10), to.AddDate(0, 0, 12), "Bathroom leak repair", "repair")

	data, err := testSvc.GetTapeChart(context.Background(), testPropertyID, from, to.AddDate(0, 0, 15), IncludeShallow)
	if err != nil {
		t.Fatalf("GetTapeChart: %v", err)
	}

	if data.RoomTypes != nil {
		t.Errorf("shallow response should omit room_types, got %+v", data.RoomTypes)
	}

	var found *ReservationBlock
	for i := range data.Reservations {
		if data.Reservations[i].RoomID == testRoomID && data.Reservations[i].Code == res.Code {
			found = &data.Reservations[i]
		}
	}
	if found == nil {
		t.Fatalf("reservation block for room %s not found in %+v", testRoomID, data.Reservations)
	}
	if found.GuestName != "Jane Doe" {
		t.Errorf("guest_name = %q, want %q", found.GuestName, "Jane Doe")
	}
	if found.PricePence == nil || *found.PricePence != 10000 {
		t.Errorf("price_pence = %v, want 10000", found.PricePence)
	}

	var maintFound *MaintenanceBlock
	for i := range data.MaintenanceBlocks {
		if data.MaintenanceBlocks[i].ID == mbID {
			maintFound = &data.MaintenanceBlocks[i]
		}
	}
	if maintFound == nil {
		t.Fatalf("maintenance block %s not found in %+v", mbID, data.MaintenanceBlocks)
	}
	if maintFound.Reason != "Bathroom leak repair" {
		t.Errorf("maintenance reason = %q, want the reason text, not the block type", maintFound.Reason)
	}
}

func TestGetTapeChart_Full_NestsRoomsUnderRoomTypes(t *testing.T) {
	from, to := nextWindow(t)
	res := seedReservation(t, from, to)

	data, err := testSvc.GetTapeChart(context.Background(), testPropertyID, from, to, IncludeFull)
	if err != nil {
		t.Fatalf("GetTapeChart: %v", err)
	}

	if data.Reservations != nil || data.MaintenanceBlocks != nil {
		t.Errorf("full response should omit flat arrays, got reservations=%+v maintenance=%+v", data.Reservations, data.MaintenanceBlocks)
	}
	if len(data.RoomTypes) == 0 {
		t.Fatal("expected at least one room type")
	}

	var roomFound *RoomNode
	for _, rt := range data.RoomTypes {
		if rt.ID != testRoomTypeID {
			continue
		}
		for i := range rt.Rooms {
			if rt.Rooms[i].ID == testRoomID {
				roomFound = &rt.Rooms[i]
			}
		}
	}
	if roomFound == nil {
		t.Fatalf("room %s not found nested under room type %s", testRoomID, testRoomTypeID)
	}
	if len(roomFound.Reservations) != 1 || roomFound.Reservations[0].Code != res.Code {
		t.Errorf("room reservations = %+v, want one block for %s", roomFound.Reservations, res.Code)
	}
}

func TestGetTapeChart_AccentIndexStablePerReservation(t *testing.T) {
	from, to := nextWindow(t)
	res := seedReservation(t, from, to)

	data1, err := testSvc.GetTapeChart(context.Background(), testPropertyID, from, to, IncludeShallow)
	if err != nil {
		t.Fatalf("GetTapeChart (1): %v", err)
	}
	data2, err := testSvc.GetTapeChart(context.Background(), testPropertyID, from, to, IncludeShallow)
	if err != nil {
		t.Fatalf("GetTapeChart (2): %v", err)
	}

	accent1, accent2 := -1, -1
	for _, block := range data1.Reservations {
		if block.Code == res.Code {
			accent1 = block.AccentIndex
		}
	}
	for _, block := range data2.Reservations {
		if block.Code == res.Code {
			accent2 = block.AccentIndex
		}
	}
	if accent1 == -1 || accent2 == -1 {
		t.Fatalf("reservation block not found in one of the two responses")
	}
	if accent1 != accent2 {
		t.Errorf("accent_index not stable across requests: %d vs %d", accent1, accent2)
	}
}

func TestGetTapeChart_NoReservationsInWindow_ReturnsEmptyBlocks(t *testing.T) {
	from, to := nextWindow(t)
	// Use a window far enough away that no seeded fixture overlaps it.
	from = from.AddDate(0, 0, 365)
	to = to.AddDate(0, 0, 365)

	data, err := testSvc.GetTapeChart(context.Background(), testPropertyID, from, to, IncludeShallow)
	if err != nil {
		t.Fatalf("GetTapeChart: %v", err)
	}
	if len(data.Reservations) != 0 {
		t.Errorf("reservations = %+v, want empty", data.Reservations)
	}
	if len(data.MaintenanceBlocks) != 0 {
		t.Errorf("maintenance_blocks = %+v, want empty", data.MaintenanceBlocks)
	}
}

func TestGetTapeChart_Inventory_ReflectsReservedNights(t *testing.T) {
	from, to := nextWindow(t)
	seedReservation(t, from, to)

	data, err := testSvc.GetTapeChart(context.Background(), testPropertyID, from, to, IncludeShallow)
	if err != nil {
		t.Fatalf("GetTapeChart: %v", err)
	}
	if len(data.Inventory) == 0 {
		t.Fatal("expected inventory ledger rows for the reserved room's nights")
	}
	for _, day := range data.Inventory {
		if day.RoomID == testRoomID {
			return
		}
	}
	t.Errorf("no inventory day for room %s in %+v", testRoomID, data.Inventory)
}
