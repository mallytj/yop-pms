package tapechart

import (
	"github.com/google/uuid"

	"github.com/lexxcode1/yop-pms/internal/platform/types"
)

// IncludeMode selects how much of the grid GetTapeChart returns.
type IncludeMode string

const (
	// IncludeShallow omits room/room-type metadata (client already has it cached).
	IncludeShallow IncludeMode = "shallow"
	// IncludeFull returns room/room-type metadata alongside inventory data.
	IncludeFull IncludeMode = "full"
)

// accentPaletteSize is the number of semantic --color-reservation-accent-N
// tokens the frontend defines (web/src/app.css) to cycle reservation colours
// through.
const accentPaletteSize = 8

// TapeChartResponse is the GET /v1/tape-chart response envelope.
type TapeChartResponse struct {
	Data TapeChartData `json:"data"`
}

// TapeChartData is the room-availability grid for a date-range window.
//
// On IncludeFull, RoomTypes nests rooms which nest their own reservation and
// maintenance blocks. On IncludeShallow, the client already holds room/room-type
// metadata, so blocks are returned as flat arrays instead.
type TapeChartData struct {
	From              types.ISO8601Date  `json:"from"`
	To                types.ISO8601Date  `json:"to"`
	RoomTypes         []RoomTypeNode     `json:"room_types,omitempty"`
	Reservations      []ReservationBlock `json:"reservations,omitempty"`
	MaintenanceBlocks []MaintenanceBlock `json:"maintenance_blocks,omitempty"`
	Inventory         []InventoryDay     `json:"inventory"`
}

// RoomTypeNode is a room type grouping rooms in the nested (full) response.
type RoomTypeNode struct {
	ID    uuid.UUID  `json:"id"`
	Name  string     `json:"name"`
	Rooms []RoomNode `json:"rooms"`
}

// RoomNode is a single room and its blocks in the nested (full) response.
type RoomNode struct {
	ID                uuid.UUID          `json:"id"`
	Name              string             `json:"name"`
	Reservations      []ReservationBlock `json:"reservations"`
	MaintenanceBlocks []MaintenanceBlock `json:"maintenance_blocks"`
}

// ReservationBlock is a single room's occupied span within a reservation.
// AccentIndex is a stable, deterministic index (0..accentPaletteSize-1) the
// frontend maps to a --color-reservation-accent-N token, so every block
// belonging to the same reservation renders with the same accent colour.
type ReservationBlock struct {
	ID          uuid.UUID         `json:"id"`
	RoomID      uuid.UUID         `json:"room_id"`
	From        types.ISO8601Date `json:"from"`
	To          types.ISO8601Date `json:"to"`
	Status      string            `json:"status"`
	AccentIndex int               `json:"accent_index"`
	GuestName   string            `json:"guest_name"`
	Code        string            `json:"code"`
	PricePence  *int32            `json:"price_pence,omitempty"`
}

// MaintenanceBlock is a room's out-of-service span.
type MaintenanceBlock struct {
	ID     uuid.UUID         `json:"id"`
	RoomID uuid.UUID         `json:"room_id"`
	Reason string            `json:"reason"`
	Start  types.ISO8601Date `json:"start"`
	End    types.ISO8601Date `json:"end"`
}

// InventoryDay is one room's ledger status for one calendar day. It carries
// statuses (e.g. decommissioned) that have no reservation or maintenance
// block of their own, so the grid can't derive them from those blocks alone.
type InventoryDay struct {
	RoomID       uuid.UUID         `json:"room_id"`
	CalendarDate types.ISO8601Date `json:"calendar_date"`
	Status       string            `json:"status"`
}
