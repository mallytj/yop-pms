import type {
  BookedDailyRateDraft,
  UUID,
  RateMap,
  RateMapResponse,
  RateMapItem,
  ReservationItemDraft,
  Adjustment,
  Money,
  ISO8601Date,
} from "$types/booking_engine";

/**
 * Generates an array of daily rate drafts for a booked reservation.
 *
 * @param {string} checkInDate - The check-in date in YYYY-MM-DD format.
 * @param {string} checkOutDate - The check-out date in YYYY-MM-DD format.
 * @param {UUID} ratePlanId - The ID of the selected rate plan.
 * @param {UUID} roomTypeId - The ID of the selected room type.
 * @param {RateMap} priceGrid - The map containing rate information.
 * @returns {BookedDailyRateDraft[]} An array of daily rate drafts for the given dates.
 */
function generateDailyRates(
  checkInDate: string,
  checkOutDate: string,
  ratePlanId: UUID,
  roomTypeId: UUID,
  priceGrid: RateMap,
): BookedDailyRateDraft[] {
  const dates = getDatesInRange(checkInDate, checkOutDate);

  return dates
    .map((date) => {
      const key = `${roomTypeId}-${ratePlanId}-${date}`;
      const price = priceGrid.get(key)?.price;
      if (!price) return null;
      return {
        date,
        ratePlanId,
        basePricePence: price,
        adjustment: undefined,
        adjustmentApproved: false,
        computedFinalPricePence: price,
      };
    })
    .filter((rate) => rate !== null);
}

/**
 * Generates an array of date strings between a start and end date.
 *
 * @param {string} startDate - The start date in YYYY-MM-DD format.
 * @param {string} endDate - The end date (exclusive) in YYYY-MM-DD format.
 * @returns {string[]} An array of ISO date strings.
 */
function getDatesInRange(startDate: string, endDate: string) {
  const dates: string[] = [];
  const cur = new Date(startDate);
  const end = new Date(endDate);
  while (cur < end) {
    dates.push(cur.toISOString().split("T")[0]);
    cur.setDate(cur.getDate() + 1);
  }
  return dates;
}

/**
 * Builds a rate map dictionary from an API response.
 *
 * @param {RateMapResponse} rateMap - The raw rate map response from the backend.
 * @returns {RateMap} A structured dictionary mapping keys to rate items.
 */
function buildRateMap(rateMap: RateMapResponse): RateMap {
  const map = new Map<string, RateMapItem>();

  rateMap.rates.forEach((rate) => {
    const key = makeRateMapKey(
      rate.room_type_id,
      rate.rate_plan_id,
      rate.calendar_date.split("T")[0],
    );
    map.set(key, {
      price: rate.price,
      minLos: rate.min_los,
      maxLos: rate.max_los,
    });
  });

  return map;
}

/**
 * Recalculates the base, adjustment, and final totals for a reservation item draft.
 *
 * @param {ReservationItemDraft} item - The reservation item draft containing daily rates.
 * @returns {ReservationItemDraft["computed"]} The computed total amounts.
 */
function recalculateTotals(
  item: ReservationItemDraft,
): ReservationItemDraft["computed"] {
  const baseTotal = item.dailyRates.reduce(
    (sum, d) => sum + d.basePricePence,
    0,
  );
  const finalTotal = item.dailyRates.reduce(
    (sum, d) => sum + d.computedFinalPricePence,
    0,
  );
  return {
    baseTotal,
    adjustmentTotal: finalTotal - baseTotal,
    finalTotal,
  };
}

/**
 * Applies an adjustment (fixed or percentage) to a base price.
 *
 * @param {Adjustment} adjustment - The adjustment object to apply.
 * @param {number} basePricePence - The base price in pence.
 * @returns {number} The newly adjusted price in pence.
 */
function applyAdjustment(adjustment: Adjustment, basePricePence: number) {
  if (!adjustment.value) return basePricePence;

  return adjustment.type === "fixed_amount"
    ? basePricePence + (adjustment.value ?? 0) * 100
    : !!adjustment.value
      ? basePricePence + (basePricePence * adjustment.value) / 100
      : basePricePence;
}

/**
 * Generates a unique key for looking up rates in a rate map.
 *
 * @param {UUID} roomTypeId - The ID of the room type.
 * @param {UUID} ratePlanId - The ID of the rate plan.
 * @param {ISO8601Date} date - The date string.
 * @returns {string} The constructed key string.
 */
function makeRateMapKey(roomTypeId: UUID, ratePlanId: UUID, date: ISO8601Date) {
  return `${roomTypeId}_${ratePlanId}_${date}`;
}

/**
 * getTotalRate - Get the total rate for a given set of dates
 * @param roomTypeId - The room type ID
 * @param ratePlanId - The rate plan ID
 * @param rateMap - The rate map
 * @param dates - The dates to get the total rate for
 * @returns The total rate for the given dates
 */
function getTotalRate(
  roomTypeId: UUID,
  ratePlanId: UUID,
  rateMap: RateMap,
  dates: ISO8601Date[],
): Money {
  let total = 0;

  for (const date of dates) {
    const key = makeRateMapKey(roomTypeId, ratePlanId, date);

    const price = rateMap.get(key)?.price;
    if (!price) return -1;

    total += price;
  }
  return total;
}

/**
 * Calculates the total number of days between two dates.
 *
 * @param {string} startDate - The start date in YYYY-MM-DD format.
 * @param {string} endDate - The end date in YYYY-MM-DD format.
 * @returns {number} The difference in days.
 */
function getDaysBetween(startDate: string, endDate: string) {
  const dates = getDatesInRange(startDate, endDate);
  return dates.length;
}

/**
 * Converts a string to a two decimal place currency string
 *
 * @example strNumToTwoDecimalPoints("123.456") // "123.46"
 * @example strNumToTwoDecimalPoints("123") // "123.00"
 * @example strNumToTwoDecimalPoints("123.4") // "123.40"
 * @example strNumToTwoDecimalPoints("") // "0.00"
 *
 * @param {string} str - The string number to convert
 * @returns {string} The two decimal place currency string
 */
function strNumToTwoDecimalPoints(str: string) {
  if (!str) return "0.00";
  return String(Math.round(Number(str) * 100) / 100);
}

/**
 * Formats a given price in pence to a string represented in pounds (£)
 *
 * @example formatPrice(1000) => £10.00
 *
 * @param {number} pence - The price amount in pence.
 * @returns {string} The formatted price string (e.g., £10.00).
 */
function formatPrice(pence: number): string {
  return `${pence < 0 ? "-" : ""}£${Math.abs(pence / 100).toFixed(2)}`;
}

export {
  formatPrice,
  getDaysBetween,
  generateDailyRates,
  getDatesInRange,
  buildRateMap,
  applyAdjustment,
  recalculateTotals,
  getTotalRate,
  strNumToTwoDecimalPoints as strNumToCurrencyStr,
};
