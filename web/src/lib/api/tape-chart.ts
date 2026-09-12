import type { TapeChartResponse } from '$types/tape-chart.js';
import { api } from '$lib/api/client.js';
import type { ISO8601Date } from '$helpers/dates.js';

/** Days fetched per tape chart window — one fetch covers ~7 weeks of columns. */
export const TAPE_CHART_WINDOW_DAYS = 50;

/** Default window starts this many days before today, so a fresh visit
 * already has room to scroll backward. */
export const DEFAULT_LOOKBACK_DAYS = 20;

export async function fetchTapeChart(
	from: ISO8601Date,
	to: ISO8601Date,
	include: 'shallow' | 'full' = 'full',
	customFetch?: typeof fetch
): Promise<TapeChartResponse> {
	const { data, error: apiError } = await api.GET('/v1/tape-chart', {
		params: { query: { from, to, include } },
		fetch: customFetch
	});

	if (apiError || !data?.data) {
		throw new Error(apiError?.message ?? 'Failed to fetch tape chart');
	}

	return data;
}
