import { describe, it, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte/svelte5';
import RoomCell from './RoomCell.svelte';

describe('RoomCell', () => {
	afterEach(() => {
		cleanup();
	});

	it('renders only the room name when no room type is given', () => {
		render(RoomCell, { name: '101', roomColumnWidth: 160, onRoomColumnResize: vi.fn() });

		expect(document.querySelector('.room-name')?.textContent).toBe('101');
		expect(document.querySelector('.room-type')).toBeNull();
	});

	it('renders the room type alongside the name when provided', () => {
		render(RoomCell, {
			name: '101',
			roomTypeName: 'Deluxe King',
			roomColumnWidth: 160,
			onRoomColumnResize: vi.fn()
		});

		expect(document.querySelector('.room-name')?.textContent).toBe('101');
		expect(document.querySelector('.room-type')?.textContent).toBe('Deluxe King');
		expect(document.querySelector('.room-cell')?.getAttribute('title')).toBe('101 · Deluxe King');
	});

	it('reports the new column width when the resize handle is dragged', async () => {
		const onRoomColumnResize = vi.fn();
		render(RoomCell, { name: '101', roomColumnWidth: 160, onRoomColumnResize });
		const handle = document.querySelector('.column-resize-handle')!;

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerMove(window, { clientX: 120 });

		expect(onRoomColumnResize).toHaveBeenCalledWith(180);
	});
});
