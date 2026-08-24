import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import MaintenanceBlock from './MaintenanceBlock.svelte';

function givenMaintenanceBlock(blockType: string) {
	return { id: 'block-1', blockType };
}

describe('MaintenanceBlock', () => {
	afterEach(() => {
		cleanup();
	});

	it('formats a snake_case block type as a capitalized, spaced label', () => {
		const block = givenMaintenanceBlock('deep_clean');

		render(MaintenanceBlock, { block, startIndex: 2, span: 3 });

		expect(document.querySelector('.maintenance-block strong')?.textContent).toBe('Deep clean');
	});

	it('falls back to "Maintenance" when no block type is known', () => {
		const block = givenMaintenanceBlock('');

		render(MaintenanceBlock, { block, startIndex: 0, span: 1 });

		expect(document.querySelector('.maintenance-block strong')?.textContent).toBe('Maintenance');
	});

	it('positions the block from the room column using startIndex and span', () => {
		const block = givenMaintenanceBlock('repair');

		render(MaintenanceBlock, { block, startIndex: 4, span: 2 });

		const element = document.querySelector('.maintenance-block') as HTMLElement;
		expect(element.style.left).toBe('calc(var(--room-w) + 4 * var(--cell-w) + 4px)');
		expect(element.style.width).toBe('calc(2 * var(--cell-w) - 8px)');
	});
});
