import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import TapeChartSkeletonBody from './TapeChartSkeletonBody.svelte';

describe('TapeChartSkeletonBody', () => {
	afterEach(() => {
		cleanup();
	});

	it('renders one skeleton cell per visible column, for every skeleton row', () => {
		render(TapeChartSkeletonBody, { columnCount: 5 });
		const rows = document.querySelectorAll('.skeleton-row');
		expect(rows.length).toBeGreaterThan(0);
		for (const row of rows) {
			expect(row.querySelectorAll('.skeleton-cell').length).toBe(5);
			expect(row.querySelectorAll('.skeleton-room').length).toBe(1);
		}
	});
});
