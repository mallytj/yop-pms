import type { TapeChartResponse } from '$types/tape-chart.js';
import { api } from '$lib/api/client.js';

export async function fetchTapeChart(
	from: string,
	to: string,
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
