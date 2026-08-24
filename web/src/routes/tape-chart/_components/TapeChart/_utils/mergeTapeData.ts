import type { TapeChartData } from '$types/tape-chart.js';
import { maxDateKey, minDateKey, toDateKey } from '$helpers/dates.js';
import { normalizeTapeData, type NormalizedTapeChartData } from './normalizeTapeData.js';

function byId<T extends { id?: string }>(items: T[]): Map<string, T> {
	const map = new Map<string, T>();
	for (const item of items) {
		if (item.id) map.set(item.id, item);
	}
	return map;
}

export function mergeTapeData(
	existing: NormalizedTapeChartData,
	incomingRaw: TapeChartData
): NormalizedTapeChartData {
	const incoming = normalizeTapeData(incomingRaw);

	const reservationsById = byId([...existing.reservations, ...incoming.reservations]);
	const maintenanceBlocksById = byId([
		...existing.maintenanceBlocks,
		...incoming.maintenanceBlocks
	]);
	const inventoryByRoomAndDate = new Map(
		[...existing.inventory, ...incoming.inventory].map((day) => [
			`${day.room_id ?? ''}|${toDateKey(day.calendar_date)}`,
			day
		])
	);

	return {
		from: minDateKey(existing.from, incoming.from),
		to: maxDateKey(existing.to, incoming.to),
		// Room metadata only arrives on the initial full fetch — incremental
		// (shallow) loads never carry it, so once set it doesn't change.
		roomTypes: existing.roomTypes.length > 0 ? existing.roomTypes : incoming.roomTypes,
		reservations: [...reservationsById.values()],
		maintenanceBlocks: [...maintenanceBlocksById.values()],
		inventory: [...inventoryByRoomAndDate.values()]
	};
}
