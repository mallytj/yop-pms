import { addDays, isISO8601Date, localDateKey, type ISO8601Date } from '$helpers/dates.js';
import { DEFAULT_LOOKBACK_DAYS, TAPE_CHART_WINDOW_DAYS } from '$lib/api/tape-chart.js';

export function isValidDateKey(value: string | null): value is ISO8601Date {
	return value !== null && isISO8601Date(value);
}

export function normalizeDateRange(
	requestedFrom: string | null,
	today: Date = new Date()
): { from: ISO8601Date; to: ISO8601Date; shouldRedirect: boolean } {
	const fallbackFrom = addDays(localDateKey(today), -DEFAULT_LOOKBACK_DAYS);
	const from = isValidDateKey(requestedFrom) ? requestedFrom : fallbackFrom;
	const to = addDays(from, TAPE_CHART_WINDOW_DAYS);

	return { from, to, shouldRedirect: requestedFrom !== from };
}
