import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types.js';
import { fetchTapeChart, TAPE_CHART_WINDOW_DAYS } from '$lib/api/tape-chart.js';
import { addDays } from '$helpers/dates.js';
import { isValidDateKey, normalizeDateRange } from './_utils/date.js';

export const load: PageServerLoad = ({ fetch, url }) => {
	const defaults = normalizeDateRange(null);
	const requestedFrom = url.searchParams.get('from');
	const requestedTo = url.searchParams.get('to');
	const from = isValidDateKey(requestedFrom) ? requestedFrom : defaults.from;
	// `to` is derived, never taken from the URL — a matching requestedTo just
	// means no redirect is needed to normalize the URL.
	const to = addDays(from, TAPE_CHART_WINDOW_DAYS);

	if (requestedFrom !== from || requestedTo !== to) {
		const next = new URL(url);
		next.searchParams.set('from', from);
		next.searchParams.set('to', to);
		throw redirect(302, `${next.pathname}?${next.searchParams.toString()}`);
	}

	return {
		from,
		to,
		// Streamed (not awaited): the route navigates with `from`/`to` right
		// away, so the grid can render its date header immediately, while
		// room/reservation data resolves in the background.
		tapeChart: fetchTapeChart(from, to, 'full', fetch)
			.then((response) => {
				if (!response.data) throw new Error('Tape chart response missing data');
				return response.data;
			})
			.catch((err) => {
				console.error('[tape-chart] load failed:', err);
				throw new Error(
					'Failed to load tape chart — check API is running and VITE_DEV_PROPERTY_ID is set'
				);
			})
	};
};
