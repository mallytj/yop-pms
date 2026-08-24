import { addDays } from '$helpers/dates.js';

export const TAPE_CHART_WINDOW_DAYS = 50;
/** Default window starts this many days before today, so a fresh visit already has room to scroll backward. */
export const DEFAULT_LOOKBACK_DAYS = 20;

export function isValidDateKey(value: string | null): value is string {
	// Either anchor alone still requires a 10-char digit-dash prefix, and any
	// value with extra content before/after it fails to round-trip through
	// the `date.toISOString().startsWith(value)` check below regardless — so
	// dropping just one anchor here is unobservable. Verified empirically.
	// Stryker disable next-line Regex
	if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
	const date = new Date(`${value}T00:00:00Z`);
	return !Number.isNaN(date.getTime()) && date.toISOString().startsWith(value);
}

export function normalizeDateRange(
	requestedFrom: string | null,
	today: Date = new Date()
): { from: string; to: string; shouldRedirect: boolean } {
	const todayKey = [
		String(today.getFullYear()),
		String(today.getMonth() + 1).padStart(2, '0'),
		String(today.getDate()).padStart(2, '0')
	].join('-');

	const fallbackFrom = addDays(todayKey, -DEFAULT_LOOKBACK_DAYS);
	const from = isValidDateKey(requestedFrom) ? requestedFrom : fallbackFrom;
	const to = addDays(from, TAPE_CHART_WINDOW_DAYS);

	return { from, to, shouldRedirect: requestedFrom !== from };
}
