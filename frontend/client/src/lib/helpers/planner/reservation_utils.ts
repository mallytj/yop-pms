// lib/helpers/reservationHelpers.ts
import { addDays, dateToString } from "./planner_utils.ts";
import type { CellSelection, Room } from "../../types/planner_data.ts";
import type {
  ReservationBase,
  RateData,
  GuestData,
} from "../../stores/_reservation_modal_store.ts";

/**
 * Generates an array of reservation bases from a set of selections.
 *
 * @param {CellSelection[]} selections - The list of cell selections from the planner grid.
 * @param {Room[]} rooms - The list of all available rooms.
 * @param {Date} startDate - The starting date of the planner.
 * @returns {ReservationBase[]} An array of base reservation objects.
 */
export function generateReservationBases(
  selections: CellSelection[],
  rooms: Room[],
  startDate: Date,
): ReservationBase[] {
  return selections.flatMap((selection, selIdx) => {
    const selectedRooms = rooms.slice(
      selection.startRoom,
      selection.endRoom + 1,
    );
    const checkIn = addDays(startDate, selection.startDay);
    const checkOut = addDays(startDate, selection.endDay + 1);

    return selectedRooms.map((room, roomIdx) => ({
      id: `res-${selIdx}-${roomIdx}`,
      roomId: room.room_id,
      roomName: room.room_name,
      checkInDate: dateToString(checkIn),
      checkOutDate: dateToString(checkOut),
      nights: selection.endDay - selection.startDay + 1,
    }));
  });
}

/**
 * Initializes rate data objects for a given array of reservation bases.
 *
 * @param {ReservationBase[]} bases - The reservation bases to initialize rate data for.
 * @returns {RateData[]} The initialized array of rate data objects.
 */
export function initializeRateData(bases: ReservationBase[]): RateData[] {
  return bases.map((res) => ({
    id: res.id,
    ratePerNight: 0,
    totalPrice: 0,
  }));
}

/**
 * Initializes guest data objects for a given array of reservation bases.
 *
 * @param {ReservationBase[]} bases - The reservation bases to initialize guest data for.
 * @returns {GuestData[]} The initialized array of guest data objects.
 */
export function initializeGuestData(bases: ReservationBase[]): GuestData[] {
  return bases.map((res, idx) => ({
    id: res.id,
    guestName: idx === 0 ? "" : "",
    guestEmail: idx === 0 ? "" : "",
    guestPhone: idx === 0 ? "" : "",
  }));
}

/**
 * Updates the rate and total price for a specific rate data entry.
 *
 * @param {RateData[]} rateData - The array of current rate data.
 * @param {number} idx - The index of the rate data entry to update.
 * @param {number} rate - The new rate per night.
 * @param {number} nights - The number of nights for the reservation.
 * @returns {RateData[]} A new array with the updated rate data.
 */
export function updateRateData(
  rateData: RateData[],
  idx: number,
  rate: number,
  nights: number,
): RateData[] {
  const updated = [...rateData];
  updated[idx] = {
    ...updated[idx],
    ratePerNight: rate,
    totalPrice: rate * nights,
  };
  return updated;
}

/**
 * Applies the guest details from the primary guest to all other guests.
 *
 * @param {GuestData[]} guestData - The array of guest data elements.
 * @returns {GuestData[]} A new array of guest data with matching primary guest info.
 */
export function applyGuestToAll(guestData: GuestData[]): GuestData[] {
  const primaryGuest = guestData[0];
  return guestData.map((_, idx) => {
    if (idx === 0) return primaryGuest;
    return {
      ...guestData[idx],
      guestName: primaryGuest.guestName,
      guestEmail: primaryGuest.guestEmail,
      guestPhone: primaryGuest.guestPhone,
    };
  });
}

/**
 * Calculates the total price from an array of rate data objects.
 *
 * @param {RateData[]} rateData - The array of rate data to sum.
 * @returns {number} The aggregated total price.
 */
export function calculateTotalPrice(rateData: RateData[]): number {
  return rateData.reduce((sum, r) => sum + r.totalPrice, 0);
}

/**
 * Validates whether a given string is a correctly formatted email address.
 *
 * @param {string} email - The email string to validate.
 * @returns {boolean} True if the email format is valid, false otherwise.
 */
export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

/**
 * Validates that all items in an array of rate data have a rate strictly greater than 0.
 *
 * @param {RateData[]} rateData - The array of rate data to validate.
 * @returns {boolean} True if all rates are valid, false otherwise.
 */
export function validateRates(rateData: RateData[]): boolean {
  return rateData.every((r) => r.ratePerNight > 0);
}

/**
 * Validates a single guest data object, ensuring a name and valid email are provided.
 *
 * @param {GuestData} guest - The guest data to validate.
 * @returns {boolean} True if the guest name and email are valid.
 */
export function validateGuest(guest: GuestData): boolean {
  return !!guest.guestName && isValidEmail(guest.guestEmail);
}
