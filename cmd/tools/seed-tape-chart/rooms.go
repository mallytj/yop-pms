package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// maintenanceRoomIndices picks the last room in the Suite and Family blocks
// to carry a maintenance block. The reservation generator (see
// maintenanceBlockedRanges) avoids booking these rooms only during their
// blocked days, so they stay bookable the rest of the window.
func maintenanceRoomIndices(roomsPerType int) []int {
	const suiteTypeIndex = 3
	const familyTypeIndex = 4
	return []int{
		suiteTypeIndex*roomsPerType + (roomsPerType - 1),
		familyTypeIndex*roomsPerType + (roomsPerType - 1),
	}
}

// seedRooms assumes clearPropertyData already wiped any prior rooms for
// this property, so it always inserts fresh rows.
func seedRooms(ctx context.Context, conn *pgx.Conn, propertyID string, roomTypeIDs []string, roomsPerType int) ([]string, error) {
	roomIDs := make([]string, 0, len(roomTypeIDs)*roomsPerType)

	for typeIndex, roomType := range roomTypeSeeds {
		for roomNumber := 1; roomNumber <= roomsPerType; roomNumber++ {
			var roomID string
			roomName := fmt.Sprintf("%c%02d", roomType.name[0]+'a'-'A', typeIndex*roomsPerType+roomNumber)
			err := conn.QueryRow(ctx, `
				INSERT INTO inventory.rooms (property_id, room_type_id, name, housekeeping_status, occupancy_status)
				VALUES ($1, $2, $3, 'clean', 'vacant')
				RETURNING id
			`, propertyID, roomTypeIDs[typeIndex], roomName).Scan(&roomID)
			if err != nil {
				return nil, fmt.Errorf("room %s: %w", roomName, err)
			}
			roomIDs = append(roomIDs, roomID)
		}
	}
	return roomIDs, nil
}
