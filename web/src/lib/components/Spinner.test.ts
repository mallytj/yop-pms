import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import Spinner from './Spinner.svelte';

describe('Spinner', () => {
	afterEach(() => {
		cleanup();
	});

	it('renders at the default size when none is given', () => {
		render(Spinner);

		const spinner = document.querySelector('.spinner') as HTMLElement;

		expect(spinner.style.width).toBe('16px');
		expect(spinner.style.height).toBe('16px');
	});

	it('renders at a custom size when one is given', () => {
		render(Spinner, { size: 24 });

		const spinner = document.querySelector('.spinner') as HTMLElement;

		expect(spinner.style.width).toBe('24px');
		expect(spinner.style.height).toBe('24px');
	});

	it('exposes a status role with a default accessible label', () => {
		render(Spinner);

		const spinner = document.querySelector('[role="status"]');

		expect(spinner?.textContent).toBe('Loading');
	});

	it('exposes a custom accessible label when one is given', () => {
		render(Spinner, { label: 'Loading more dates' });

		const spinner = document.querySelector('[role="status"]');

		expect(spinner?.textContent).toBe('Loading more dates');
	});
});
