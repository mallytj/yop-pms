import { describe, it, expect } from 'vitest';
import { mergeTapeData } from './mergeTapeData.js';
import type { TapeChartData } from '$types/tape-chart.js';
import type { NormalizedTapeChartData } from './normalizeTapeData.js';

function rawData(overrides: Partial<TapeChartData> = {}): TapeChartData {
	return {
		from: '2026-08-20' as never,
		to: '2026-10-09' as never,
		reservations: [],
		maintenance_blocks: [],
		inventory: [],
		...overrides
	};
}

function normalizedData(overrides: Partial<NormalizedTapeChartData> = {}): NormalizedTapeChartData {
	return {
		from: '2026-08-20',
		to: '2026-10-09',
		roomTypes: [{ id: 'rt-1', name: 'Standard', rooms: [{ id: 'room-1', name: '101' }] } as never],
		reservations: [],
		maintenanceBlocks: [],
		inventory: [],
		...overrides
	};
}

describe('mergeTapeData', () => {
	it('widens the date range to cover both inputs', () => {
		const existing = normalizedData({ from: '2026-08-20', to: '2026-10-09' });
		const incoming = rawData({ from: '2026-10-10' as never, to: '2026-11-08' as never });

		const merged = mergeTapeData(existing, incoming);

		expect(merged.from).toBe('2026-08-20');
		expect(merged.to).toBe('2026-11-08');
	});

	it('keeps room type metadata from the existing (server-loaded) data', () => {
		const existing = normalizedData();
		const incoming = rawData();

		const merged = mergeTapeData(existing, incoming);

		expect(merged.roomTypes).toEqual(existing.roomTypes);
	});

	it('falls back to incoming room type metadata when existing has none yet', () => {
		const existing = normalizedData({ roomTypes: [] });
		const incomingRoomTypes = [{ id: 'rt-2', name: 'Suite', rooms: [] } as never];
		const incoming = rawData({ room_types: incomingRoomTypes });

		const merged = mergeTapeData(existing, incoming);

		expect(merged.roomTypes).toEqual(incomingRoomTypes);
	});

	it('concatenates inventory rows that do not overlap', () => {
		const existing = normalizedData({
			inventory: [{ room_id: 'room-1', calendar_date: '2026-08-20', status: 'available' } as never]
		});
		const incoming = rawData({
			inventory: [{ room_id: 'room-1', calendar_date: '2026-08-21', status: 'available' } as never]
		});

		const merged = mergeTapeData(existing, incoming);

		expect(merged.inventory).toHaveLength(2);
	});

	it('keeps inventory rows for different rooms on the same date distinct', () => {
		const existing = normalizedData({
			inventory: [{ room_id: 'room-1', calendar_date: '2026-08-20', status: 'available' } as never]
		});
		const incoming = rawData({
			inventory: [{ room_id: 'room-2', calendar_date: '2026-08-20', status: 'available' } as never]
		});

		const merged = mergeTapeData(existing, incoming);

		expect(merged.inventory).toHaveLength(2);
	});

	it('collides an unset room_id with an explicit empty-string room_id on the same date', () => {
		const existing = normalizedData({
			inventory: [{ room_id: undefined, calendar_date: '2026-08-20', status: 'available' } as never]
		});
		const incoming = rawData({
			inventory: [{ room_id: '', calendar_date: '2026-08-20', status: 'sold' } as never]
		});

		const merged = mergeTapeData(existing, incoming);

		expect(merged.inventory).toEqual([
			{ room_id: '', calendar_date: '2026-08-20', status: 'sold' }
		]);
	});

	it('lets incoming inventory for the same room+date replace the existing row', () => {
		const existing = normalizedData({
			inventory: [{ room_id: 'room-1', calendar_date: '2026-08-20', status: 'available' } as never]
		});
		const incoming = rawData({
			inventory: [{ room_id: 'room-1', calendar_date: '2026-08-20', status: 'sold' } as never]
		});

		const merged = mergeTapeData(existing, incoming);

		expect(merged.inventory).toEqual([
			{ room_id: 'room-1', calendar_date: '2026-08-20', status: 'sold' }
		]);
	});

	it('deduplicates reservation blocks by id', () => {
		const shared = { id: 'res-1', room_id: 'room-1', status: 'confirmed' } as never;
		const existing = normalizedData({ reservations: [shared] });
		const incoming = rawData({ reservations: [shared] });

		const merged = mergeTapeData(existing, incoming);

		expect(merged.reservations).toHaveLength(1);
	});

	it('concatenates maintenance blocks from both sides when neither is empty', () => {
		const existing = normalizedData({
			maintenanceBlocks: [{ id: 'block-1', room_id: 'room-1' } as never]
		});
		const incoming = rawData({
			maintenance_blocks: [{ id: 'block-2', room_id: 'room-1' } as never]
		});

		const merged = mergeTapeData(existing, incoming);

		expect(merged.maintenanceBlocks).toHaveLength(2);
	});

	it('drops reservations/blocks with no id', () => {
		const existing = normalizedData({ reservations: [{ status: 'confirmed' } as never] });

		const merged = mergeTapeData(existing, rawData());

		expect(merged.reservations).toEqual([]);
	});
});
