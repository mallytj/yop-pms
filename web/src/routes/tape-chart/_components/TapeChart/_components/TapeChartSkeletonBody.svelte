<script lang="ts">
	interface Props {
		/** Number of visible date columns, so skeleton cells line up with the real header. */
		columnCount: number;
	}

	let { columnCount }: Props = $props();
	const SKELETON_ROWS = 8;
</script>

<div class="skeleton-body" role="status" aria-label="Loading rooms">
	{#each { length: SKELETON_ROWS } as _, rowIndex (rowIndex)}
		<div class="skeleton-row">
			<span class="skeleton-bar skeleton-room"></span>
			{#each { length: columnCount } as _, cellIndex (cellIndex)}
				<span class="skeleton-bar skeleton-cell"></span>
			{/each}
		</div>
	{/each}
</div>

<style>
	.skeleton-body {
		display: flex;
		flex-direction: column;
	}

	.skeleton-row {
		display: flex;
		flex-direction: row;
		flex-shrink: 0;
		height: var(--row-h);
		align-items: center;
		gap: 2px;
		border-bottom: 1px solid var(--color-border);
	}

	.skeleton-bar {
		display: block;
		height: 20px;
		border-radius: 4px;
		background: var(--color-border-subtle);
		animation: skeleton-pulse 1.2s ease-in-out infinite;
	}

	.skeleton-room {
		flex: 0 0 calc(var(--room-w) - var(--spacing-sm) * 2);
		margin-left: var(--spacing-sm);
		position: sticky;
		left: var(--spacing-sm);
	}

	.skeleton-cell {
		flex: 0 0 calc(var(--cell-w) - 4px);
	}

	@keyframes skeleton-pulse {
		0%,
		100% {
			opacity: 0.5;
		}
		50% {
			opacity: 0.9;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.skeleton-bar {
			animation: none;
		}
	}
</style>
