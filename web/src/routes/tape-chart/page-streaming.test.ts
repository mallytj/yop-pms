import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup, waitFor } from '@testing-library/svelte/svelte5';
import TapeChartPage from './+page.svelte';
import type { TapeChartData } from '$types/tape-chart.js';
import { asISO8601Date } from '$helpers/dates.js';

function deferred<T>() {
	let resolve!: (value: T) => void;
	let reject!: (reason: unknown) => void;
	const promise = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

describe('tape-chart +page.svelte', () => {
	afterEach(() => {
		cleanup();
	});

	function renderWithStream(tapeChart: Promise<TapeChartData>) {
		return render(TapeChartPage, {
			props: {
				data: { from: asISO8601Date('2026-08-20'), to: asISO8601Date('2026-10-09'), tapeChart }
			}
		});
	}

	function tapeChartDataWithOneRoom(): TapeChartData {
		return {
			from: '2026-08-20' as never,
			to: '2026-10-09' as never,
			room_types: [
				{
					id: 'rt-1',
					name: 'Standard',
					rooms: [{ id: 'room-1', name: '101', reservations: [], maintenance_blocks: [] }]
				} as never
			],
			inventory: []
		};
	}

	it('renders the grid header immediately, before the streamed data resolves', () => {
		const { promise } = deferred<TapeChartData>();

		renderWithStream(promise);

		expect(document.querySelector('[role="grid"]')).toBeTruthy();
	});

	it('shows the loading skeleton before the streamed data resolves', () => {
		const { promise } = deferred<TapeChartData>();

		renderWithStream(promise);

		expect(document.querySelector('.skeleton-body')).toBeTruthy();
	});

	it('swaps the skeleton for real data once the stream resolves', async () => {
		const { promise, resolve } = deferred<TapeChartData>();
		renderWithStream(promise);

		resolve(tapeChartDataWithOneRoom());

		await waitFor(() => {
			expect(document.querySelector('.skeleton-body')).toBeNull();
		});
	});

	it('does not show the empty state once the stream resolves with data', async () => {
		const { promise, resolve } = deferred<TapeChartData>();
		renderWithStream(promise);

		resolve(tapeChartDataWithOneRoom());

		// Row rendering itself is virtualizer-driven and needs real layout
		// (jsdom reports zero size), so assert on the non-virtualized header
		// instead of a specific room row.
		await waitFor(() => {
			expect(document.querySelector('.empty-state')).toBeNull();
		});
		expect(document.querySelector('[role="grid"]')).toBeTruthy();
	});

	it('shows an error title when the stream rejects', async () => {
		const { promise, reject } = deferred<TapeChartData>();
		renderWithStream(promise);

		reject(new Error('backend unreachable'));

		await waitFor(() => {
			expect(document.querySelector('.empty-title')?.textContent).toBe('Failed to load tape chart');
		});
	});

	it('shows the rejection message as a hint when the stream rejects', async () => {
		const { promise, reject } = deferred<TapeChartData>();
		renderWithStream(promise);

		reject(new Error('backend unreachable'));

		await waitFor(() => {
			expect(document.querySelector('.empty-hint')?.textContent).toBe('backend unreachable');
		});
	});
});
