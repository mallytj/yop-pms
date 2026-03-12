import type { UUID } from "./booking_engine";

// --- Types ---
interface Booking {
  reservation_id: string;
  reservation_item_id: string;
  guest_id: string;
  guest_name: string;
  check_in_date: string;
  check_out_date: string;
  item_status?: string;
  item_status_code?: string;
  status_color?: string;
  stay_price_pence?: number;
}

interface RoomType {
  room_type_id: string;
  room_type_code: string;
  room_type_name: string;
  room_type_description: string;
}

interface Room {
  id: UUID;
  room_id: string;
  room_name: string;
  room_type_id: UUID;
  room_type_code: string;
  reservations: Booking[];
}

interface PlannerData {
  start_date: string;
  end_date: string;
  rooms: Room[];
}

type DragMode = "move" | "resize" | "select-cells";

interface DragState {
  id: string;
  item_id?: string;
  mode: DragMode;

  // Selection tracking
  startRoomIndex?: number;
  startDayIndex?: number;
  currentRoomIndex?: number;
  currentDayIndex?: number;
  shadowRoomSpan?: number;

  // Floating "Ghost" Visuals (Pixels)
  currentX: number;
  currentY: number;
  width: number;
  height: number;

  // Snapping "Shadow" Data (Grid Units)
  shadowStartOffset: number;
  shadowRowIndex: number;
  shadowDuration: number;

  // Math Baselines
  startX: number;
  startY: number;
  initialStart: Date;
  initialEnd: Date;
  initialRoomIndex: number;
}

interface ReservationDraft {
  roomId: string;
  checkInDate: string;
  checkOutDate: string;
}

interface CellSelection {
  startRoom: number;
  endRoom: number;
  startDay: number;
  endDay: number;
}

export type {
  Booking,
  Room,
  PlannerData,
  DragState,
  DragMode,
  ReservationDraft,
  CellSelection,
  RoomType,
};
