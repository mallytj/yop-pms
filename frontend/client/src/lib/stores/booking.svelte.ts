import {
  applyAdjustment,
  generateDailyRates,
  recalculateTotals,
} from "$helpers/booking-engine";
import type {
  Adjustment,
  ISO8601Date,
  RateMap,
  RatePlan,
  ReservationDraft,
  ReservationItemDraft,
  UUID,
  Step,
} from "$types/booking_engine";
import { getContext, setContext } from "svelte";

const BOOKING_STORE_KEY = Symbol("BOOKING_STORE");

/**
 * High-level wrapper to initialize and provide the store.
 * @param {ReservationDraft} initialData The initial reservation draft
 * @param {RateMap} rateMap The rate map for the booking session
 * @param {RatePlan[]} ratePlans The available rate plans for the booking session
 * @returns {BookingStore} The booking store
 */
export function setBookingContext(
  initialData: ReservationDraft,
  rateMap: RateMap,
  ratePlans: RatePlan[],
): BookingStore {
  const store = new BookingStore(initialData, rateMap, ratePlans);
  setContext(BOOKING_STORE_KEY, store);
  return store;
}

/**
 * Hook-style helper to retrieve the store in child components.
 * @returns {BookingStore} The booking store
 */
export function useBookingStore(): BookingStore {
  const store = getContext<BookingStore>(BOOKING_STORE_KEY);
  if (!store) {
    throw new Error(
      "useBookingStore must be used within a BookingEngine component.",
    );
  }
  return store;
}

/**
 * BookingStore manages the reactive state of a reservation draft
 * It handles all the logic for updating the draft and its items
 */
export class BookingStore {
  // -- State

  /** The current step of the booking process */
  step = $state<Step>("rates");

  /** The root state of the reservation. Using non-null as initialised in the constructor */
  draft = $state<ReservationDraft>()!;

  /** Available rate plans for the booking session */
  plans = $state<RatePlan[]>([]);

  /** Reference data for pricing calculations. Read-only to ensure state integrity */
  readonly rateMap: RateMap;

  // -- Methods
  constructor(
    initialData: ReservationDraft,
    rateMap: RateMap,
    ratePlans: RatePlan[],
  ) {
    this.draft = initialData;
    this.rateMap = rateMap;
    this.plans = ratePlans;
  }

  /**
   * Returns the list of reservation items (rooms) currently in the draft.
   * @returns {ReservationItemDraft[]} Items in the draft
   */
  get items(): ReservationItemDraft[] {
    return this.draft?.items;
  }

  /**
   * Calculates the grand total of all items in the draft.
   * Automatically recalculates when any item's 'computed' property changes.
   * @returns {number} Total price in pence
   */
  get grandTotal(): number {
    return this.items?.reduce((acc, item) => {
      return acc + (item.computed?.finalTotal ?? 0);
    }, 0);
  }

  // -- Public Functions
  /**
   * Sets the step of the booking process
   * @param {Step} step The step to set
   * @returns {void}
   */
  public setStep(step: Step): void {
    if (!this.#validStep(step)) return;
    this.step = step;
  }

  /**
   * Selects a new rate plan for the provided item
   * @param {UUID} itemId The unique temporary ID of the item to update
   * @param {UUID} ratePlanId The ID of the rate plan to apply
   * @throws {Error} If the item or rate plan is not found
   * @returns {void}
   */
  public selectRatePlan(itemId: UUID, ratePlanId: UUID): void {
    const item = this.#getItem(itemId);

    item.selectedRatePlanId = ratePlanId;
    item.dailyRates = generateDailyRates(
      this.draft?.globalCheckInDate!,
      this.draft?.globalCheckOutDate!,
      ratePlanId,
      item.bookedRoomTypeId,
      this.rateMap,
    );

    item.computed = recalculateTotals(item);
  }

  /**
   * Updates the occupancy for a specific item
   * @param {UUID} itemId - The unique temporary ID of the item to update
   * @param {number} adults - Number of adults (min 1)
   * @param {number} children - Number of children (min 0)
   * @throws {Error} If the item is not found
   * @returns {void}
   */
  public updateOccupancy(itemId: UUID, adults: number, children: number): void {
    const item = this.#getItem(itemId);

    item.adults = Math.max(1, adults);
    item.children = Math.max(0, children);
  }

  /**
   * Applies a manual price adjustment to a single date.
   * Marks the day as 'adjustmentApproved' to lock it from global changes.
   * @param {UUID} itemId The item to update the adjustment on
   * @param {ISO8601Date} date The YYYY-MM-DD to update
   * @param {Adjustment} adjustment The adjustment to apply
   * @throws {Error} If the item is not found
   * @returns {void}
   */
  public setDailyAdjustment(
    itemId: UUID,
    date: ISO8601Date,
    adjustment: Adjustment,
  ): void {
    const item = this.#getItem(itemId);

    const day = item.dailyRates?.find((d) => d.date === date);

    // Don't need to throw here as it's not a critical error
    if (!day) return;

    day.adjustment = adjustment;
    day.adjustmentApproved = true;
    day.computedFinalPricePence = applyAdjustment(
      day.adjustment,
      day.basePricePence,
    );

    item.computed = recalculateTotals(item);
  }

  /**
   * Applies an adjustment across all days in an item.
   * Uses weighted distribution for fixed amounts to preserve price ratios.
   * @param {UUID} itemId The item to adjust globally
   * @param {Adjustment} adjustment The adjustment to distribute
   * @returns {void}
   */
  public setGlobalAdjustment(itemId: UUID, adjustment: Adjustment) {
    const item = this.#getItem(itemId);
    if (!item) return;

    const cleanValue = adjustment.value ?? 0;

    if (adjustment.type === "percentage") {
      this.#applyPercentageToAll(item, { ...adjustment, value: cleanValue });
    } else {
      this.#distributeFixedAmount(item, { ...adjustment, value: cleanValue });
    }

    item.computed = recalculateTotals(item);
  }

  /**
   * Manually flags a daily adjustment as approved/locked.
   * @param {UUID} itemId The item to approve
   * @param {ISO8601Date} date The date to approve
   * @returns {void}
   */
  public approveAdjustment(itemId: UUID, date: ISO8601Date): void {
    const item = this.#getItem(itemId);
    if (!item) return;

    const day = item.dailyRates?.find((d) => d.date === date);
    if (!day) return;

    day.adjustmentApproved = true;
  }

  /**
   * Copies the rate plan and occupancy from the first item to all other items in the draft
   * Useful for group bookings where all rooms share the same configuration
   */
  public applyFirstToAll(): void {
    const source = this.draft?.items[0];
    if (!source?.selectedRatePlanId) return;

    for (let i = 1; i < (this.draft?.items.length ?? 0); i++) {
      this.selectRatePlan(
        this.draft?.items[i].tempId ?? "",
        source.selectedRatePlanId,
      );
      this.updateOccupancy(
        this.draft?.items[i].tempId ?? "",
        source.adults ?? 2,
        source.children ?? 0,
      );
    }
  }

  // -- Private Functions
  /**
   * Validates that the provided step is a valid step
   * @param step The step to validate
   * @returns True if the step is valid, false otherwise
   */
  #validStep(step: Step): boolean {
    return ["rates", "guests", "review"].includes(step);
  }

  /**
   * Applies a percentage adjustment to all days that aren't manually locked
   * @param {ReservationItemDraft} item The item to adjust
   * @param {Adjustment} adj The percentage adjustment
   * @returns {void}
   */
  #applyPercentageToAll(item: ReservationItemDraft, adj: Adjustment): void {
    item.dailyRates?.forEach((day) => {
      // If a manual adjustment has been made, skip
      if (day.adjustmentApproved) return;

      day.adjustment = adj;
      day.adjustmentApproved = false;
      day.computedFinalPricePence = applyAdjustment(
        day.adjustment,
        day.basePricePence,
      );
    });
  }

  /**
   * Distributes a fixed amount across all non-locked days using weighted distribution
   * Handles penny-rounding errors on the final distributed day
   * @param {ReservationItemDraft} item The item to adjust
   * @param {Adjustment} adj The fixed amount adjustment
   * @returns {void}
   */
  #distributeFixedAmount(item: ReservationItemDraft, adj: Adjustment): void {
    const dailyRates = item.dailyRates ?? [];
    // If a manual adjustment has been made, do not apply
    const targetDays = dailyRates.filter((d) => !d.adjustmentApproved);

    const totalBase = targetDays.reduce((acc, d) => acc + d.basePricePence, 0);
    if (totalBase === 0) return;

    let distributedSum = 0;
    targetDays.forEach((day, index) => {
      const weight = day.basePricePence / totalBase;
      const portion = Math.round(adj.value! * weight);
      distributedSum += portion;

      day.adjustment = { ...adj, value: portion };
      day.adjustmentApproved = false;
      day.computedFinalPricePence = applyAdjustment(
        day.adjustment,
        day.basePricePence,
      );

      // Last-day penny correction
      if (index === dailyRates.length - 1) {
        const remainder = adj.value! - distributedSum;

        if (remainder !== 0) {
          day.adjustment.value = (day.adjustment.value ?? 0) + remainder;
          day.computedFinalPricePence = applyAdjustment(
            day.adjustment,
            day.basePricePence,
          );
        }
      }
    });
  }

  /**
   * Retrieves an item from the draft and validates its existence
   * @param {UUID} itemId The item to find
   * @returns {ReservationItemDraft} The found item
   * @throws {Error} If the item is not found
   */
  #getItem(itemId: UUID): ReservationItemDraft {
    const item = this.items?.find((item) => item.tempId === itemId);
    if (!item) {
      throw new Error(`[BookingStore] Item ${itemId} not found`);
    }

    return item;
  }
}
