import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import { tick } from 'svelte';
import TapeChartGrid from './TapeChartGrid.svelte';
import { tapeChartView } from '$stores/tapeChartView.svelte';
import { initialScrollLeftForToday } from './_utils/scrollEdge.js';
import { buildDateRange } from '$helpers/dates.js';
import type { NormalizedTapeChartData } from './_utils/normalizeTapeData.js';

const TODAY = '2026-08-15';
const CELL_WIDTH_PX = 96;

function givenData(overrides: Partial<NormalizedTapeChartData> = {}): NormalizedTapeChartData {
	return {
		from: '2026-08-01',
		to: '2026-10-10',
		roomTypes: [],
		reservations: [],
		maintenanceBlocks: [],
		inventory: [],
		...overrides
	};
}

function renderGrid(
	overrides: { data?: NormalizedTapeChartData; loading?: boolean; today?: string } = {}
) {
	return render(TapeChartGrid, {
		data: overrides.data ?? givenData(),
		today: overrides.today ?? TODAY,
		...(overrides.loading !== undefined ? { loading: overrides.loading } : {})
	});
}

function getScrollElement(): HTMLDivElement {
	return document.querySelector('.tape-chart-scroll') as HTMLDivElement;
}

function expectedScrollLeftForToday(data: NormalizedTapeChartData, today: string): number {
	const dates = buildDateRange(data.from, data.to);
	return initialScrollLeftForToday(dates, today, CELL_WIDTH_PX);
}

describe('TapeChartGrid', () => {
	afterEach(() => {
		cleanup();
	});

	describe('scrolling to today', () => {
		it('positions the initial scroll at today on mount', async () => {
			const data = givenData();

			renderGrid({ data });
			await tick();

			const scrollElement = getScrollElement();
			expect(scrollElement.scrollLeft).toBe(expectedScrollLeftForToday(data, TODAY));
		});

		it('scrolls back to today when a scroll-to-today is requested after the operator has scrolled away', async () => {
			const data = givenData();
			renderGrid({ data });
			await tick();

			const scrollElement = getScrollElement();
			scrollElement.scrollLeft = 5000;
			tapeChartView.requestScrollToToday();
			await tick();

			expect(scrollElement.scrollLeft).toBe(expectedScrollLeftForToday(data, TODAY));
		});

		it('does not force a scroll on mount just because a prior grid instance already requested one', async () => {
			tapeChartView.requestScrollToToday();
			const requestIdBeforeMount = tapeChartView.scrollToTodayRequestId;

			const data = givenData();
			renderGrid({ data });
			await tick();

			const scrollElement = getScrollElement();
			scrollElement.scrollLeft = 5000;
			await tick();

			expect(tapeChartView.scrollToTodayRequestId).toBe(requestIdBeforeMount);
			expect(scrollElement.scrollLeft).toBe(5000);
		});

		it('honors a scroll-to-today request made before any dates had loaded', async () => {
			const emptyRangeData = givenData({ from: '', to: '' });
			const { rerender } = renderGrid({ data: emptyRangeData });
			await tick();

			tapeChartView.requestScrollToToday();
			await tick();

			const populatedData = givenData();
			await rerender({ data: populatedData, today: TODAY });
			await tick();

			expect(getScrollElement().scrollLeft).toBe(expectedScrollLeftForToday(populatedData, TODAY));
		});

		it('honors a scroll-to-today request made before the component has processed its first effect', async () => {
			const data = givenData();

			renderGrid({ data });
			// No `await tick()` yet: the component's script has run (so its
			// mount-time request-id baseline is captured) but the effect that
			// reacts to `scrollToTodayRequestId` has not flushed. This is the
			// earliest a caller could possibly call requestScrollToToday() and
			// still expect it to land on this instance.
			tapeChartView.requestScrollToToday();
			await tick();

			expect(getScrollElement().scrollLeft).toBe(expectedScrollLeftForToday(data, TODAY));
		});
	});

	describe('date ranges', () => {
		it('renders an empty state instead of the scrollable grid when there are no dates in range', () => {
			const data = givenData({ from: '', to: '' });

			const { container } = renderGrid({ data });

			expect(container.querySelector('.tape-chart-scroll')).toBeNull();
			expect(container.textContent).toContain('No dates in range');
		});

		it('positions the scroll without throwing when the range is a single day', async () => {
			const data = givenData({ from: TODAY, to: TODAY });

			renderGrid({ data });
			await tick();

			expect(getScrollElement().scrollLeft).toBe(expectedScrollLeftForToday(data, TODAY));
		});
	});

	describe('rooms still loading', () => {
		it('shows a loading skeleton instead of an empty state while data is still loading', () => {
			const data = givenData();

			const { container } = renderGrid({ data, loading: true });

			expect(container.textContent).not.toContain('Add a room to this property');
			expect(container.querySelector('[role="grid"]')).not.toBeNull();
		});

		it('shows a hint to add a room once loading finishes with no rooms in range', () => {
			const data = givenData();

			const { container } = renderGrid({ data, loading: false });

			expect(container.textContent).toContain(
				'Add a room to this property to see its availability.'
			);
		});
	});
});
