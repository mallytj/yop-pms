package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type roomTypeSeed struct {
	code         string
	name         string
	stdOccupancy int
	minOccupancy int
	maxOccupancy int
}

// roomTypeSeeds is shared with rooms.go and reservations.go for rate/occupancy lookups.
var roomTypeSeeds = []roomTypeSeed{
	{"SGL", "Single", 1, 1, 1},
	{"DBL", "Double", 2, 1, 2},
	{"TWN", "Twin", 2, 1, 2},
	{"STE", "Suite", 2, 1, 4},
	{"FAM", "Family", 4, 2, 6},
}

// seedRoomTypes assumes clearPropertyData already wiped any prior room
// types for this property, so it always inserts fresh rows.
func seedRoomTypes(ctx context.Context, conn *pgx.Conn, propertyID string) ([]string, error) {
	roomTypeIDs := make([]string, len(roomTypeSeeds))
	for i, roomType := range roomTypeSeeds {
		err := conn.QueryRow(ctx, `
			INSERT INTO inventory.room_types (property_id, name, code, std_occupancy, min_occupancy, max_occupancy)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, propertyID, roomType.name, roomType.code, roomType.stdOccupancy, roomType.minOccupancy, roomType.maxOccupancy).Scan(&roomTypeIDs[i])
		if err != nil {
			return nil, fmt.Errorf("room_type %s: %w", roomType.name, err)
		}
	}
	return roomTypeIDs, nil
}
