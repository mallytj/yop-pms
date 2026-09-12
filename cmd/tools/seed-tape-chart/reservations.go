package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
)

// The stay window mirrors the seeded ledger window (ledgerStartOffset,
// ledgerDays in main.go) so every ledger day is coverable by a stay.
const (
	minGapDays      = 0
	maxGapDays      = 2
	minStayNights   = 1
	maxStayNights   = 7
	doNotMoveEveryN = 4
	holdEveryN      = 5

	// reservationRandomSeed fixes the generated stay pattern so reseeds
	// produce comparable occupancy shapes.
	reservationRandomSeed = 42
)

// nightlyRatePenceByTypeIndex mirrors the roomTypeSeeds order in roomtypes.go.
var nightlyRatePenceByTypeIndex = []int{8000, 12000, 11000, 20000, 25000}

var reservationSources = []string{"website", "internal", "ota"}

type reservationSeed struct {
	guestIndex    int
	source        string
	status        string
	notes         string
	expiresAt     any
	checkIn       time.Time
	checkOut      time.Time
	roomTypeIndex int
	roomIndex     int
	nightlyPence  int
	adultsCount   int
	childrenCount int
	itemStatus    string
	doNotMove     bool
}

// SeedReservationsParams bundles every dependency the reservation generator
// needs, so new fixture dimensions extend this struct instead of every
// signature in the call chain.
type SeedReservationsParams struct {
	PropertyID    string
	RoomTypeIDs   []string
	RoomIDs       []string
	RoomsPerType  int
	GuestIDs      []string
	RatePlanID    string
	BlockedRanges map[int][2]time.Time
	// Rand drives gap and stay-length selection; the caller picks the seed
	// so the generator cannot hide how deterministic a run is.
	Rand *rand.Rand
}

func seedReservations(ctx context.Context, conn *pgx.Conn, params SeedReservationsParams) (int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	seeds := generateReservationSeeds(params, today)

	for _, seed := range seeds {
		if err := insertReservation(ctx, conn, params, seed); err != nil {
			return 0, err
		}
	}
	return len(seeds), nil
}

// generateReservationSeeds fills each room's stay window with back-to-back
// bookings separated by short gaps (0-2 nights) and varying durations (1-7
// nights), which averages to roughly 80% occupancy while still leaving
// visible gaps on the tape chart. Rooms with a maintenance block (see
// maintenanceBlockedRanges) stay bookable outside their blocked days.
func generateReservationSeeds(params SeedReservationsParams, today time.Time) []reservationSeed {
	windowStart := today.AddDate(0, 0, ledgerStartOffset)
	windowEnd := today.AddDate(0, 0, ledgerStartOffset+ledgerDays)

	seeds := make([]reservationSeed, 0, len(params.RoomIDs)*4)

	for roomIndex := range params.RoomIDs {
		roomTypeIndex := roomIndex / params.RoomsPerType
		blocked, hasBlock := params.BlockedRanges[roomIndex]

		cursor := windowStart
		stayIndex := 0
		for cursor.Before(windowEnd) {
			gap := minGapDays + params.Rand.Intn(maxGapDays-minGapDays+1)
			cursor = cursor.AddDate(0, 0, gap)
			if !cursor.Before(windowEnd) {
				break
			}

			if hasBlock && cursor.Before(blocked[1]) && blocked[0].Before(cursor.AddDate(0, 0, 1)) {
				cursor = blocked[1]
				continue
			}

			nights := minStayNights + params.Rand.Intn(maxStayNights-minStayNights+1)
			checkOutDay := cursor.AddDate(0, 0, nights)
			if checkOutDay.After(windowEnd) {
				checkOutDay = windowEnd
			}
			if hasBlock && checkOutDay.After(blocked[0]) && blocked[0].After(cursor) {
				checkOutDay = blocked[0]
			}
			if !checkOutDay.After(cursor) {
				break
			}

			seeds = append(seeds, buildReservationSeed(params, roomIndex, roomTypeIndex, stayIndex, cursor, checkOutDay, today))

			cursor = checkOutDay
			stayIndex++
		}
	}
	return seeds
}

func buildReservationSeed(
	params SeedReservationsParams,
	roomIndex, roomTypeIndex, stayIndex int,
	stayStart, stayEnd, today time.Time,
) reservationSeed {
	status, itemStatus, expiresAt := deriveReservationStatus(stayStart, stayEnd, today, stayIndex)
	roomType := roomTypeSeeds[roomTypeIndex]

	return reservationSeed{
		guestIndex:    (roomIndex*7 + stayIndex) % len(params.GuestIDs),
		source:        reservationSources[stayIndex%len(reservationSources)],
		status:        status,
		notes:         "",
		expiresAt:     expiresAt,
		checkIn:       stayStart.Add(13 * time.Hour),
		checkOut:      stayEnd.Add(10 * time.Hour),
		roomTypeIndex: roomTypeIndex,
		roomIndex:     roomIndex,
		nightlyPence:  nightlyRatePenceByTypeIndex[roomTypeIndex],
		adultsCount:   roomType.stdOccupancy,
		childrenCount: 0,
		itemStatus:    itemStatus,
		doNotMove:     stayIndex%doNotMoveEveryN == doNotMoveEveryN-1,
	}
}

func deriveReservationStatus(stayStart, stayEnd, today time.Time, stayIndex int) (status, itemStatus string, expiresAt any) {
	switch {
	case !stayEnd.After(today):
		return "checked_out", "checked_out", nil
	case !stayStart.After(today) && stayEnd.After(today):
		return "checked_in", "checked_in", nil
	case stayIndex%holdEveryN == holdEveryN-1:
		return "hold", "booked", today.AddDate(0, 0, 5).Add(13 * time.Hour)
	default:
		return "confirmed", "booked", nil
	}
}

func insertReservation(ctx context.Context, conn *pgx.Conn, params SeedReservationsParams, seed reservationSeed) error {
	nights := int(seed.checkOut.Truncate(24*time.Hour).Sub(seed.checkIn.Truncate(24*time.Hour)).Hours() / 24)
	if nights <= 0 {
		nights = 1
	}
	totalPence := seed.nightlyPence * nights

	var reservationID string
	err := conn.QueryRow(ctx, `
		INSERT INTO operations.reservations
			(property_id, primary_guest_id, stay_period_envelope, source, notes, status, version, expires_at)
		VALUES ($1, $2, tstzrange($3::timestamptz, $4::timestamptz, '[)'),
		        $5, NULLIF($6, ''), $7::operations.reservation_status, 1, $8)
		RETURNING id
	`, params.PropertyID, params.GuestIDs[seed.guestIndex], seed.checkIn, seed.checkOut,
		seed.source, seed.notes, seed.status, seed.expiresAt).Scan(&reservationID)
	if err != nil {
		return fmt.Errorf("reservation insert: %w", err)
	}

	var itemID string
	err = conn.QueryRow(ctx, `
		INSERT INTO operations.reservation_items
			(property_id, reservation_id, booked_room_type_id, assigned_room_id, guest_id,
			 rate_plan_id, stay_period, base_rate_pence, adults_count, children_count,
			 status, version, do_not_move)
		VALUES ($1, $2, $3, $4, $5, $6,
		        tstzrange($7::timestamptz, $8::timestamptz, '[)'),
		        $9, $10, $11, $12::operations.reservation_item_status, 1, $13)
		RETURNING id
	`, params.PropertyID, reservationID, params.RoomTypeIDs[seed.roomTypeIndex], params.RoomIDs[seed.roomIndex],
		params.GuestIDs[seed.guestIndex], params.RatePlanID, seed.checkIn, seed.checkOut,
		totalPence, seed.adultsCount, seed.childrenCount, seed.itemStatus, seed.doNotMove).Scan(&itemID)
	if err != nil {
		return fmt.Errorf("reservation_item insert: %w", err)
	}

	inventoryStatus := "sold"
	if seed.status == "hold" {
		inventoryStatus = "on_hold"
	}
	for day := seed.checkIn.Truncate(24 * time.Hour); day.Before(seed.checkOut.Truncate(24 * time.Hour)); day = day.AddDate(0, 0, 1) {
		_, err = conn.Exec(ctx, `
			UPDATE inventory.room_inventory_ledger
			SET status = $1, reservation_id = $2, reservation_item_id = $3, updated_at = NOW()
			WHERE room_id = $4 AND calendar_date = $5 AND property_id = $6 AND deleted_at IS NULL
		`, inventoryStatus, reservationID, itemID, params.RoomIDs[seed.roomIndex], day.Format("2006-01-02"), params.PropertyID)
		if err != nil {
			return fmt.Errorf("inventory ledger update: %w", err)
		}
	}
	return nil
}
