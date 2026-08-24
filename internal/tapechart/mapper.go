package tapechart

import (
	"hash/fnv"

	"github.com/google/uuid"

	"github.com/lexxcode1/yop-pms/internal/platform/types"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// accentIndex derives a stable accent-palette index from a reservation ID so
// the same reservation always renders with the same colour across requests.
func accentIndex(reservationID uuid.UUID) int {
	h := fnv.New32a()
	_, _ = h.Write(reservationID[:])
	return int(h.Sum32() % accentPaletteSize)
}

// buildReservationBlocks maps each room-assigned reservation item to the
// room-level block the grid renders, carrying its parent reservation's
// status and accent colour.
func buildReservationBlocks(
	items []store.GetTapeChartReservationItemsRow,
	reservations []store.GetTapeChartReservationsRow,
	accentByReservation map[uuid.UUID]int,
) []ReservationBlock {
	reservationByID := make(map[uuid.UUID]store.GetTapeChartReservationsRow, len(reservations))
	for _, reservation := range reservations {
		reservationByID[reservation.ID] = reservation
	}

	blocks := make([]ReservationBlock, 0, len(items))
	for _, item := range items {
		if !item.AssignedRoomID.Valid {
			continue
		}
		reservation, ok := reservationByID[item.ReservationID]
		if !ok {
			continue
		}
		var pricePence *int32
		if item.BaseRatePence != 0 {
			price := item.BaseRatePence
			pricePence = &price
		}
		blocks = append(blocks, ReservationBlock{
			ID:          item.ID,
			RoomID:      item.AssignedRoomID.UUID,
			From:        types.ISO8601Date{Time: item.StayPeriod.Lower.Time},
			To:          types.ISO8601Date{Time: item.StayPeriod.Upper.Time},
			Status:      string(reservation.Status),
			AccentIndex: accentByReservation[reservation.ID],
			GuestName:   reservation.GuestName,
			Code:        reservation.Code,
			PricePence:  pricePence,
		})
	}
	return blocks
}

// buildMaintenanceBlocks maps maintenance block rows to the grid DTO,
// carrying the operator-authored reason text (not the block-type enum).
func buildMaintenanceBlocks(rows []store.GetTapeChartMaintenanceBlocksRow) []MaintenanceBlock {
	blocks := make([]MaintenanceBlock, 0, len(rows))
	for _, row := range rows {
		blocks = append(blocks, MaintenanceBlock{
			ID:     row.ID,
			RoomID: row.RoomID,
			Reason: row.Reason,
			Start:  types.ISO8601Date{Time: row.StartDate.Time},
			End:    types.ISO8601Date{Time: row.EndDate.Time},
		})
	}
	return blocks
}

// buildInventoryDays maps inventory ledger rows to the grid DTO.
func buildInventoryDays(rows []store.GetTapeChartInventoryRow) []InventoryDay {
	days := make([]InventoryDay, 0, len(rows))
	for _, row := range rows {
		days = append(days, InventoryDay{
			RoomID:       row.RoomID,
			CalendarDate: types.ISO8601Date{Time: row.CalendarDate.Time},
			Status:       string(row.Status),
		})
	}
	return days
}

// nestRoomTypes groups rooms under their room type and each room's blocks
// under the room, per the full-response contract.
func nestRoomTypes(
	roomTypes []store.GetTapeChartRoomTypesRow,
	rooms []store.GetTapeChartRoomsRow,
	reservationBlocks []ReservationBlock,
	maintenanceBlocks []MaintenanceBlock,
) []RoomTypeNode {
	reservationsByRoom := make(map[uuid.UUID][]ReservationBlock, len(rooms))
	for _, block := range reservationBlocks {
		reservationsByRoom[block.RoomID] = append(reservationsByRoom[block.RoomID], block)
	}
	maintenanceByRoom := make(map[uuid.UUID][]MaintenanceBlock, len(rooms))
	for _, block := range maintenanceBlocks {
		maintenanceByRoom[block.RoomID] = append(maintenanceByRoom[block.RoomID], block)
	}

	roomsByType := make(map[uuid.UUID][]store.GetTapeChartRoomsRow, len(roomTypes))
	for _, room := range rooms {
		roomsByType[room.RoomTypeID] = append(roomsByType[room.RoomTypeID], room)
	}

	nodes := make([]RoomTypeNode, 0, len(roomTypes))
	for _, roomType := range roomTypes {
		roomNodes := make([]RoomNode, 0, len(roomsByType[roomType.ID]))
		for _, room := range roomsByType[roomType.ID] {
			roomNodes = append(roomNodes, RoomNode{
				ID:                room.ID,
				Name:              room.Name,
				Reservations:      reservationsByRoom[room.ID],
				MaintenanceBlocks: maintenanceByRoom[room.ID],
			})
		}
		nodes = append(nodes, RoomTypeNode{
			ID:    roomType.ID,
			Name:  roomType.Name,
			Rooms: roomNodes,
		})
	}
	return nodes
}
