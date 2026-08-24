<script lang="ts">
	import ColumnResizeHandle from './ColumnResizeHandle.svelte';

	interface Props {
		name: string;
		roomTypeName?: string;
		roomColumnWidth: number;
		onRoomColumnResize: (width: number) => void;
	}

	let { name, roomTypeName, roomColumnWidth, onRoomColumnResize }: Props = $props();
</script>

<div class="room-cell" role="rowheader" title={roomTypeName ? `${name} · ${roomTypeName}` : name}>
	<span class="room-name">{name}</span>
	{#if roomTypeName}<span class="room-type">{roomTypeName}</span>{/if}
	<ColumnResizeHandle
		width={roomColumnWidth}
		onResize={onRoomColumnResize}
		label="Resize room column"
	/>
</div>

<style>
	.room-cell {
		position: sticky;
		left: 0;
		z-index: 10;
		flex: 0 0 var(--room-w);
		width: var(--room-w);
		min-width: var(--room-w);
		max-width: var(--room-w);
		height: var(--row-h);
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		justify-content: center;
		padding: 0 var(--spacing-sm);
		background: var(--color-surface-raised);
		border-right: 1px solid var(--color-border-strong);
		border-bottom: 1px solid var(--color-border);
		box-shadow: 1px 0 0 0 var(--color-border);
		font-size: var(--font-size-2xs);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-secondary);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.room-name,
	.room-type {
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.room-type {
		font-size: var(--font-size-4xs);
		color: var(--color-text-muted);
	}

	.room-cell:hover {
		background: var(--color-surface-overlay);
		color: var(--color-text);
	}
</style>
