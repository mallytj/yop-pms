import type { RateData, RatePlan } from "$lib/types";

  /**
   * Sums up the base and final prices of rate plans from a list of rate data objects.
   * @param rateDataList - The list of rate data objects.
   * @returns An array of aggregated rate plans.
   */
  function aggregateRatePlans(rateDataList: RateData[]): RatePlan[] {
    const aggregated = new Map<string, RatePlan>();

    rateDataList.forEach((rateData) => {
      rateData.ratePlans.forEach((plan) => {
        const existing = aggregated.get(plan.id);
        if (existing) {
          existing.basePrice += plan.basePrice;
          existing.finalPrice += plan.finalPrice;
        } else {
          aggregated.set(plan.id, { ...plan });
        }
      });
    });

    return Array.from(aggregated.values());
  }

  export { aggregateRatePlans };