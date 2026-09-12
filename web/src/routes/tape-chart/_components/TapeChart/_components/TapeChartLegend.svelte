<script lang="ts">
	import { formatDayLabel } from '$helpers/dates.js';
	import Spinner from '$components/Spinner.svelte';

	interface Props {
		rangeFrom: string;
		rangeTo: string;
		/** More days are being fetched by the scroll-triggered loader. */
		loading?: boolean;
	}

	let { rangeFrom, rangeTo, loading = false }: Props = $props();
</script>

<div class="legend-bar">
	<div class="legend" aria-hidden="true">
		<span class="legend-item"><span class="swatch swatch-available"></span>Available</span>
		<span class="legend-item"><span class="swatch swatch-booked"></span>Booked</span>
		<span class="legend-item"><span class="swatch swatch-hold"></span>Hold</span>
		<span class="legend-item"><span class="swatch swatch-maintenance"></span>Maintenance</span>
	</div>
	<span class="range-label" aria-live="polite">
		{formatDayLabel(rangeFrom)} – {formatDayLabel(rangeTo)}
		{#if loading}
			<Spinner size={11} label="Loading more dates" />
		{/if}
	</span>
</div>

<style>
	.legend-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--spacing-md);
		padding: var(--spacing-md) var(--spacing-md);
		border-top: 1px solid var(--color-border);
		background: var(--color-surface-raised);
		flex-shrink: 0;
	}

	.legend {
		display: flex;
		align-items: center;
		gap: var(--spacing-md);
		min-width: 0;
		overflow: hidden;
	}

	.legend-item {
		display: inline-flex;
		align-items: center;
		gap: var(--spacing-sm);
		font-size: var(--font-size-2xs);
		color: var(--color-text-muted);
		white-space: nowrap;
	}

	.swatch {
		width: 10px;
		height: 10px;
		border-radius: 3px;
		flex-shrink: 0;
	}

	.swatch-available {
		background: var(--color-surface-raised-strong);
		border: 1px solid var(--color-border-strong);
	}

	.swatch-booked {
		background: var(--color-accent);
	}

	.swatch-hold {
		background: var(--color-warning-subtle);
		border: 1px dashed var(--color-warning-strong);
	}

	.swatch-maintenance {
		background: var(--color-danger-subtle);
		background-image: repeating-linear-gradient(
			-45deg,
			transparent,
			transparent 2px,
			var(--color-danger-strong) 2px,
			var(--color-danger-strong) 4px
		);
	}

	.range-label {
		display: inline-flex;
		align-items: center;
		gap: var(--spacing-sm);
		font-size: var(--font-size-2xs);
		color: var(--color-text-muted);
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
		flex-shrink: 0;
	}
</style>
