import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import RoomAvailabilityCell from './RoomAvailabilityCell.svelte';

describe('RoomAvailabilityCell', () => {
	afterEach(() => {
		cleanup();
	});

	it('labels an available cell with the room, date, and status', () => {
		render(RoomAvailabilityCell, {
			roomName: '101',
			date: '2026-08-21',
			status: 'available',
			today: '2026-08-20'
		});

		const label = document.querySelector('.cell')?.getAttribute('aria-label');

		expect(label).toBe('101 · 2026-08-21 · available');
	});

	it('includes the reservation code and price in the label for a booked cell', () => {
		render(RoomAvailabilityCell, {
			roomName: '102',
			date: '2026-08-21',
			status: 'sold',
			reservation: { code: 'RES-9', guestName: 'Ada Lovelace', pricePence: 9900 },
			today: '2026-08-20'
		});

		const label = document.querySelector('.cell')?.getAttribute('aria-label');

		expect(label).toBe('102 · 2026-08-21 · RES-9 · £99');
	});

	it('marks the current day with the today class', () => {
		render(RoomAvailabilityCell, {
			roomName: '101',
			date: '2026-08-20',
			status: 'available',
			today: '2026-08-20'
		});

		const isToday = document.querySelector('.cell')?.classList.contains('today');

		expect(isToday).toBe(true);
	});

	it('marks an on-hold status with the hold class', () => {
		render(RoomAvailabilityCell, {
			roomName: '101',
			date: '2026-08-21',
			status: 'on_hold',
			today: '2026-08-20'
		});

		const isHold = document.querySelector('.cell')?.classList.contains('hold');

		expect(isHold).toBe(true);
	});
});
