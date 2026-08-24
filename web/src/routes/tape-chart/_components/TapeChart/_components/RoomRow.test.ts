import { describe, it, expect, afterEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/svelte/svelte5';
import RoomRow from './RoomRow.svelte';
import type { ReservationCellData } from './RoomAvailabilityCell.svelte';
import type { MaintenanceCellData } from './MaintenanceBlock.svelte';

const FOUR_CONSECUTIVE_DATES = ['2026-08-20', '2026-08-21', '2026-08-22', '2026-08-23'];

function givenReservationsOn(
	reservedDates: Record<string, ReservationCellData>
): (roomId: string, date: string) => { status: string; reservation?: ReservationCellData } {
	return (_roomId, date) => {
		const reservation = reservedDates[date];
		return reservation ? { status: 'sold', reservation } : { status: 'available' };
	};
}

function renderRoomRow(overrides: {
	getCell: ReturnType<typeof givenReservationsOn>;
	getMaintenanceCell?: (roomId: string, date: string) => MaintenanceCellData | undefined;
}) {
	return render(RoomRow, {
		room: { id: 'room-1', name: '101' },
		dates: FOUR_CONSECUTIVE_DATES,
		today: '2026-08-20',
		roomColumnWidth: 160,
		onRoomColumnResize: vi.fn(),
		...overrides
	});
}

describe('RoomRow', () => {
	afterEach(() => {
		cleanup();
	});

	it('renders one availability cell per date', () => {
		renderRoomRow({ getCell: givenReservationsOn({}) });

		expect(document.querySelectorAll('[role="gridcell"]').length).toBe(
			FOUR_CONSECUTIVE_DATES.length
		);
	});

	it('merges consecutive nights of the same reservation into one block', () => {
		const reservation: ReservationCellData = { code: 'RES-1', guestName: 'Ada Lovelace' };

		renderRoomRow({
			getCell: givenReservationsOn({ '2026-08-21': reservation, '2026-08-22': reservation })
		});

		const blocks = document.querySelectorAll('.reservation-block');

		expect(blocks.length).toBe(1);
		expect((blocks[0] as HTMLElement).style.width).toBe('calc(2 * var(--cell-w) - 8px)');
	});

	it('keeps separate reservations as separate blocks even when adjacent', () => {
		renderRoomRow({
			getCell: givenReservationsOn({
				'2026-08-21': { code: 'RES-1', guestName: 'Ada Lovelace' },
				'2026-08-22': { code: 'RES-2', guestName: 'Bo Diaz' }
			})
		});

		const blocks = document.querySelectorAll('.reservation-block');

		expect(blocks.length).toBe(2);
	});

	it('merges consecutive nights of the same maintenance block into one block', () => {
		renderRoomRow({
			getCell: givenReservationsOn({}),
			getMaintenanceCell: (_roomId, date) =>
				date === '2026-08-21' || date === '2026-08-22'
					? { id: 'block-1', blockType: 'repair' }
					: undefined
		});

		const blocks = document.querySelectorAll('.maintenance-block');

		expect(blocks.length).toBe(1);
		expect((blocks[0] as HTMLElement).style.width).toBe('calc(2 * var(--cell-w) - 8px)');
	});
});
