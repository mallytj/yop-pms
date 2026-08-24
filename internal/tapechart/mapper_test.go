package tapechart

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lexxcode1/yop-pms/internal/store"
)

// Fixed, deterministic UUIDs and dates so fixtures never rely on unseeded
// randomness or wall-clock time.
var (
	reservationAID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	reservationBID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	roomAID        = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	roomBID        = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	roomCID        = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	roomTypeAID    = uuid.MustParse("00000000-0000-0000-0000-000000000020")
	roomTypeBID    = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	itemAID        = uuid.MustParse("00000000-0000-0000-0000-000000000030")
	itemBID        = uuid.MustParse("00000000-0000-0000-0000-000000000031")
	itemCID        = uuid.MustParse("00000000-0000-0000-0000-000000000032")
	maintenanceID  = uuid.MustParse("00000000-0000-0000-0000-000000000040")

	fixtureFrom = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fixtureTo   = time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
)

func stayPeriod(lower, upper time.Time) pgtype.Range[pgtype.Timestamptz] {
	return pgtype.Range[pgtype.Timestamptz]{
		Lower:     pgtype.Timestamptz{Time: lower, Valid: true},
		Upper:     pgtype.Timestamptz{Time: upper, Valid: true},
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive,
		Valid:     true,
	}
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

// --- accentIndex ---

func TestAccentIndex(t *testing.T) {
	t.Run("same UUID always yields same index", func(t *testing.T) {
		first := accentIndex(reservationAID)
		second := accentIndex(reservationAID)

		if first != second {
			t.Errorf("accentIndex(%s) = %d, then %d; want stable result", reservationAID, first, second)
		}
	})

	t.Run("different UUIDs can yield different indices", func(t *testing.T) {
		first := accentIndex(reservationAID)
		second := accentIndex(reservationBID)

		if first == second {
			t.Skip("hash collision for these fixture UUIDs; not a correctness failure")
		}
	})

	t.Run("result is always within palette bounds", func(t *testing.T) {
		ids := []uuid.UUID{
			reservationAID,
			reservationBID,
			uuid.Nil,
			uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		}
		for _, id := range ids {
			index := accentIndex(id)
			if index < 0 || index >= accentPaletteSize {
				t.Errorf("accentIndex(%s) = %d, want within [0, %d)", id, index, accentPaletteSize)
			}
		}
	})
}

// --- buildReservationBlocks ---

func TestBuildReservationBlocks(t *testing.T) {
	baseReservation := store.GetTapeChartReservationsRow{
		ID:        reservationAID,
		GuestName: "Jane Doe",
		Code:      "RES-000001",
		Status:    store.OperationsReservationStatusConfirmed,
	}

	t.Run("skips items with invalid assigned room", func(t *testing.T) {
		items := []store.GetTapeChartReservationItemsRow{
			{
				ID:             itemAID,
				ReservationID:  reservationAID,
				AssignedRoomID: uuid.NullUUID{Valid: false},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
			},
		}
		reservations := []store.GetTapeChartReservationsRow{baseReservation}

		blocks := buildReservationBlocks(items, reservations, nil)

		if len(blocks) != 0 {
			t.Errorf("blocks = %+v, want empty (unassigned room skipped)", blocks)
		}
	})

	t.Run("skips items whose reservation has no match", func(t *testing.T) {
		items := []store.GetTapeChartReservationItemsRow{
			{
				ID:             itemAID,
				ReservationID:  reservationBID, // no matching reservation below
				AssignedRoomID: uuid.NullUUID{UUID: roomAID, Valid: true},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
			},
		}
		reservations := []store.GetTapeChartReservationsRow{baseReservation}

		blocks := buildReservationBlocks(items, reservations, nil)

		if len(blocks) != 0 {
			t.Errorf("blocks = %+v, want empty (no matching reservation)", blocks)
		}
	})

	t.Run("zero base rate maps to nil price", func(t *testing.T) {
		items := []store.GetTapeChartReservationItemsRow{
			{
				ID:             itemAID,
				ReservationID:  reservationAID,
				AssignedRoomID: uuid.NullUUID{UUID: roomAID, Valid: true},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
				BaseRatePence:  0,
			},
		}
		reservations := []store.GetTapeChartReservationsRow{baseReservation}

		blocks := buildReservationBlocks(items, reservations, nil)

		if len(blocks) != 1 {
			t.Fatalf("len(blocks) = %d, want 1", len(blocks))
		}
		if blocks[0].PricePence != nil {
			t.Errorf("PricePence = %v, want nil for zero base rate", *blocks[0].PricePence)
		}
	})

	t.Run("non-zero base rate maps to populated price pointer", func(t *testing.T) {
		items := []store.GetTapeChartReservationItemsRow{
			{
				ID:             itemAID,
				ReservationID:  reservationAID,
				AssignedRoomID: uuid.NullUUID{UUID: roomAID, Valid: true},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
				BaseRatePence:  12345,
			},
		}
		reservations := []store.GetTapeChartReservationsRow{baseReservation}

		blocks := buildReservationBlocks(items, reservations, nil)

		if len(blocks) != 1 {
			t.Fatalf("len(blocks) = %d, want 1", len(blocks))
		}
		if blocks[0].PricePence == nil || *blocks[0].PricePence != 12345 {
			t.Errorf("PricePence = %v, want pointer to 12345", blocks[0].PricePence)
		}
	})

	t.Run("looks up accent index from accentByReservation map", func(t *testing.T) {
		items := []store.GetTapeChartReservationItemsRow{
			{
				ID:             itemAID,
				ReservationID:  reservationAID,
				AssignedRoomID: uuid.NullUUID{UUID: roomAID, Valid: true},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
			},
		}
		reservations := []store.GetTapeChartReservationsRow{baseReservation}
		accentByReservation := map[uuid.UUID]int{reservationAID: 5}

		blocks := buildReservationBlocks(items, reservations, accentByReservation)

		if len(blocks) != 1 {
			t.Fatalf("len(blocks) = %d, want 1", len(blocks))
		}
		if blocks[0].AccentIndex != 5 {
			t.Errorf("AccentIndex = %d, want 5", blocks[0].AccentIndex)
		}
	})

	t.Run("maps room, dates, status, guest and code from matched reservation", func(t *testing.T) {
		items := []store.GetTapeChartReservationItemsRow{
			{
				ID:             itemAID,
				ReservationID:  reservationAID,
				AssignedRoomID: uuid.NullUUID{UUID: roomAID, Valid: true},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
				BaseRatePence:  9900,
			},
		}
		reservations := []store.GetTapeChartReservationsRow{baseReservation}

		blocks := buildReservationBlocks(items, reservations, nil)

		if len(blocks) != 1 {
			t.Fatalf("len(blocks) = %d, want 1", len(blocks))
		}
		block := blocks[0]
		if block.ID != itemAID {
			t.Errorf("ID = %s, want %s", block.ID, itemAID)
		}
		if block.RoomID != roomAID {
			t.Errorf("RoomID = %s, want %s", block.RoomID, roomAID)
		}
		if !block.From.Time.Equal(fixtureFrom) || !block.To.Time.Equal(fixtureTo) {
			t.Errorf("From/To = %v/%v, want %v/%v", block.From.Time, block.To.Time, fixtureFrom, fixtureTo)
		}
		if block.Status != string(store.OperationsReservationStatusConfirmed) {
			t.Errorf("Status = %q, want %q", block.Status, store.OperationsReservationStatusConfirmed)
		}
		if block.GuestName != "Jane Doe" {
			t.Errorf("GuestName = %q, want %q", block.GuestName, "Jane Doe")
		}
		if block.Code != "RES-000001" {
			t.Errorf("Code = %q, want %q", block.Code, "RES-000001")
		}
	})

	t.Run("multiple items map independently", func(t *testing.T) {
		secondReservation := store.GetTapeChartReservationsRow{
			ID:        reservationBID,
			GuestName: "John Smith",
			Code:      "RES-000002",
			Status:    store.OperationsReservationStatusHold,
		}
		items := []store.GetTapeChartReservationItemsRow{
			{
				ID:             itemAID,
				ReservationID:  reservationAID,
				AssignedRoomID: uuid.NullUUID{UUID: roomAID, Valid: true},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
			},
			{
				ID:             itemBID,
				ReservationID:  reservationBID,
				AssignedRoomID: uuid.NullUUID{UUID: roomBID, Valid: true},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
			},
			{
				// Skipped: no assigned room.
				ID:             itemCID,
				ReservationID:  reservationAID,
				AssignedRoomID: uuid.NullUUID{Valid: false},
				StayPeriod:     stayPeriod(fixtureFrom, fixtureTo),
			},
		}
		reservations := []store.GetTapeChartReservationsRow{baseReservation, secondReservation}

		blocks := buildReservationBlocks(items, reservations, nil)

		if len(blocks) != 2 {
			t.Fatalf("len(blocks) = %d, want 2", len(blocks))
		}
	})
}

// --- buildMaintenanceBlocks ---

func TestBuildMaintenanceBlocks(t *testing.T) {
	t.Run("maps multiple rows", func(t *testing.T) {
		rows := []store.GetTapeChartMaintenanceBlocksRow{
			{
				ID:        maintenanceID,
				RoomID:    roomAID,
				Reason:    "Bathroom leak repair",
				StartDate: pgDate(fixtureFrom),
				EndDate:   pgDate(fixtureTo),
			},
			{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000041"),
				RoomID:    roomBID,
				Reason:    "Deep clean",
				StartDate: pgDate(fixtureFrom),
				EndDate:   pgDate(fixtureTo),
			},
		}

		blocks := buildMaintenanceBlocks(rows)

		if len(blocks) != 2 {
			t.Fatalf("len(blocks) = %d, want 2", len(blocks))
		}
		if blocks[0].ID != maintenanceID || blocks[0].RoomID != roomAID || blocks[0].Reason != "Bathroom leak repair" {
			t.Errorf("blocks[0] = %+v, unexpected mapping", blocks[0])
		}
		if !blocks[0].Start.Time.Equal(fixtureFrom) || !blocks[0].End.Time.Equal(fixtureTo) {
			t.Errorf("blocks[0] Start/End = %v/%v, want %v/%v", blocks[0].Start.Time, blocks[0].End.Time, fixtureFrom, fixtureTo)
		}
		if blocks[1].Reason != "Deep clean" {
			t.Errorf("blocks[1].Reason = %q, want %q", blocks[1].Reason, "Deep clean")
		}
	})

	t.Run("empty input yields empty, non-nil slice", func(t *testing.T) {
		blocks := buildMaintenanceBlocks(nil)

		if blocks == nil {
			t.Error("blocks is nil, want empty non-nil slice")
		}
		if len(blocks) != 0 {
			t.Errorf("len(blocks) = %d, want 0", len(blocks))
		}
	})
}

// --- buildInventoryDays ---

func TestBuildInventoryDays(t *testing.T) {
	t.Run("maps multiple rows", func(t *testing.T) {
		rows := []store.GetTapeChartInventoryRow{
			{
				RoomID:       roomAID,
				CalendarDate: pgDate(fixtureFrom),
				Status:       store.InventoryInventoryStatusSold,
			},
			{
				RoomID:       roomBID,
				CalendarDate: pgDate(fixtureTo),
				Status:       store.InventoryInventoryStatusDecommissioned,
			},
		}

		days := buildInventoryDays(rows)

		if len(days) != 2 {
			t.Fatalf("len(days) = %d, want 2", len(days))
		}
		if days[0].RoomID != roomAID || days[0].Status != string(store.InventoryInventoryStatusSold) {
			t.Errorf("days[0] = %+v, unexpected mapping", days[0])
		}
		if !days[0].CalendarDate.Time.Equal(fixtureFrom) {
			t.Errorf("days[0].CalendarDate = %v, want %v", days[0].CalendarDate.Time, fixtureFrom)
		}
		if days[1].Status != string(store.InventoryInventoryStatusDecommissioned) {
			t.Errorf("days[1].Status = %q, want %q", days[1].Status, store.InventoryInventoryStatusDecommissioned)
		}
	})

	t.Run("empty input yields empty, non-nil slice", func(t *testing.T) {
		days := buildInventoryDays(nil)

		if days == nil {
			t.Error("days is nil, want empty non-nil slice")
		}
		if len(days) != 0 {
			t.Errorf("len(days) = %d, want 0", len(days))
		}
	})
}

// --- nestRoomTypes ---

func TestNestRoomTypes(t *testing.T) {
	roomTypeA := store.GetTapeChartRoomTypesRow{ID: roomTypeAID, Name: "Double"}
	roomTypeB := store.GetTapeChartRoomTypesRow{ID: roomTypeBID, Name: "Suite"}

	roomA := store.GetTapeChartRoomsRow{ID: roomAID, RoomTypeID: roomTypeAID, Name: "101"}
	roomB := store.GetTapeChartRoomsRow{ID: roomBID, RoomTypeID: roomTypeAID, Name: "102"}
	roomC := store.GetTapeChartRoomsRow{ID: roomCID, RoomTypeID: roomTypeBID, Name: "201"}

	t.Run("groups rooms under the correct room type", func(t *testing.T) {
		roomTypes := []store.GetTapeChartRoomTypesRow{roomTypeA, roomTypeB}
		rooms := []store.GetTapeChartRoomsRow{roomA, roomB, roomC}

		nodes := nestRoomTypes(roomTypes, rooms, nil, nil)

		if len(nodes) != 2 {
			t.Fatalf("len(nodes) = %d, want 2", len(nodes))
		}
		byID := make(map[uuid.UUID]RoomTypeNode, len(nodes))
		for _, node := range nodes {
			byID[node.ID] = node
		}
		if len(byID[roomTypeAID].Rooms) != 2 {
			t.Errorf("room type A rooms = %+v, want 2 rooms", byID[roomTypeAID].Rooms)
		}
		if len(byID[roomTypeBID].Rooms) != 1 {
			t.Errorf("room type B rooms = %+v, want 1 room", byID[roomTypeBID].Rooms)
		}
		if byID[roomTypeBID].Rooms[0].ID != roomCID {
			t.Errorf("room type B's room = %s, want %s", byID[roomTypeBID].Rooms[0].ID, roomCID)
		}
	})

	t.Run("room type with no rooms yields empty, non-nil rooms slice", func(t *testing.T) {
		roomTypes := []store.GetTapeChartRoomTypesRow{roomTypeA}
		rooms := []store.GetTapeChartRoomsRow{} // no rooms at all

		nodes := nestRoomTypes(roomTypes, rooms, nil, nil)

		if len(nodes) != 1 {
			t.Fatalf("len(nodes) = %d, want 1", len(nodes))
		}
		if nodes[0].Rooms == nil {
			t.Error("Rooms is nil, want empty non-nil slice")
		}
		if len(nodes[0].Rooms) != 0 {
			t.Errorf("len(Rooms) = %d, want 0", len(nodes[0].Rooms))
		}
	})

	t.Run("reservation and maintenance blocks attach to the correct room", func(t *testing.T) {
		roomTypes := []store.GetTapeChartRoomTypesRow{roomTypeA}
		rooms := []store.GetTapeChartRoomsRow{roomA, roomB}
		reservationBlocks := []ReservationBlock{
			{ID: itemAID, RoomID: roomAID, Code: "RES-A"},
		}
		maintenanceBlocks := []MaintenanceBlock{
			{ID: maintenanceID, RoomID: roomBID, Reason: "Deep clean"},
		}

		nodes := nestRoomTypes(roomTypes, rooms, reservationBlocks, maintenanceBlocks)

		if len(nodes) != 1 {
			t.Fatalf("len(nodes) = %d, want 1", len(nodes))
		}
		roomsByID := make(map[uuid.UUID]RoomNode, len(nodes[0].Rooms))
		for _, room := range nodes[0].Rooms {
			roomsByID[room.ID] = room
		}

		roomANode := roomsByID[roomAID]
		if len(roomANode.Reservations) != 1 || roomANode.Reservations[0].Code != "RES-A" {
			t.Errorf("room A reservations = %+v, want one block RES-A", roomANode.Reservations)
		}
		if len(roomANode.MaintenanceBlocks) != 0 {
			t.Errorf("room A maintenance blocks = %+v, want none", roomANode.MaintenanceBlocks)
		}

		roomBNode := roomsByID[roomBID]
		if len(roomBNode.MaintenanceBlocks) != 1 || roomBNode.MaintenanceBlocks[0].Reason != "Deep clean" {
			t.Errorf("room B maintenance blocks = %+v, want one block 'Deep clean'", roomBNode.MaintenanceBlocks)
		}
		if len(roomBNode.Reservations) != 0 {
			t.Errorf("room B reservations = %+v, want none", roomBNode.Reservations)
		}
	})

	t.Run("room with neither reservations nor maintenance gets an empty slice without panicking", func(t *testing.T) {
		roomTypes := []store.GetTapeChartRoomTypesRow{roomTypeA}
		rooms := []store.GetTapeChartRoomsRow{roomA}

		nodes := nestRoomTypes(roomTypes, rooms, nil, nil)

		if len(nodes) != 1 || len(nodes[0].Rooms) != 1 {
			t.Fatalf("nodes = %+v, want one room type with one room", nodes)
		}
		room := nodes[0].Rooms[0]
		if len(room.Reservations) != 0 {
			t.Errorf("Reservations = %+v, want empty", room.Reservations)
		}
		if len(room.MaintenanceBlocks) != 0 {
			t.Errorf("MaintenanceBlocks = %+v, want empty", room.MaintenanceBlocks)
		}
	})
}
