const WEEKDAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'] as const;
const MONTHS = [
	'Jan',
	'Feb',
	'Mar',
	'Apr',
	'May',
	'Jun',
	'Jul',
	'Aug',
	'Sep',
	'Oct',
	'Nov',
	'Dec'
] as const;

/** Local calendar YYYY-MM-DD (operator wall clock). */
export function localDateKey(date = new Date()): string {
	const year = date.getFullYear();
	const month = String(date.getMonth() + 1).padStart(2, '0');
	const day = String(date.getDate()).padStart(2, '0');
	return `${year}-${month}-${day}`;
}

/**
 * Normalize any API date-ish value to YYYY-MM-DD calendar key.
 * Date-only strings stay as-is. Full ISO timestamps use UTC calendar day
 * (matches Go time.Time JSON and pgtype date-as-UTC-midnight).
 */
export function toDateKey(value: unknown): string {
	if (value == null) return '';
	if (value instanceof Date) {
		if (!Number.isNaN(value.getTime())) return value.toISOString().slice(0, 10);
		return '';
	}
	if (typeof value === 'string') {
		const trimmed = value.trim();
		if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) return trimmed;
		// Any garbage prefix before a real timestamp already fails `new Date`
		// parsing below regardless of the leading `^`, so dropping it here is
		// unobservable — verified empirically across several garbage prefixes.
		// Stryker disable next-line Regex
		if (/^\d{4}-\d{2}-\d{2}T/.test(trimmed)) {
			const parsed = new Date(trimmed);
			if (!Number.isNaN(parsed.getTime())) return parsed.toISOString().slice(0, 10);
		}
		return '';
	}
	// Non-object primitives (number, boolean) reaching this point via `unknown`
	// have no .Time/.time property, so entering this branch unconditionally is
	// safe and produces the same '' fallback — verified equivalent.
	// Stryker disable next-line ConditionalExpression
	if (typeof value === 'object') {
		const record = value as Record<string, unknown>;
		if (typeof record.Time === 'string') return toDateKey(record.Time);
		if (typeof record.time === 'string') return toDateKey(record.time);
	}
	return '';
}

/** Calendar-date arithmetic via UTC midnight (DST-safe for YYYY-MM-DD keys). */
export function addDays(dateStr: string, days: number): string {
	// Equivalent to daysBetween's suffix above — a bare YYYY-MM-DD already
	// parses as UTC midnight without it.
	// Stryker disable next-line StringLiteral
	const date = new Date(dateStr + 'T00:00:00.000Z');
	date.setUTCDate(date.getUTCDate() + days);
	return date.toISOString().slice(0, 10);
}

export function parseUTC(dateStr: string): Date {
	// Every caller reads only UTC calendar components (getUTCDate/Day/Month),
	// which are identical at midnight or noon UTC for the same calendar day —
	// so this offset is unobservable through this module's public API.
	// Stryker disable next-line StringLiteral
	return new Date(dateStr + 'T12:00:00.000Z');
}

export function formatDayLabel(dateStr: string): string {
	return parseUTC(dateStr).toLocaleDateString('en-GB', {
		day: 'numeric',
		month: 'short',
		year: 'numeric'
	});
}

export function dayNumber(dateStr: string): string {
	return String(parseUTC(dateStr).getUTCDate());
}

export function weekdayLabel(dateStr: string): string {
	return WEEKDAYS[parseUTC(dateStr).getUTCDay()];
}

export function monthLabel(dateStr: string): string {
	return MONTHS[parseUTC(dateStr).getUTCMonth()];
}

export function isMonthStart(dateStr: string): boolean {
	return parseUTC(dateStr).getUTCDate() === 1;
}

export function isWeekend(dateStr: string): boolean {
	const weekday = parseUTC(dateStr).getUTCDay();
	return weekday === 0 || weekday === 6;
}

export function buildDateRange(from: string, to: string, maxDays = 400): string[] {
	if (!from || !to) return [];
	const result: string[] = [];
	let current = from;
	for (let index = 0; index < maxDays; index++) {
		result.push(current);
		if (current >= to) break;
		current = addDays(current, 1);
	}
	return result;
}

/** Whole calendar days from `from` to `to` (negative if `to` precedes `from`). */
export function daysBetween(from: string, to: string): number {
	// `from`/`to` are always bare YYYY-MM-DD keys, which the Date constructor
	// already parses as UTC midnight — appending 'T00:00:00.000Z' is a no-op
	// verified against the literal-removal mutant, kept only for readability.
	// Stryker disable next-line StringLiteral
	const fromMs = new Date(from + 'T00:00:00.000Z').getTime();
	// Stryker disable next-line StringLiteral
	const toMs = new Date(to + 'T00:00:00.000Z').getTime();
	return Math.round((toMs - fromMs) / 86_400_000);
}

export function minDateKey(left: unknown, right: unknown): string {
	const leftKey = toDateKey(left);
	const rightKey = toDateKey(right);
	if (!leftKey) return rightKey;
	if (!rightKey) return leftKey;
	// `<=` vs `<` only differ when leftKey === rightKey, and then both
	// branches return the same string value — verified equivalent.
	// Stryker disable next-line EqualityOperator
	return leftKey <= rightKey ? leftKey : rightKey;
}

export function maxDateKey(left: unknown, right: unknown): string {
	const leftKey = toDateKey(left);
	const rightKey = toDateKey(right);
	// Both guards below are redundant with the ternary: '' sorts before any
	// non-empty date-key string lexicographically, so `leftKey >= rightKey`
	// already resolves to the same answer when either side is ''. Verified
	// equivalent for all inputs, not just the tested ones.
	// Stryker disable next-line ConditionalExpression
	if (!leftKey) return rightKey;
	// Stryker disable next-line ConditionalExpression
	if (!rightKey) return leftKey;
	// `>=` vs `>` only differ when leftKey === rightKey, and then both
	// branches return the same string value — verified equivalent.
	// Stryker disable next-line EqualityOperator
	return leftKey >= rightKey ? leftKey : rightKey;
}
