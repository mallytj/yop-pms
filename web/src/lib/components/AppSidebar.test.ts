import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte/svelte5';
import AppSidebar from './AppSidebar.svelte';

describe('AppSidebar', () => {
	it('renders Tape Chart link pointing to /planner/reservations', () => {
		const { container } = render(AppSidebar);
		const link = container.querySelector('a[href="/planner/reservations"]');
		expect(link).toBeTruthy();
		expect(link?.textContent).toContain('Tape Chart');
	});

	it('renders Housekeeping link', () => {
		const { container } = render(AppSidebar);
		const link = container.querySelector('a[href="/housekeeping"]');
		expect(link).toBeTruthy();
		expect(link?.textContent).toContain('Housekeeping');
	});

	it('renders SVG icons', () => {
		const { container } = render(AppSidebar);
		expect(container.querySelectorAll('svg').length).toBeGreaterThanOrEqual(5);
	});

	it('renders 3 dummy buttons', () => {
		const { container } = render(AppSidebar);
		expect(container.querySelectorAll('button.nav-item').length).toBe(3);
	});
});
