import { describe, expect, it } from 'vitest';
import { isValidDateKey, normalizeDateRange } from './date.js';

describe('normalizeDateRange', () => {
	it('keeps a fixed 50-day window across a month boundary', () => {
		const range = normalizeDateRange('2026-01-31', new Date('2026-01-31T12:00:00Z'));

		expect(range).toEqual({ from: '2026-01-31', to: '2026-03-22', shouldRedirect: false });
	});

	it('does not redirect when the requested start date is already valid', () => {
		const range = normalizeDateRange('2026-05-01', new Date('2026-04-10T12:00:00Z'));

		expect(range).toEqual({ from: '2026-05-01', to: '2026-06-20', shouldRedirect: false });
	});

	it('falls back to a window starting DEFAULT_LOOKBACK_DAYS before today when the start date is invalid', () => {
		const range = normalizeDateRange('not-a-date', new Date('2026-04-10T12:00:00Z'));

		expect(range).toEqual({ from: '2026-03-21', to: '2026-05-10', shouldRedirect: true });
	});

	it('zero-pads a single-digit day and month in the fallback', () => {
		const range = normalizeDateRange(null, new Date('2026-01-05T12:00:00Z'));

		expect(range).toEqual({ from: '2025-12-16', to: '2026-02-04', shouldRedirect: true });
	});
});

describe('isValidDateKey', () => {
	it('accepts a real calendar date', () => {
		expect(isValidDateKey('2026-02-28')).toBe(true);
	});

	it('rejects an impossible calendar date', () => {
		expect(isValidDateKey('2026-02-29')).toBe(false);
	});

	it('rejects null', () => {
		expect(isValidDateKey(null)).toBe(false);
	});

	it('rejects an empty string', () => {
		expect(isValidDateKey('')).toBe(false);
	});

	it('rejects a month or day missing its zero-padding', () => {
		expect(isValidDateKey('2026-2-5')).toBe(false);
	});

	it('rejects a 2-digit year', () => {
		expect(isValidDateKey('26-02-05')).toBe(false);
	});

	it('rejects a date-time string with trailing content past the date', () => {
		expect(isValidDateKey('2026-02-05T00:00:00Z')).toBe(false);
	});

	it('rejects a short prefix even though it round-trips through Date as a valid year', () => {
		// `new Date('2026T00:00:00Z')` parses fine and its ISO output starts
		// with '2026', so only the regex length check protects against this.
		expect(isValidDateKey('2026')).toBe(false);
	});
});
