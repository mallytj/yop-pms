import { describe, it, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte/svelte5';

vi.mock('$app/state', () => ({
	page: {
		url: new URL('http://localhost/tape-chart')
	}
}));
import { topBar } from '$stores/topbar.svelte';
import { tapeChartView } from '$stores/tapeChartView.svelte';
import TopBar from './TopBar.svelte';

function givenTabs(tabs: { id: string; label: string }[]) {
	topBar.tabs = tabs.map((tab) => ({ ...tab, icon: (() => null) as never }));
}

describe('TopBar', () => {
	afterEach(() => {
		cleanup();
		topBar.reset();
	});

	it('renders one tab per entry in shared tape-chart tab state', () => {
		givenTabs([
			{ id: 'reservations', label: 'Reservations' },
			{ id: 'maintenance', label: 'Maintenance' }
		]);

		render(TopBar);

		expect(document.querySelectorAll('[role="tab"]').length).toBe(2);
	});

	it('marks the active tab as selected', () => {
		givenTabs([
			{ id: 'reservations', label: 'Reservations' },
			{ id: 'maintenance', label: 'Maintenance' }
		]);
		topBar.active = 'maintenance';

		render(TopBar);

		expect(document.querySelector('[aria-selected="true"]')?.textContent).toContain('Maintenance');
	});

	it('shows date range controls on the tape-chart route', () => {
		givenTabs([{ id: '', label: 'Reservations' }]);

		render(TopBar);

		expect(document.querySelector('[aria-label="Tape Chart dates"]')).toBeTruthy();
		expect(document.querySelector('[aria-label="Tape Chart start date"]')).toBeTruthy();
	});

	it('requests a scroll-to-today when the Today button is clicked, so an already-scrolled grid snaps back', async () => {
		givenTabs([{ id: '', label: 'Reservations' }]);
		render(TopBar);
		const before = tapeChartView.scrollToTodayRequestId;
		const todayButton = document.querySelector('button.today') as HTMLButtonElement;

		expect(todayButton).toBeTruthy();
		await fireEvent.click(todayButton);

		expect(tapeChartView.scrollToTodayRequestId).toBe(before + 1);
	});
});
