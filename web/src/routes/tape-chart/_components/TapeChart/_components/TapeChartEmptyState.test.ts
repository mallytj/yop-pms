import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import TapeChartEmptyState from './TapeChartEmptyState.svelte';

describe('TapeChartEmptyState', () => {
	afterEach(() => {
		cleanup();
	});

	it('shows the title and hint', () => {
		render(TapeChartEmptyState, {
			title: 'No rooms to display',
			hint: 'Add a room to this property to see its availability.'
		});
		expect(document.querySelector('.empty-title')?.textContent).toBe('No rooms to display');
		expect(document.querySelector('.empty-hint')?.textContent).toBe(
			'Add a room to this property to see its availability.'
		);
	});
});
