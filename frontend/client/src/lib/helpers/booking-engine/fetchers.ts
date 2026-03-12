import type {
  RatePlan,
  GetRatePlanResponse,
  ISO8601Date,
  RateMapResponse,
} from "$lib/types/booking_engine";

const API_BASE_URL = "http://localhost:8080/v1";
/**
 * Fetches available rate plans from the API for the given date range.
 *
 * @param {string} checkInDate - The check-in date in YYYY-MM-DD format.
 * @param {string} checkOutDate - The check-out date in YYYY-MM-DD format.
 * @returns {Promise<RatePlan[]>} A promise resolving to an array of rate plans.
 * @throws Will throw an error if the request fails.
 */
async function fetchRatePlans(
  checkInDate: string,
  checkOutDate: string,
): Promise<RatePlan[]> {
  const response = await fetch(`${API_BASE_URL}/rate-plans`);
  const data = await response.json();

  if (!response.ok) {
    throw new Error(data);
  }

  const ratePlans: RatePlan[] = data.map((ratePlan: GetRatePlanResponse) => ({
    id: ratePlan.id,
    name: ratePlan.name,
    description: ratePlan.description,
    code: ratePlan.code,
  }));

  return ratePlans;
}

/**
 * Fetches the rate map from the API containing prices and restrictions.
 *
 * @param {ISO8601Date} checkInDate - The state date in YYYY-MM-DD format.
 * @param {ISO8601Date} checkOutDate - The end date in YYYY-MM-DD format.
 * @returns {Promise<RateMapResponse>} A promise resolving to the rate map response.
 * @throws Will throw an error if the request fails.
 */
async function fetchRateMap(
  checkInDate: ISO8601Date,
  checkOutDate: ISO8601Date,
): Promise<RateMapResponse> {
  const response = await fetch(
    `${API_BASE_URL}/rate-map?startDate=${checkInDate}&endDate=${checkOutDate}`,
  );
  const data = await response.json();

  if (!response.ok) {
    throw new Error(data);
  }

  return data as RateMapResponse;
}

export { fetchRatePlans, fetchRateMap };
