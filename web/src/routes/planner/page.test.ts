import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, cleanup } from '@testing-library/svelte/svelte5';
import type { Snippet } from 'svelte';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$app/stores', () => ({
	page: {
		subscribe(fn: Function) {
			fn({ url: { pathname: '/planner/reservations' } });
			return () => {};
		}
	}
}));

import PlannerLayout from './+layout.svelte';
import { topBarStore } from '$stores/topbar.svelte';
import { goto } from '$app/navigation';
import TopBar from '$components/TopBar.svelte';
import { shellStore } from '$stores/shell.svelte';

const children = (() => document.createTextNode('')) as unknown as Snippet;

describe('Planner layout - tab system', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		topBarStore.reset();
		shellStore.topLeft = 'The Ollerod';
	});

	it('sets topBarStore tabs to Reservations, Maintenance, Rates', () => {
		render(PlannerLayout, { props: { children } });
		expect(topBarStore.tabs.length).toBe(3);
		expect(topBarStore.tabs[0].label).toBe('Reservations');
		expect(topBarStore.tabs[1].label).toBe('Maintenance');
		expect(topBarStore.tabs[2].label).toBe('Rates');
	});

	it('sets topBarStore active to current URL segment', () => {
		render(PlannerLayout, { props: { children } });
		expect(topBarStore.active).toBe('reservations');
	});

	it('renders tabs via TopBar after store is set', () => {
		render(PlannerLayout, { props: { children } });
		render(TopBar);
		const tabs = document.querySelectorAll('.role-tab');
		expect(tabs.length).toBe(3);
		expect(tabs[0].textContent).toContain('Reservations');
	});

	it('highlights active tab via TopBar', () => {
		render(PlannerLayout, { props: { children } });
		render(TopBar);
		const active = document.querySelector('.role-tab.active');
		expect(active?.textContent).toContain('Reservations');
	});

	it('calls goto when tab is clicked', async () => {
		topBarStore.tabs = [
			{ id: 'reservations', label: 'Reservations' },
			{ id: 'maintenance', label: 'Maintenance' },
			{ id: 'rates', label: 'Rates' }
		];
		topBarStore.onchange = (id: string) => goto(`/planner/${id}`);
		render(TopBar);
		const tabs = document.querySelectorAll('.role-tab');

		await fireEvent.click(tabs[1]);
		expect(vi.mocked(goto)).toHaveBeenCalledWith('/planner/maintenance');
	});
});
