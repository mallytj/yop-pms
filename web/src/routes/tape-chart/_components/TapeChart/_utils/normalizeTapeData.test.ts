import { describe, it, expect } from 'vitest';
import { normalizeTapeData } from './normalizeTapeData.js';
import type { TapeChartData } from '$types/tape-chart.js';

function rawData(overrides: Partial<TapeChartData> = {}): TapeChartData {
	return {
		from: '2026-08-20' as never,
		to: '2026-10-09' as never,
		...overrides
	} as TapeChartData;
}

describe('normalizeTapeData', () => {
	it('defaults room_types to an empty array when absent', () => {
		const normalized = normalizeTapeData(rawData());

		expect(normalized.roomTypes).toEqual([]);
	});

	it('flattens reservations and maintenance blocks nested under room_types → rooms', () => {
		const reservation = { id: 'res-1', room_id: 'room-1' } as never;
		const maintenanceBlock = { id: 'block-1', room_id: 'room-1' } as never;
		const data = rawData({
			room_types: [
				{
					id: 'rt-1',
					name: 'Standard',
					rooms: [
						{
							id: 'room-1',
							name: '101',
							reservations: [reservation],
							maintenance_blocks: [maintenanceBlock]
						}
					]
				} as never
			]
		});

		const normalized = normalizeTapeData(data);

		expect(normalized.reservations).toEqual([reservation]);
		expect(normalized.maintenanceBlocks).toEqual([maintenanceBlock]);
	});

	it('defaults a nested room without reservations/maintenance_blocks to empty arrays', () => {
		const data = rawData({
			room_types: [
				{ id: 'rt-1', name: 'Standard', rooms: [{ id: 'room-1', name: '101' }] } as never
			]
		});

		const normalized = normalizeTapeData(data);

		expect(normalized.reservations).toEqual([]);
		expect(normalized.maintenanceBlocks).toEqual([]);
	});

	it('falls back to the flat reservations array when a room type has no nested rooms', () => {
		const reservation = { id: 'res-1', room_id: 'room-1' } as never;
		const data = rawData({
			room_types: [{ id: 'rt-1', name: 'Standard' } as never],
			reservations: [reservation]
		});

		const normalized = normalizeTapeData(data);

		expect(normalized.reservations).toEqual([reservation]);
	});

	it('falls back to the flat reservations/maintenance_blocks arrays when a room type has an empty rooms list', () => {
		const reservation = { id: 'res-1', room_id: 'room-1' } as never;
		const maintenanceBlock = { id: 'block-1', room_id: 'room-1' } as never;
		const data = rawData({
			room_types: [{ id: 'rt-1', name: 'Standard', rooms: [] } as never],
			reservations: [reservation],
			maintenance_blocks: [maintenanceBlock]
		});

		const normalized = normalizeTapeData(data);

		expect(normalized.reservations).toEqual([reservation]);
		expect(normalized.maintenanceBlocks).toEqual([maintenanceBlock]);
	});

	it('defaults flat reservations/maintenance_blocks/inventory to empty arrays when absent', () => {
		const normalized = normalizeTapeData(rawData());

		expect(normalized.reservations).toEqual([]);
		expect(normalized.maintenanceBlocks).toEqual([]);
		expect(normalized.inventory).toEqual([]);
	});

	it('normalizes the from/to timestamps into plain date keys', () => {
		const data = rawData({
			from: '2026-08-20T00:00:00Z' as never,
			to: '2026-10-09T00:00:00Z' as never
		});

		const normalized = normalizeTapeData(data);

		expect(normalized.from).toBe('2026-08-20');
		expect(normalized.to).toBe('2026-10-09');
	});
});
