import type {
	TapeChartData,
	TapeChartInventoryDay,
	TapeChartMaintenanceBlock,
	TapeChartReservationBlock,
	TapeChartRoomTypeNode
} from '$types/tape-chart.js';
import { toDateKey } from '$helpers/dates.js';

export interface NormalizedTapeChartData {
	from: string;
	to: string;
	roomTypes: TapeChartRoomTypeNode[];
	reservations: TapeChartReservationBlock[];
	maintenanceBlocks: TapeChartMaintenanceBlock[];
	inventory: TapeChartInventoryDay[];
}

/**
 * The full response (initial load) nests reservations/maintenance blocks
 * under room_types → rooms. The shallow response (used to extend the loaded
 * range without re-fetching room metadata) returns them as flat arrays
 * instead. This flattens either wire shape into the one the grid renders.
 */
export function normalizeTapeData(data: TapeChartData): NormalizedTapeChartData {
	const roomTypes = data.room_types ?? [];
	const nestedRooms = roomTypes.flatMap((roomType) => roomType.rooms ?? []);

	return {
		from: toDateKey(data.from),
		to: toDateKey(data.to),
		roomTypes,
		reservations:
			nestedRooms.length > 0
				? nestedRooms.flatMap((room) => room.reservations ?? [])
				: (data.reservations ?? []),
		maintenanceBlocks:
			nestedRooms.length > 0
				? nestedRooms.flatMap((room) => room.maintenance_blocks ?? [])
				: (data.maintenance_blocks ?? []),
		inventory: data.inventory ?? []
	};
}
