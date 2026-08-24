<script lang="ts">
	import RoomAvailabilityCell, { type ReservationCellData } from './RoomAvailabilityCell.svelte';
	import ReservationBlock from './ReservationBlock.svelte';
	import MaintenanceBlock, { type MaintenanceCellData } from './MaintenanceBlock.svelte';
	import RoomCell from './RoomCell.svelte';

	interface Room {
		id: string;
		name: string;
	}

	interface Props {
		room: Room;
		dates: string[];
		today: string;
		getCell: (
			roomId: string,
			date: string
		) => { status: string; reservation?: ReservationCellData };
		getMaintenanceCell?: (roomId: string, date: string) => MaintenanceCellData | undefined;
		dateOffset?: number;
		roomColumnWidth: number;
		onRoomColumnResize: (width: number) => void;
	}

	// `span` is the number of consecutive date columns a block covers — a
	// reservation or maintenance block spanning several nights renders as one
	// block stretched across `span` columns starting at `startIndex`.
	interface ReservationSpan {
		reservation: ReservationCellData;
		startIndex: number;
		span: number;
	}

	interface MaintenanceSpan {
		block: MaintenanceCellData;
		startIndex: number;
		span: number;
	}

	let {
		room,
		dates,
		today,
		getCell,
		getMaintenanceCell,
		dateOffset = 0,
		roomColumnWidth,
		onRoomColumnResize
	}: Props = $props();

	let reservationBlocks = $derived.by(() => {
		const blocks: ReservationSpan[] = [];
		for (let i = 0; i < dates.length; i += 1) {
			const reservation = getCell(room.id, dates[i]).reservation;
			if (!reservation) continue;

			const previous = blocks.at(-1);
			const isContinuationOfSameReservation =
				previous?.reservation.code === reservation.code &&
				previous.startIndex + previous.span === i + dateOffset;
			if (isContinuationOfSameReservation && previous) {
				previous.span += 1;
				continue;
			}

			blocks.push({ reservation, startIndex: i + dateOffset, span: 1 });
		}
		return blocks;
	});

	let maintenanceBlocks = $derived.by(() => {
		const blocks: MaintenanceSpan[] = [];
		if (!getMaintenanceCell) return blocks;

		for (let index = 0; index < dates.length; index += 1) {
			const block = getMaintenanceCell(room.id, dates[index]);
			if (!block) continue;

			const previous = blocks.at(-1);
			const isContinuationOfSameMaintenanceBlock =
				previous?.block.id === block.id &&
				previous.startIndex + previous.span === index + dateOffset;
			if (isContinuationOfSameMaintenanceBlock && previous) {
				previous.span += 1;
				continue;
			}

			blocks.push({ block, startIndex: index + dateOffset, span: 1 });
		}
		return blocks;
	});
</script>

<div class="row room-row" role="row">
	<RoomCell name={room.name} {roomColumnWidth} {onRoomColumnResize} />
	<div class="date-spacer" style:width={`calc(${dateOffset} * var(--cell-w))`}></div>
	{#each maintenanceBlocks as block (block.startIndex)}
		<MaintenanceBlock {...block} />
	{/each}

	{#each reservationBlocks as block (block.startIndex)}
		<ReservationBlock {...block} />
	{/each}

	{#each dates as date (date)}
		{@const cell = getCell(room.id, date)}
		<RoomAvailabilityCell
			roomName={room.name}
			{date}
			status={cell.status}
			reservation={cell.reservation}
			{today}
		/>
	{/each}
</div>

<style>
	.date-spacer {
		flex: 0 0 auto;
		height: var(--row-h);
	}

	.row {
		display: flex;
		flex-direction: row;
		flex-shrink: 0;
		position: relative;
	}
</style>
