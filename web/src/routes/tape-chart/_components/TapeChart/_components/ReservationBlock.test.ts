import { describe, it, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import ReservationBlock from './ReservationBlock.svelte';
import type { ReservationCellData } from './RoomAvailabilityCell.svelte';

function givenReservation(overrides: Partial<ReservationCellData> = {}): ReservationCellData {
	return {
		code: 'RES-1',
		guestName: 'Ada Lovelace',
		status: 'confirmed',
		...overrides
	};
}

describe('ReservationBlock', () => {
	afterEach(() => {
		cleanup();
	});

	it('shows the guest name and the price formatted as GBP', () => {
		const reservation = givenReservation({ pricePence: 12500 });

		render(ReservationBlock, { reservation, startIndex: 2, span: 3 });

		expect(document.querySelector('.reservation-block strong')?.textContent).toBe('Ada Lovelace');
		expect(document.querySelector('.reservation-block span')?.textContent).toBe('£125');
	});

	it('falls back to the reservation code when no price is known', () => {
		const reservation = givenReservation({ pricePence: undefined });

		render(ReservationBlock, { reservation, startIndex: 0, span: 1 });

		expect(document.querySelector('.reservation-block span')?.textContent).toBe('RES-1');
	});

	it('flags an on-hold reservation with the hold class', () => {
		const reservation = givenReservation({ status: 'on_hold' });

		render(ReservationBlock, { reservation, startIndex: 0, span: 1 });

		expect(document.querySelector('.reservation-block')?.classList.contains('hold')).toBe(true);
	});

	it('positions the block from the room column using startIndex and span', () => {
		const reservation = givenReservation();

		render(ReservationBlock, { reservation, startIndex: 4, span: 2 });

		const block = document.querySelector('.reservation-block') as HTMLElement;
		expect(block.style.left).toBe('calc(var(--room-w) + 4 * var(--cell-w) + 4px)');
		expect(block.style.width).toBe('calc(2 * var(--cell-w) - 8px)');
	});
});
