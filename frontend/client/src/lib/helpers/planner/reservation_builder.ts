import type {
  ReservationBase,
  RateData,
  GuestData,
} from "../../stores/_reservation_modal_store.ts";

export interface FinalReservation {
  roomId: string;
  guestName: string;
  guestEmail: string;
  guestPhone: string;
  checkInDate: string;
  checkOutDate: string;
  ratePerNight: number;
  totalPrice: number;
  status: string;
}

/**
 * Builds a list of final reservation objects combining bases, rate data, and guest data.
 *
 * @param {ReservationBase[]} bases - The reservation base configurations.
 * @param {RateData[]} rateData - The cost and rate info per reservation.
 * @param {GuestData[]} guestData - The guest details per reservation.
 * @param {string} [status="pending"] - The initial status assigned to the reservations.
 * @returns {FinalReservation[]} An array of finalized reservation objects.
 */
export function buildFinalReservations(
  bases: ReservationBase[],
  rateData: RateData[],
  guestData: GuestData[],
  status: string = "pending",
): FinalReservation[] {
  return bases.map((base, idx) => {
    const rate = rateData[idx];
    const guest = guestData[idx];

    return {
      roomId: base.roomId,
      guestName: guest.guestName,
      guestEmail: guest.guestEmail,
      guestPhone: guest.guestPhone,
      checkInDate: base.checkInDate,
      checkOutDate: base.checkOutDate,
      ratePerNight: rate.ratePerNight,
      totalPrice: rate.totalPrice,
      status,
    };
  });
}
