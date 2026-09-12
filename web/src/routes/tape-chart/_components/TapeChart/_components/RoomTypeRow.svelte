<script lang="ts">
	import { isMonthStart } from '$helpers/dates.js';
	import ColumnResizeHandle from './ColumnResizeHandle.svelte';

	interface Props {
		name: string;
		dates: string[];
		/**
		 * Number of date columns `dates` is offset from the grid's global date
		 * range. Data can load in batches that extend the range backwards, so
		 * this aligns this row's local column indices with the grid's global
		 * column indices via a leading spacer of that width.
		 */
		dateOffset?: number;
		roomColumnWidth: number;
		onRoomColumnResize: (width: number) => void;
	}

	let { name, dates, dateOffset = 0, roomColumnWidth, onRoomColumnResize }: Props = $props();
</script>

<div class="row type-row" role="row">
	<div class="type-label" role="rowheader">
		{name}
		<ColumnResizeHandle
			width={roomColumnWidth}
			onResize={onRoomColumnResize}
			label="Resize room column"
		/>
	</div>
	<div class="date-spacer" style:width={`calc(${dateOffset} * var(--cell-w))`}></div>
	{#each dates as date (date)}
		<div class="type-fill" class:month-start={isMonthStart(date)}></div>
	{/each}
</div>

<style>
	.row {
		display: flex;
		flex-direction: row;
		flex-shrink: 0;
	}

	.type-row {
		background: var(--color-surface-raised-strong);
	}

	.type-label {
		position: sticky;
		left: 0;
		z-index: 20;
		flex: 0 0 var(--room-w);
		width: var(--room-w);
		min-width: var(--room-w);
		max-width: var(--room-w);
		height: var(--row-h);
		display: flex;
		align-items: center;
		padding: 0 var(--spacing-sm);
		background: var(--color-surface-raised-strong);
		border-right: 1px solid var(--color-border-strong);
		border-bottom: 1px solid var(--color-border);
		box-shadow: 1px 0 0 0 var(--color-border);
		font-size: var(--font-size-2xs);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-secondary);
		text-transform: uppercase;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.date-spacer {
		flex: 0 0 auto;
		height: var(--row-h);
	}

	.type-fill {
		flex: 0 0 var(--cell-w);
		width: var(--cell-w);
		min-width: var(--cell-w);
		max-width: var(--cell-w);
		height: var(--row-h);
		border-bottom: 1px solid var(--color-border);
		background: var(--color-surface-raised-strong);
	}

	.type-fill.month-start {
		box-shadow: inset 2px 0 0 0 var(--color-border-strong);
	}
</style>
