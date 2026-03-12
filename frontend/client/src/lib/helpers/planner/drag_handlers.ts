// lib/helpers/dragHandlers.ts
import type {
  DragState,
  CellSelection,
  Room,
} from "../../types/planner_data.ts";
import { parseLocalYMD, diffDays, addDays } from "./planner_utils.ts";
import type { Booking } from "../../types/planner_data.ts";
import { canSelectArea, getSelectableEndDay } from "./overlap.ts";

const COL_WIDTH = 100;
const ROW_HEIGHT = 80;

/**
 * Checks if a booking can be moved to a specific start/end date in a given room.
 *
 * @param {Date} newStart - The new proposed start date.
 * @param {Date} newEnd - The new proposed end date.
 * @param {string} newRoomId - The target room ID.
 * @param {Room[]} rooms - The list of all available rooms.
 * @param {Date} startDate - The starting date of the planner grid.
 * @returns {boolean} True if the proposed move is valid and has no conflicts.
 */
export function canMoveTo(
  newStart: Date,
  newEnd: Date,
  newRoomId: string,
  rooms: Room[],
  startDate: Date,
): boolean {
  const room = rooms.find((r) => r.room_id === newRoomId);
  if (!room) return false;

  // Convert dates to day indices
  const startDayIndex = diffDays(newStart, startDate);
  const endDayIndex = diffDays(newEnd, startDate) - 1;

  // Create a CellSelection to check
  const selection: CellSelection = {
    startRoom: rooms.indexOf(room),
    endRoom: rooms.indexOf(room),
    startDay: startDayIndex,
    endDay: endDayIndex,
  };

  // Use your existing overlap checker
  return canSelectArea(selection, rooms, startDate);
}

/**
 * Initializes the drag state for moving or resizing an existing booking.
 *
 * @param {Booking} booking - The booking being dragged.
 * @param {number} roomIndex - The index of the room containing the booking.
 * @param {Date} startDate - The planner's start date.
 * @param {DOMRect} rect - The DOMRect of the booking element being dragged.
 * @param {DOMRect} parentRect - The DOMRect of the parent container.
 * @param {number} scrollLeft - The current scrollLeft value of the container.
 * @param {number} scrollTop - The current scrollTop value of the container.
 * @param {number} startX - The initial X coordinate of the mouse pointer.
 * @param {number} startY - The initial Y coordinate of the mouse pointer.
 * @returns {DragState} The initialized drag state object.
 */
export function initializeBookingDrag(
  booking: Booking,
  roomIndex: number,
  startDate: Date,
  rect: DOMRect,
  parentRect: DOMRect,
  scrollLeft: number,
  scrollTop: number,
  startX: number,
  startY: number,
): DragState {
  const bStart = parseLocalYMD(booking.check_in_date);
  const bEnd = parseLocalYMD(booking.check_out_date);

  return {
    id: booking.reservation_id,
    item_id: booking.reservation_item_id,
    mode: "move",
    startX,
    startY,

    currentX: rect.left - parentRect.left + scrollLeft,
    currentY: rect.top - parentRect.top + scrollTop,
    width: rect.width,
    height: rect.height,

    initialStart: bStart,
    initialEnd: bEnd,
    initialRoomIndex: roomIndex,
    shadowRowIndex: roomIndex + 2,
    shadowStartOffset: diffDays(bStart, startDate) + 2,
    shadowDuration: Math.max(1, diffDays(bEnd, bStart)),
  };
}

/**
 * Initializes the drag state for selecting empty cells on the grid.
 *
 * @param {number} dayIndex - The starting day index.
 * @param {number} roomIndex - The starting room index.
 * @param {Date} startDate - The start date of the cell.
 * @param {Date} endDate - The end date of the cell.
 * @param {number} startX - The initial X coordinate of the mouse pointer.
 * @param {number} startY - The initial Y coordinate of the mouse pointer.
 * @returns {DragState} The initialized drag state corresponding to cell selection.
 */
export function initializeCellSelection(
  dayIndex: number,
  roomIndex: number,
  startDate: Date,
  endDate: Date,
  startX: number,
  startY: number,
): DragState {
  return {
    id: `select-${Date.now()}`,
    mode: "select-cells",
    startX,
    startY,

    startRoomIndex: roomIndex,
    startDayIndex: dayIndex,
    currentRoomIndex: roomIndex,
    currentDayIndex: dayIndex,

    currentX: 0,
    currentY: 0,
    width: 0,
    height: 0,
    initialStart: startDate,
    initialEnd: endDate,
    initialRoomIndex: roomIndex,
    shadowRowIndex: roomIndex + 2,
    shadowStartOffset: dayIndex + 2,
    shadowDuration: 1,
  };
}

/**
 * Updates the state during a cell selection drag operation.
 *
 * @param {DragState} dragState - The current drag state.
 * @param {number} dx - The delta X movement of the mouse.
 * @param {number} totalDays - The total number of days in the current view.
 * @param {Room[]} rooms - The list of rooms to check for overlaps.
 * @param {Date} startDate - The planner's overall start date.
 * @returns {DragState} The updated drag state object.
 */
export function updateCellSelectionDrag(
  dragState: DragState,
  dx: number,
  totalDays: number,
  rooms: Room[],
  startDate: Date,
): DragState {
  if (dragState.mode !== "select-cells") return dragState;

  const gridDeltaX = Math.round(dx / COL_WIDTH);
  let dayIndex = dragState.startDayIndex! + gridDeltaX;

  dayIndex = Math.max(0, Math.min(dayIndex, totalDays - 1));

  const selectableEnd = getSelectableEndDay(
    dragState.startDayIndex!,
    dragState.currentRoomIndex!,
    rooms,
    startDate,
  );

  dayIndex = Math.min(dayIndex, selectableEnd);

  const startDay = Math.min(dragState.startDayIndex!, dayIndex);
  const endDay = Math.max(dragState.startDayIndex!, dayIndex);

  return {
    ...dragState,
    currentDayIndex: dayIndex,
    shadowStartOffset: startDay + 2,
    shadowDuration: endDay - startDay + 1,
  };
}

/**
 * Updates the state during a booking drag (move or resize) operation.
 *
 * @param {DragState} dragState - The current drag state.
 * @param {number} dx - The horizontal delta movement of the mouse.
 * @param {number} dy - The vertical delta movement of the mouse.
 * @param {Date} startDate - The planner's start date.
 * @param {any[]} rooms - The current array of rooms.
 * @returns {DragState} The updated drag state object.
 */
export function updateBookingDrag(
  dragState: DragState,
  dx: number,
  dy: number,
  startDate: Date,
  rooms: any[],
): DragState {
  if (dragState.mode === "select-cells") return dragState;

  const gridDeltaX = Math.round(dx / COL_WIDTH);
  const gridDeltaY = Math.round(dy / ROW_HEIGHT);

  if (dragState.mode === "move") {
    return {
      ...dragState,
      shadowStartOffset:
        diffDays(dragState.initialStart, startDate) + 2 + gridDeltaX,
      shadowRowIndex: dragState.initialRoomIndex + 2 + gridDeltaY,
    };
  }
  const startDayIndex = diffDays(dragState.initialStart, startDate);
  const selectableEnd = getSelectableEndDay(
    startDayIndex,
    dragState.initialRoomIndex,
    rooms,
    startDate,
  );

  let newDuration = Math.max(
    1,
    diffDays(dragState.initialEnd, dragState.initialStart) + gridDeltaX,
  );

  const maxDuration = Math.max(1, selectableEnd - startDayIndex + 1);
  newDuration = Math.min(newDuration, maxDuration);

  return {
    ...dragState,
    shadowDuration: newDuration,
  };
}

/**
 * Finalizes cell selection and returns the precise boundary of the selected area.
 *
 * @param {DragState} dragState - The current drag state upon release.
 * @param {Date} startDate - The planner's start date (unused directly here, but kept for signature).
 * @param {number} totalDays - The total days in the view (unused directly).
 * @returns {CellSelection} Final selection configuration.
 * @throws {Error} If the drag state mode is not "select-cells".
 */
export function finalizeCellSelection(
  dragState: DragState,
  startDate: Date,
  totalDays: number,
): CellSelection {
  if (dragState.mode !== "select-cells") {
    throw new Error("Not in select-cells mode");
  }

  return {
    startRoom: dragState.startRoomIndex!,
    endRoom: dragState.currentRoomIndex!,
    startDay: Math.min(dragState.startDayIndex!, dragState.currentDayIndex!),
    endDay: Math.max(dragState.startDayIndex!, dragState.currentDayIndex!),
  };
}

/**
 * Finalizes a booking update, providing the new date bounds and room ID.
 *
 * @param {DragState} dragState - The active drag state upon drop.
 * @param {Date} startDate - The baseline start date of the grid.
 * @param {any[]} rooms - The array of room objects for identifying target rows.
 * @returns {{ finalStart: Date; finalEnd: Date; finalRoomId: string } | null} The new booking details or null if invalid.
 */
export function finalizeBookingUpdate(
  dragState: DragState,
  startDate: Date,
  rooms: any[],
): { finalStart: Date; finalEnd: Date; finalRoomId: string } | null {
  if (dragState.mode === "select-cells") return null;

  const daysFromStart = dragState.shadowStartOffset - 2;
  const finalStart = addDays(startDate, daysFromStart);
  const finalEnd = addDays(finalStart, dragState.shadowDuration);
  const finalRoomIdx = dragState.shadowRowIndex - 2;

  if (finalRoomIdx < 0 || finalRoomIdx >= rooms.length) {
    return null;
  }

  return {
    finalStart,
    finalEnd,
    finalRoomId: rooms[finalRoomIdx].room_id,
  };
}
