// Seed tape chart test data for local tape-chart development.
// Run: go run ./cmd/tools/seed-tape-chart [-xl]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
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

	licenceID, err := ensureLicence(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("licence_id:", licenceID)

	propertyID, err := ensureProperty(ctx, conn, licenceID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("property_id:", propertyID)

	if err := clearPropertyData(ctx, conn, propertyID); err != nil {
		log.Fatal(err)
	}

	roomTypeIDs, err := seedRoomTypes(ctx, conn, propertyID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("room_types: %d seeded\n", len(roomTypeIDs))

	roomIDs, err := seedRooms(ctx, conn, propertyID, roomTypeIDs, roomsPerType)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("rooms: %d seeded\n", len(roomIDs))

	if err := seedInventory(ctx, conn, propertyID, roomIDs, roomTypeIDs, roomsPerType, ledgerStartOffset, ledgerDays); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("inventory: %d days x %d rooms\n", ledgerDays, len(roomIDs))

	ratePlanID, err := seedRatePlan(ctx, conn, propertyID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("rate_plan_id:", ratePlanID)

	guestIDs, err := seedGuests(ctx, conn, propertyID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("guests: %d seeded\n", len(guestIDs))

	maintRoomIndices := maintenanceRoomIndices(roomsPerType)
	blockedRanges := maintenanceBlockedRanges(maintRoomIndices)

	count, err := seedReservations(ctx, conn, propertyID, roomTypeIDs, roomIDs, roomsPerType, guestIDs, ratePlanID, blockedRanges)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("reservations: %d seeded\n", count)

	blockCount, err := seedMaintenanceBlocks(ctx, conn, propertyID, roomIDs, maintRoomIndices)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("maintenance_blocks: %d seeded\n", blockCount)

	fmt.Println("\n--- DONE ---")
	fmt.Printf("Add to web/.env: VITE_DEV_PROPERTY_ID=%s\n", propertyID)
}
