import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import TapeChartDateHeader from './TapeChartDateHeader.svelte';

describe('TapeChartDateHeader', () => {
	afterEach(() => {
		cleanup();
	});

	it('renders the day number for the given date', () => {
		render(TapeChartDateHeader, { date: '2026-08-21', today: '2026-08-20' });

		const dayNumber = document.querySelector('.date-day')?.textContent;

		expect(dayNumber).toBe('21');
	});

	it('renders the abbreviated weekday for the given date', () => {
		render(TapeChartDateHeader, { date: '2026-08-21', today: '2026-08-20' });

		const weekday = document.querySelector('.date-wd')?.textContent;

		expect(weekday).toBe('Fr');
	});

	it('marks the column as today when its date matches today', () => {
		render(TapeChartDateHeader, { date: '2026-08-20', today: '2026-08-20' });

		const isMarkedToday = document.querySelector('.header-cell')?.classList.contains('today');

		expect(isMarkedToday).toBe(true);
	});

	it('does not mark the column as today when its date differs from today', () => {
		render(TapeChartDateHeader, { date: '2026-08-21', today: '2026-08-20' });

		const isMarkedToday = document.querySelector('.header-cell')?.classList.contains('today');

		expect(isMarkedToday).toBe(false);
	});
});
