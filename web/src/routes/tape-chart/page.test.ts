import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte/svelte5';
import type { Snippet } from 'svelte';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$app/state', () => ({
	page: {
		url: new URL('http://localhost/tape-chart/reservations')
	}
}));

import TapeChartLayout from './+layout.svelte';
import { topBar, type TabDef } from '$stores/topbar.svelte';
import TopBar from '$components/TopBar.svelte';

const children = (() => document.createTextNode('')) as unknown as Snippet;
const noIcon = (() => null) as never as TabDef['icon'];

function renderLayout() {
	return render(TapeChartLayout, { props: { children } });
}

describe('TapeChart layout - tab system', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		topBar.reset();
	});

	it('sets topBar tabs to the three tape-chart sections in order', () => {
		renderLayout();

		expect(topBar.tabs.map((tab) => tab.label)).toEqual(['Reservations', 'Maintenance', 'Rates']);
	});

	it('sets topBar active to the current route segment from the URL', () => {
		// Mocked page.url is /tape-chart/reservations → active = 'reservations'
		renderLayout();

		expect(topBar.active).toBe('reservations');
	});

	it('renders one tab link per configured tab via TopBar', () => {
		renderLayout();

		render(TopBar);

		const tabs = document.querySelectorAll('.tab');
		expect(tabs.length).toBe(3);
	});

	it('renders tab links with hrefs built from each tab id', () => {
		topBar.tabs = [
			{ id: '', label: 'Reservations', icon: noIcon },
			{ id: 'reservations', label: 'Reservations', icon: noIcon }
		];
		topBar.active = '';

		render(TopBar);

		const links = document.querySelectorAll('.tab');
		expect(links[0].getAttribute('href')).toBe('/tape-chart');
		expect(links[1].getAttribute('href')).toBe('/tape-chart/reservations');
	});
});
