import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte/svelte5';
import AppSidebar from './AppSidebar.svelte';

function collapseToggle(container: HTMLElement): HTMLButtonElement {
	return container.querySelector('button.collapse-toggle') as HTMLButtonElement;
}

async function givenCollapsedSidebar() {
	const { container } = render(AppSidebar);
	const toggle = collapseToggle(container);

	await fireEvent.click(toggle);

	return { container, toggle };
}

describe('AppSidebar', () => {
	it('renders a Tape Chart link pointing to /tape-chart', () => {
		const { container } = render(AppSidebar);

		const link = container.querySelector('a[href="/tape-chart"]');

		expect(link).toBeTruthy();
		expect(link?.textContent).toContain('Tape Chart');
	});

	it('renders a Housekeeping link', () => {
		const { container } = render(AppSidebar);

		const link = container.querySelector('a[href="/housekeeping"]');

		expect(link).toBeTruthy();
		expect(link?.textContent).toContain('Housekeeping');
	});

	it('renders an icon for every nav item', () => {
		const { container } = render(AppSidebar);

		const icons = container.querySelectorAll('svg');

		expect(icons.length).toBeGreaterThanOrEqual(5);
	});

	it('renders the not-yet-implemented nav items as disabled buttons', () => {
		const { container } = render(AppSidebar);

		const disabledItems = container.querySelectorAll('button.nav-item');

		expect(disabledItems.length).toBe(3);
	});

	it('starts expanded', () => {
		const { container } = render(AppSidebar);

		const toggle = collapseToggle(container);

		expect(toggle.getAttribute('aria-expanded')).toBe('true');
		expect(toggle.getAttribute('aria-label')).toMatch(/collapse sidebar/i);
	});

	it('collapses when the toggle is clicked', async () => {
		const { container, toggle } = await givenCollapsedSidebar();

		expect(container.querySelector('aside')?.getAttribute('data-collapsed')).toBe('true');
	});

	it('flips the toggle to an expand control once collapsed', async () => {
		const { toggle } = await givenCollapsedSidebar();

		expect(toggle.getAttribute('aria-expanded')).toBe('false');
		expect(toggle.getAttribute('aria-label')).toMatch(/expand sidebar/i);
	});

	it('hides nav labels from the accessibility tree when collapsed', async () => {
		const { container } = await givenCollapsedSidebar();

		const tapeChartLink = container.querySelector('a[href="/tape-chart"]');

		expect(tapeChartLink?.querySelector('span')?.getAttribute('aria-hidden')).toBe('true');
	});

	it('shows the nav label as a tooltip when collapsed', async () => {
		const { container } = await givenCollapsedSidebar();

		const tapeChartLink = container.querySelector('a[href="/tape-chart"]');

		expect(tapeChartLink?.getAttribute('title')).toBe('Tape Chart');
	});
});
