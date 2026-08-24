import { describe, it, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte/svelte5';
import TapeChartHeaderRow from './TapeChartHeaderRow.svelte';

describe('TapeChartHeaderRow', () => {
	afterEach(() => {
		cleanup();
	});

	function renderWithHandle(onRoomColumnResize: (width: number) => void) {
		render(TapeChartHeaderRow, {
			dates: [],
			today: '2026-08-20',
			roomColumnWidth: 160,
			onRoomColumnResize
		});
		return document.querySelector('.column-resize-handle')!;
	}

	it('renders one column header per visible date, plus the corner header', () => {
		render(TapeChartHeaderRow, {
			dates: ['2026-08-20', '2026-08-21'],
			today: '2026-08-20',
			roomColumnWidth: 160,
			onRoomColumnResize: vi.fn()
		});

		expect(document.querySelectorAll('[role="columnheader"]').length).toBe(3);
	});

	it('reports the new column width when the resize handle is dragged within bounds', async () => {
		const onRoomColumnResize = vi.fn();
		const handle = renderWithHandle(onRoomColumnResize);

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerMove(window, { clientX: 140 });

		expect(onRoomColumnResize).toHaveBeenCalledWith(200);
	});

	it('clamps the resize to the minimum column width when dragged far left', async () => {
		const onRoomColumnResize = vi.fn();
		const handle = renderWithHandle(onRoomColumnResize);

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerMove(window, { clientX: -1000 });

		expect(onRoomColumnResize).toHaveBeenCalledWith(100);
	});

	it('clamps the resize to the maximum column width when dragged far right', async () => {
		const onRoomColumnResize = vi.fn();
		const handle = renderWithHandle(onRoomColumnResize);

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerMove(window, { clientX: 5000 });

		expect(onRoomColumnResize).toHaveBeenCalledWith(320);
	});
});
