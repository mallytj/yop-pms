import { addDays, diffDays, parseLocalYMD } from "./date_conversions.ts";
import type { CellSelection, Room } from "../../types/planner_data.ts";

/**
 * Checks if a selected area is valid and does not overlap with existing bookings.
 *
 * @param {CellSelection} selection - The area selected on the grid.
 * @param {Room[]} rooms - The list of rooms to check against.
 * @param {Date} startDate - The start date of the grid.
 * @returns {boolean} True if the selection is valid, false if there is an overlap.
 */
export function canSelectArea(
  selection: CellSelection,
  rooms: Room[],
  startDate: Date,
): boolean {
  const selectedRooms = rooms.slice(selection.startRoom, selection.endRoom + 1);
  const checkInDate = addDays(startDate, selection.startDay);
  const checkOutDate = addDays(startDate, selection.endDay + 1);

  // Check if any room in the selection has existing bookings that overlap
  for (const room of selectedRooms) {
    const existingBookings = room.reservations || [];

    for (const booking of existingBookings) {
      const bookingStart = parseLocalYMD(booking.check_in_date);
      const bookingEnd = parseLocalYMD(booking.check_out_date);

      // If there's any overlap, return false
      if (checkInDate < bookingEnd && checkOutDate > bookingStart) {
        return false;
      }
    }
  }

  return true;
}

/**
 * Determines the maximum selectable end day for a given room, based on existing bookings.
 *
 * @param {number} startDay - The starting day index of the selection.
 * @param {number} roomIdx - The index of the room being selected.
 * @param {Room[]} rooms - The list of available rooms.
 * @param {Date} startDate - The start date of the grid.
 * @returns {number} The maximum selectable day index before conflicting with another booking.
 */
export function getSelectableEndDay(
  startDay: number,
  roomIdx: number,
  rooms: Room[],
  startDate: Date,
): number {
  const room = rooms[roomIdx];
  if (!room) return startDay; // No room means no conflict

  let earliestConflict = 999; // Default to unlimited

  const bookings = room.reservations || [];

  // Find earliest booking in this room that's after startDay
  const conflictingBooking = bookings.find((booking) => {
    const bookingStartDay = diffDays(
      parseLocalYMD(booking.check_in_date),
      startDate,
    );
    return bookingStartDay > startDay;
  });

  if (conflictingBooking) {
    const bookingStartDay = diffDays(
      parseLocalYMD(conflictingBooking.check_in_date),
      startDate,
    );
    earliestConflict = Math.min(earliestConflict, bookingStartDay - 1);
  }

  return earliestConflict;
}
