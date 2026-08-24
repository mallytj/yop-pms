import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import TapeChartLegend from './TapeChartLegend.svelte';

describe('TapeChartLegend', () => {
	afterEach(() => {
		cleanup();
	});

	it('shows the visible date range', () => {
		render(TapeChartLegend, { rangeFrom: '2026-08-20', rangeTo: '2026-10-09' });

		const label = document.querySelector('.range-label')?.textContent ?? '';

		expect(label).toContain('20 Aug 2026');
		expect(label).toContain('9 Oct 2026');
	});

	it('does not show a loading spinner when no more days are being fetched', () => {
		render(TapeChartLegend, { rangeFrom: '2026-08-20', rangeTo: '2026-10-09' });

		expect(document.querySelector('.range-label [role="status"]')).toBeNull();
	});

	it('shows a loading spinner while more days are being fetched', () => {
		render(TapeChartLegend, { rangeFrom: '2026-08-20', rangeTo: '2026-10-09', loading: true });

		expect(document.querySelector('.range-label [role="status"]')).toBeTruthy();
	});

	it('renders the four status swatches', () => {
		render(TapeChartLegend, { rangeFrom: '2026-08-20', rangeTo: '2026-10-09' });

		expect(document.querySelectorAll('.legend-item').length).toBe(4);
	});
});
