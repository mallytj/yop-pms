import type { Room, RoomType } from "./planner_data";

export type UUID = string;
export type Money = number;
export type ISO8601Date = string; // "2026-10-12"
export type Step = "rates" | "guests" | "review"

export type AdjustmentType = "fixed_amount" | "percentage";

export interface RoomData {
  id: UUID;
  room_id: UUID;
  room_name: string;
  room_type_id: UUID;
  room_type_code: string;
  room_type_name: string;
}

export interface Adjustment {
  type: AdjustmentType;
  value: number | null;
  reason: string;
}

export interface DailyPriceQuote {
  date: ISO8601Date;
  ratePlanId: UUID;
  basePricePence: Money;
  minLos: number;
}

export interface ReservationDraft {
  primaryGuestId?: UUID;
  tempGuestProfile?: GuestProfileDraft;
  globalCheckInDate: ISO8601Date;
  globalCheckOutDate: ISO8601Date;
  items: ReservationItemDraft[];
}

export interface ItemTotals {
  baseTotal: Money;
  adjustmentTotal: Money;
  finalTotal: Money;
}

export interface ReservationItemDraft {
  tempId: UUID; // Used for #each loops, not for backedn
  bookedRoomTypeId: UUID;
  assignedRoomId?: UUID;
  assignedRoom?: RoomData;
  primaryGuestId?: UUID; // May be inherited
  tempGuestProfile?: GuestProfileDraft; // May be inherited from master

  selectedRatePlanId?: UUID | null;

  adults?: number;
  children?: number;

  dailyRates: BookedDailyRateDraft[]; // Filled by rates page

  otherGuests?: GuestProfileDraft[]; // MtM relationship

  computed?: ItemTotals;
}

export interface BookedDailyRateDraft {
  date: ISO8601Date;

  // Usually matches parent, but allows for split-rate
  ratePlanId: string;
  basePricePence: Money;
  adjustment?: Adjustment;
  adjustmentApproved: boolean; // Default fause
  computedFinalPricePence: Money;
}

export interface GuestProfileDraft {
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
}

export interface Group {}

export interface RatePlan {
  id: UUID;
  name: string;
  description: string;
  code: string;
}

export interface RateData {
  id: string;
  calendarDate: Date;
  ratePlans: RatePlan[];
}

export interface Rate {
  calendar_date: string;
  room_type_id: UUID;
  rate_plan_id: UUID;
  price: Money;
  min_los: number;
  max_los: number;
  source: string;
}

// <roomTypeId>-<ratePlanId>-<date>
export type RateMapKey = string;
export interface RateMapItem {
  price: Money;
  minLos: number;
  maxLos: number;
}

export interface RateMapResponse {
  check_in_date: string;
  check_out_date: string;
  rates: Rate[];
}

export type RateMap = Map<RateMapKey, RateMapItem>;

export interface BookingReferenceData {
  ratePlans: RatePlan[];
  rateMap: RateMap;
}

export interface GetRatePlanResponse {
  id: UUID;
  parent_rate_plan_id: UUID | null;
  name: string;
  description: string;
  code: string;
}
