import { describe, it, expect } from 'vitest';
import { tapeChartView } from './tapeChartView.svelte';

describe('tapeChartView', () => {
	it('increments scrollToTodayRequestId each time a scroll-to-today is requested', () => {
		const before = tapeChartView.scrollToTodayRequestId;
		tapeChartView.requestScrollToToday();
		expect(tapeChartView.scrollToTodayRequestId).toBe(before + 1);
		tapeChartView.requestScrollToToday();
		expect(tapeChartView.scrollToTodayRequestId).toBe(before + 2);
	});
});
