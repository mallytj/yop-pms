<script lang="ts">
	import TapeChartDateHeader from './TapeChartDateHeader.svelte';
	import ColumnResizeHandle from './ColumnResizeHandle.svelte';

	interface Props {
		dates: string[];
		dateOffset?: number;
		today: string;
		roomColumnWidth: number;
		onRoomColumnResize: (width: number) => void;
	}

	let { dates, dateOffset = 0, today, roomColumnWidth, onRoomColumnResize }: Props = $props();
</script>

<div class="row header-row" role="row">
	<div class="corner-cell" role="columnheader">
		<span>Room</span>
		<ColumnResizeHandle
			width={roomColumnWidth}
			onResize={onRoomColumnResize}
			label="Resize room column"
		/>
	</div>
	<div class="date-spacer" style:width={`calc(${dateOffset} * var(--cell-w))`}></div>
	{#each dates as date (date)}
		<TapeChartDateHeader {date} {today} />
	{/each}
</div>

<style>
	.date-spacer {
		flex: 0 0 auto;
		height: var(--header-h);
	}

	.row {
		display: flex;
		flex-direction: row;
		flex-shrink: 0;
	}

	.header-row {
		position: sticky;
		top: 0;
		z-index: 30;
	}

	.corner-cell {
		position: sticky;
		left: 0;
		z-index: 50;
		flex: 0 0 var(--room-w);
		width: var(--room-w);
		min-width: var(--room-w);
		max-width: var(--room-w);
		min-height: var(--header-h);
		display: flex;
		align-items: center;
		gap: var(--spacing-sm);
		padding: 0 var(--spacing-sm);
		background: var(--color-surface-raised);
		border-right: 1px solid var(--color-border-strong);
		border-bottom: 1px solid var(--color-border-strong);
		box-shadow: 1px 0 0 0 var(--color-border-strong);
		font-size: var(--font-size-2xs);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-secondary);
	}
</style>
