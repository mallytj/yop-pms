import type { Room, RoomType } from "$types";

interface RateData {
  id: string;
  calendarDate: Date;
  ratePlans: RatePlan[];
}

interface GuestData {
  id: string;
  guestName: string;
  guestEmail: string;
  guestPhone: string;
}

interface ReservationDraft {
  assignedRoom: Room;
  assignedRoomType: RoomType;
  stayDates: [Date, Date];
  totalPrice: number;
  guests: GuestData[];
  rateData: RateData[];
}

interface RatePlan {
  id: string;
  name: string;
  description: string;
  code: string;
  basePrice: number;
  finalPrice: number;
}

export type { RateData, RatePlan };
