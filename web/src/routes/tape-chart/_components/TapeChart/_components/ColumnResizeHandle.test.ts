import { describe, it, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte/svelte5';
import ColumnResizeHandle from './ColumnResizeHandle.svelte';

function givenHandle(onResize: (width: number) => void) {
	render(ColumnResizeHandle, { width: 160, onResize, label: 'Resize room column' });
	return document.querySelector('.column-resize-handle')!;
}

describe('ColumnResizeHandle', () => {
	afterEach(() => {
		cleanup();
	});

	it('resizes by the drag distance', async () => {
		const onResize = vi.fn();
		const handle = givenHandle(onResize);

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerMove(window, { clientX: 140 });

		expect(onResize).toHaveBeenCalledWith(200);
	});

	it('clamps the resized width to the minimum bound', async () => {
		const onResize = vi.fn();
		const handle = givenHandle(onResize);

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerMove(window, { clientX: -1000 });

		expect(onResize).toHaveBeenCalledWith(100);
	});

	it('clamps the resized width to the maximum bound', async () => {
		const onResize = vi.fn();
		const handle = givenHandle(onResize);

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerMove(window, { clientX: 5000 });

		expect(onResize).toHaveBeenCalledWith(320);
	});

	it('stops resizing once the pointer is released', async () => {
		const onResize = vi.fn();
		const handle = givenHandle(onResize);

		await fireEvent.pointerDown(handle, { clientX: 100 });
		await fireEvent.pointerUp(window, { clientX: 100 });
		onResize.mockClear();
		await fireEvent.pointerMove(window, { clientX: 200 });

		expect(onResize).not.toHaveBeenCalled();
	});
});
