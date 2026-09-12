import { describe, it, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte/svelte5';
import RoomTypeRow from './RoomTypeRow.svelte';

describe('RoomTypeRow', () => {
	afterEach(() => {
		cleanup();
	});

	it('renders the type name and one fill cell per date', () => {
		render(RoomTypeRow, {
			name: 'Deluxe King',
			dates: ['2026-08-20', '2026-08-21', '2026-08-22'],
			roomColumnWidth: 160,
			onRoomColumnResize: vi.fn()
		});

		expect(document.querySelector('.type-label')?.textContent?.trim()).toBe('Deluxe King');
		expect(document.querySelectorAll('.type-fill').length).toBe(3);
	});

	it('reports the new column width when the resize handle is dragged', async () => {
		const onRoomColumnResize = vi.fn();
		render(RoomTypeRow, {
			name: 'Deluxe King',
			dates: [],
			roomColumnWidth: 160,
			onRoomColumnResize
		});
		const handle = document.querySelector('.column-resize-handle')!;

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerMove(window, { clientX: 120 });

		expect(onRoomColumnResize).toHaveBeenCalledWith(180);
	});
});
