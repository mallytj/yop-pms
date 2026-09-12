// Seed tape chart test data for local tape-chart development.
// Run: go run ./cmd/tools/seed-tape-chart [-xl]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

const (
	defaultRoomsPerType = 2
	xlRoomsPerType      = 10

	// ledger/reservation window: 3 months either side of today.
	ledgerStartOffset = -90
	ledgerDays        = 180
)

// baseEntities are the fixtures every other seed hangs off: room types,
// rooms, the inventory ledger, and the BAR rate plan.
type baseEntities struct {
	roomTypeIDs []string
	roomIDs     []string
	ratePlanID  string
}

func main() {
	xl := flag.Bool("xl", false, "seed an XL property (50 rooms instead of 10)")
	flag.Parse()

	_ = godotenv.Load()

	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		log.Fatal("GOOSE_DBSTRING not set")
	}

	roomsPerType := defaultRoomsPerType
	if *xl {
		roomsPerType = xlRoomsPerType
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	propertyID, err := assertSetupReady(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("property_id:", propertyID)

	base, err := seedBaseEntities(ctx, conn, propertyID, roomsPerType)
	if err != nil {
		log.Fatal(err)
	}

	if err := seedFullReservations(ctx, conn, propertyID, base, roomsPerType); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n--- DONE ---")
	fmt.Printf("Add to web/.env: VITE_DEV_PROPERTY_ID=%s\n", propertyID)
}

// assertSetupReady guarantees the licence and property fixtures exist and
// that leftovers from a prior run are wiped before any seeding happens.
func assertSetupReady(ctx context.Context, conn *pgx.Conn) (string, error) {
	licenceID, err := ensureLicence(ctx, conn)
	if err != nil {
		return "", err
	}
	fmt.Println("licence_id:", licenceID)

	propertyID, err := ensureProperty(ctx, conn, licenceID)
	if err != nil {
		return "", err
	}

	if err := clearPropertyData(ctx, conn, propertyID); err != nil {
		return "", err
	}
	return propertyID, nil
}

func seedBaseEntities(ctx context.Context, conn *pgx.Conn, propertyID string, roomsPerType int) (baseEntities, error) {
	roomTypeIDs, err := seedRoomTypes(ctx, conn, propertyID)
	if err != nil {
		return baseEntities{}, err
	}
	fmt.Printf("room_types: %d seeded\n", len(roomTypeIDs))

	roomIDs, err := seedRooms(ctx, conn, propertyID, roomTypeIDs, roomsPerType)
	if err != nil {
		return baseEntities{}, err
	}
	fmt.Printf("rooms: %d seeded\n", len(roomIDs))

	if err := seedInventory(ctx, conn, propertyID, roomIDs, roomTypeIDs, roomsPerType, ledgerStartOffset, ledgerDays); err != nil {
		return baseEntities{}, err
	}
	fmt.Printf("inventory: %d days x %d rooms\n", ledgerDays, len(roomIDs))

	ratePlanID, err := seedRatePlan(ctx, conn, propertyID)
	if err != nil {
		return baseEntities{}, err
	}
	fmt.Println("rate_plan_id:", ratePlanID)

	return baseEntities{roomTypeIDs: roomTypeIDs, roomIDs: roomIDs, ratePlanID: ratePlanID}, nil
}

func seedFullReservations(ctx context.Context, conn *pgx.Conn, propertyID string, base baseEntities, roomsPerType int) error {
	guestIDs, err := seedGuests(ctx, conn, propertyID)
	if err != nil {
		return err
	}
	fmt.Printf("guests: %d seeded\n", len(guestIDs))

	maintRoomIndices := maintenanceRoomIndices(roomsPerType)
	blockedRanges := maintenanceBlockedRanges(maintRoomIndices)

	count, err := seedReservations(ctx, conn, SeedReservationsParams{
		PropertyID:    propertyID,
		RoomTypeIDs:   base.roomTypeIDs,
		RoomIDs:       base.roomIDs,
		RoomsPerType:  roomsPerType,
		GuestIDs:      guestIDs,
		RatePlanID:    base.ratePlanID,
		BlockedRanges: blockedRanges,
		Rand:          rand.New(rand.NewSource(reservationRandomSeed)),
	})
	if err != nil {
		return err
	}
	fmt.Printf("reservations: %d seeded\n", count)

	blockCount, err := seedMaintenanceBlocks(ctx, conn, propertyID, base.roomIDs, maintRoomIndices)
	if err != nil {
		return err
	}
	fmt.Printf("maintenance_blocks: %d seeded\n", blockCount)
	return nil
}
