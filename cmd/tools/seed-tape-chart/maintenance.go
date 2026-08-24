package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type maintenanceBlockSeed struct {
	roomIndex   int
	blockType   string
	reason      string
	startDay    int
	blockNights int
}

func maintenanceBlockSeeds() []maintenanceBlockSeed {
	return []maintenanceBlockSeed{
		{roomIndex: 0, blockType: "repair", reason: "Bathroom leak repair", startDay: 2, blockNights: 2},
		{roomIndex: 1, blockType: "deep_clean", reason: "Post-checkout deep clean", startDay: 10, blockNights: 3},
	}
}

// maintenanceBlockedRanges maps each maintenance room's index to the
// day-range [start, end) its block occupies, so the reservation generator
// can skip only those days instead of leaving the room unbooked entirely.
func maintenanceBlockedRanges(roomIndices []int) map[int][2]time.Time {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	blocks := maintenanceBlockSeeds()

	ranges := make(map[int][2]time.Time, len(blocks))
	for i, block := range blocks {
		if i >= len(roomIndices) {
			break
		}
		start := today.AddDate(0, 0, block.startDay)
		end := today.AddDate(0, 0, block.startDay+block.blockNights)
		ranges[roomIndices[i]] = [2]time.Time{start, end}
	}
	return ranges
}

// seedMaintenanceBlocks blocks the maintenance-eligible rooms (see
// maintenanceRoomIndices in rooms.go) for a short period so the tape chart
// shows out-of-service rooms alongside bookings.
func seedMaintenanceBlocks(
	ctx context.Context,
	conn *pgx.Conn,
	propertyID string,
	roomIDs []string,
	excludedRoomIndices []int,
) (int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	blocks := maintenanceBlockSeeds()

	seeded := 0
	for i, block := range blocks {
		if i >= len(excludedRoomIndices) {
			break
		}
		roomID := roomIDs[excludedRoomIndices[i]]
		blockStart := today.AddDate(0, 0, block.startDay).Add(11 * time.Hour)
		blockEnd := today.AddDate(0, 0, block.startDay+block.blockNights).Add(11 * time.Hour)

		var blockID string
		err := conn.QueryRow(ctx, `
			INSERT INTO inventory.maintenance_blocks
				(property_id, room_id, block_period, reason, type)
			VALUES ($1, $2, tstzrange($3::timestamptz, $4::timestamptz, '[)'), $5, $6::inventory.maintenance_block_type)
			RETURNING id
		`, propertyID, roomID, blockStart, blockEnd, block.reason, block.blockType).Scan(&blockID)
		if err != nil {
			return seeded, fmt.Errorf("maintenance_block %s: %w", block.reason, err)
		}

		for day := today.AddDate(0, 0, block.startDay); day.Before(today.AddDate(0, 0, block.startDay+block.blockNights)); day = day.AddDate(0, 0, 1) {
			_, err = conn.Exec(ctx, `
				UPDATE inventory.room_inventory_ledger
				SET status = 'maintenance', maintenance_block_id = $1, updated_at = NOW()
				WHERE room_id = $2 AND calendar_date = $3 AND property_id = $4 AND deleted_at IS NULL
			`, blockID, roomID, day.Format("2006-01-02"), propertyID)
			if err != nil {
				return seeded, fmt.Errorf("maintenance ledger update: %w", err)
			}
		}
		seeded++
	}
	return seeded, nil
}
