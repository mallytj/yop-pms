import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import { topBarStore } from '$stores/topbar.svelte';
import { shellStore } from '$stores/shell.svelte';
import TopBar from './TopBar.svelte';

describe('TopBar', () => {
	afterEach(() => {
		cleanup();
		topBarStore.reset();
	});

	it('displays the topLeft text from shell store', () => {
		shellStore.topLeft = 'Dashboard';
		render(TopBar);
		const topbar = document.querySelector('.topbar');
		expect(topbar?.textContent).toContain('Dashboard');
	});

	it('renders tabs from topBarStore', () => {
		topBarStore.tabs = [
			{ id: 'reservations', label: 'Reservations' },
			{ id: 'maintenance', label: 'Maintenance' },
			{ id: 'rates', label: 'Rates' }
		];
		render(TopBar);
		expect(document.querySelectorAll('.role-tab').length).toBe(3);
	});

	it('highlights the active tab', () => {
		topBarStore.tabs = [
			{ id: 'reservations', label: 'Reservations' },
			{ id: 'maintenance', label: 'Maintenance' },
			{ id: 'rates', label: 'Rates' }
		];
		topBarStore.active = 'maintenance';
		render(TopBar);
		const active = document.querySelector('.role-tab.active');
		expect(active?.textContent).toContain('Maintenance');
	});

	it('calls store onchange when tab is clicked', async () => {
		let changed = '';
		topBarStore.tabs = [
			{ id: 'reservations', label: 'Reservations' },
			{ id: 'maintenance', label: 'Maintenance' }
		];
		topBarStore.onchange = (id: string) => (changed = id);
		render(TopBar);
		(document.querySelectorAll('.role-tab')[1] as HTMLElement).click();
		expect(changed).toBe('maintenance');
	});
});
