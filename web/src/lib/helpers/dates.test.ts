import { describe, it, expect } from 'vitest';
import {
	addDays,
	buildDateRange,
	daysBetween,
	isMonthStart,
	isWeekend,
	localDateKey,
	maxDateKey,
	minDateKey,
	monthLabel,
	toDateKey
} from './dates.js';

describe('dates helpers', () => {
	it('addDays moves forward and backward across month boundaries', () => {
		expect(addDays('2026-08-31', 1)).toBe('2026-09-01');
		expect(addDays('2026-09-01', -1)).toBe('2026-08-31');
	});

	it('daysBetween counts whole calendar days in either direction', () => {
		expect(daysBetween('2026-08-20', '2026-08-27')).toBe(7);
		expect(daysBetween('2026-08-27', '2026-08-20')).toBe(-7);
		expect(daysBetween('2026-08-20', '2026-08-20')).toBe(0);
	});

	it('buildDateRange is inclusive of both endpoints', () => {
		expect(buildDateRange('2026-08-20', '2026-08-22')).toEqual([
			'2026-08-20',
			'2026-08-21',
			'2026-08-22'
		]);
	});

	it('buildDateRange returns an empty array when either endpoint is missing', () => {
		expect(buildDateRange('', '2026-08-22')).toEqual([]);
		expect(buildDateRange('2026-08-20', '')).toEqual([]);
	});

	it('buildDateRange stops at maxDays even if `to` is never reached', () => {
		expect(buildDateRange('2026-08-20', '2026-12-31', 3)).toEqual([
			'2026-08-20',
			'2026-08-21',
			'2026-08-22'
		]);
	});

	it('toDateKey normalizes ISO timestamps and date-only strings', () => {
		expect(toDateKey('2026-08-20T00:00:00Z')).toBe('2026-08-20');
		expect(toDateKey('2026-08-20')).toBe('2026-08-20');
		expect(toDateKey(null)).toBe('');
	});

	it('toDateKey normalizes Date instances and rejects invalid ones', () => {
		expect(toDateKey(new Date('2026-08-20T00:00:00Z'))).toBe('2026-08-20');
		expect(toDateKey(new Date('not-a-date'))).toBe('');
	});

	it('toDateKey unwraps Go pgtype-style { Time } / { time } records', () => {
		expect(toDateKey({ Time: '2026-08-20T00:00:00Z' })).toBe('2026-08-20');
		expect(toDateKey({ time: '2026-08-20T00:00:00Z' })).toBe('2026-08-20');
		expect(toDateKey({})).toBe('');
	});

	it('toDateKey rejects strings that are neither date-only nor ISO timestamps', () => {
		expect(toDateKey('not-a-date')).toBe('');
		// Non-ISO strings that Date can still loosely parse must not slip
		// through just because they fail the date-only/ISO-timestamp checks.
		expect(toDateKey('August 20, 2026')).toBe('');
	});

	it('toDateKey trims surrounding whitespace before matching', () => {
		expect(toDateKey(' 2026-08-20 ')).toBe('2026-08-20');
	});

	it('toDateKey rejects a date-only match embedded in a longer string', () => {
		expect(toDateKey('xx2026-08-20')).toBe('');
	});

	it('toDateKey rejects an ISO timestamp with an impossible calendar date', () => {
		expect(toDateKey('2026-13-40T00:00:00Z')).toBe('');
	});

	it('toDateKey returns an empty string for unsupported value types', () => {
		expect(toDateKey(42)).toBe('');
	});

	it('toDateKey does not recurse into a non-string nested time field', () => {
		expect(toDateKey({ time: { time: '2026-08-20' } })).toBe('');
	});

	it('isMonthStart and isWeekend flag calendar boundaries', () => {
		expect(isMonthStart('2026-09-01')).toBe(true);
		expect(isMonthStart('2026-09-02')).toBe(false);
		expect(isWeekend('2026-08-22')).toBe(true); // Saturday
		expect(isWeekend('2026-08-23')).toBe(true); // Sunday
		expect(isWeekend('2026-08-20')).toBe(false); // Thursday
	});

	it('localDateKey formats an explicit date as YYYY-MM-DD', () => {
		expect(localDateKey(new Date(2026, 0, 5))).toBe('2026-01-05');
		expect(localDateKey(new Date(2026, 10, 30))).toBe('2026-11-30');
	});

	it('monthLabel abbreviates the calendar month', () => {
		expect(monthLabel('2026-01-15')).toBe('Jan');
		expect(monthLabel('2026-12-01')).toBe('Dec');
	});

	it('minDateKey and maxDateKey pick the earlier/later of two date-ish values', () => {
		expect(minDateKey('2026-08-20', '2026-08-21')).toBe('2026-08-20');
		expect(minDateKey('2026-08-21', '2026-08-20')).toBe('2026-08-20');
		expect(minDateKey('2026-08-20', '2026-08-20')).toBe('2026-08-20');
		expect(minDateKey(null, '2026-08-21')).toBe('2026-08-21');
		expect(minDateKey('2026-08-20', null)).toBe('2026-08-20');

		expect(maxDateKey('2026-08-20', '2026-08-21')).toBe('2026-08-21');
		expect(maxDateKey('2026-08-21', '2026-08-20')).toBe('2026-08-21');
		expect(maxDateKey('2026-08-20', '2026-08-20')).toBe('2026-08-20');
		expect(maxDateKey(null, '2026-08-21')).toBe('2026-08-21');
		expect(maxDateKey('2026-08-20', null)).toBe('2026-08-20');
	});
});
