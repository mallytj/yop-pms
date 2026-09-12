import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte/svelte5';
import { daysBetween } from '$helpers/dates.js';
import TapeChartDatePicker from './TapeChartDatePicker.svelte';

const FIXED_TODAY = new Date('2026-08-24T12:00:00Z');

async function givenTodayClicked(props: {
	from: string;
	onRangeChange: (from: string, to: string) => void;
	onToday?: () => void;
}) {
	render(TapeChartDatePicker, props);
	const todayButton = document.querySelector('button.today') as HTMLButtonElement;

	await fireEvent.click(todayButton);

	return todayButton;
}

describe('TapeChartDatePicker', () => {
	beforeEach(() => {
		vi.setSystemTime(FIXED_TODAY);
	});

	afterEach(() => {
		cleanup();
		vi.useRealTimers();
	});

	it('renders the start date field', () => {
		render(TapeChartDatePicker, { from: '2026-08-20', onRangeChange: vi.fn() });

		expect(document.querySelector('[aria-label="Tape Chart start date"]')).toBeTruthy();
	});

	it('requests a 50-day range when Today is clicked', async () => {
		const onRangeChange = vi.fn();

		await givenTodayClicked({ from: '2026-08-01', onRangeChange });

		expect(onRangeChange).toHaveBeenCalledTimes(1);
		const [from, to] = onRangeChange.mock.calls[0];
		expect(daysBetween(from, to)).toBe(50);
	});

	it('anchors the requested range 20 days before today, not at today itself, so the grid still has a past-day buffer to scroll into', async () => {
		const onRangeChange = vi.fn();

		await givenTodayClicked({ from: '2026-08-01', onRangeChange });

		const [from] = onRangeChange.mock.calls[0];
		const todayKey = FIXED_TODAY.toISOString().slice(0, 10);
		expect(daysBetween(from, todayKey)).toBe(20);
	});

	it('calls onToday, in addition to onRangeChange, when Today is clicked', async () => {
		const onToday = vi.fn();

		await givenTodayClicked({ from: '2026-08-01', onRangeChange: vi.fn(), onToday });

		expect(onToday).toHaveBeenCalledTimes(1);
	});

	it('does not require onToday to be passed', async () => {
		render(TapeChartDatePicker, { from: '2026-08-01', onRangeChange: vi.fn() });
		const todayButton = document.querySelector('button.today') as HTMLButtonElement;

		await expect(fireEvent.click(todayButton)).resolves.not.toThrow();
	});
});
