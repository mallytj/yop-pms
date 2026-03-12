import { fetchRatePlans, fetchRateMap } from "$lib/api/booking";
import type { PageServerLoad } from "./$types";

export const load = (async ({ params }) => {
  // TODO: get from params
  const checkInDate = "2026-03-05";
  const checkOutDate = "2026-03-10";

  try {
    const [ratePlans, rateMap] = await Promise.all([
      fetchRatePlans(checkInDate, checkOutDate),
      fetchRateMap(checkInDate, checkOutDate),
    ]);

    return {
      ratePlans,
      rateMap,
      checkInDate,
      checkOutDate,
    };
  } catch (error) {
    console.error(error);
  }
}) satisfies PageServerLoad;
