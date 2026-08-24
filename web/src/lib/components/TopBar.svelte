<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { topBar } from '$stores/topbar.svelte';
	import { tapeChartView } from '$stores/tapeChartView.svelte';
	import { goWithParams } from '$helpers/url.js';
	import TapeChartDatePicker from './TapeChartDatePicker.svelte';

	let urlFrom = $derived(page.url.searchParams.get('from') || '');
	let hasTabs = $derived(topBar.tabs.length > 0);
	let isTapeChart = $derived(page.url.pathname.startsWith('/tape-chart'));

	function navigateWithParams(params: Record<string, string>) {
		goto(goWithParams(page.url, params), { replaceState: true, keepFocus: true });
	}

	function tabHref(id: string): string {
		return id ? `/tape-chart/${id}` : '/tape-chart';
	}
</script>

<header class="top-bar">
	<div class="top-bar-left">
		{#if hasTabs}
			<h1 class="top-bar-title">Tape Chart</h1>
		{/if}

		{#if isTapeChart}
			<TapeChartDatePicker
				from={urlFrom}
				onRangeChange={(from, to) => navigateWithParams({ from, to })}
				onToday={() => tapeChartView.requestScrollToToday()}
			/>
		{/if}
	</div>

	{#if topBar.tabs.length > 0}
		<div class="top-bar-tabs" role="tablist" aria-label="Tape Chart tabs">
			{#each topBar.tabs as tab}
				<a
					href={tabHref(tab.id)}
					class="tab"
					class:active={topBar.active === tab.id}
					role="tab"
					aria-selected={topBar.active === tab.id}
				>
					<tab.icon size={16} />
					<span>{tab.label}</span>
				</a>
			{/each}
		</div>
	{/if}
</header>

<style>
	.top-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: var(--topbar-height);
		padding: 0 var(--spacing-lg);
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
		gap: var(--spacing-lg);
		flex-shrink: 0;
	}

	.top-bar-left {
		display: flex;
		align-items: center;
		gap: var(--spacing-lg);
		min-width: 0;
		flex: 1;
	}

	.top-bar-title {
		font-size: var(--font-size-lg);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text);
		white-space: nowrap;
		margin: 0;
	}

	.top-bar-tabs {
		display: flex;
		align-items: center;
		height: 100%;
		gap: 0;
		flex-shrink: 0;
	}

	.tab {
		display: flex;
		align-items: center;
		gap: var(--spacing-sm);
		height: 100%;
		padding: 0 var(--spacing-lg);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-secondary);
		text-decoration: none;
		border-bottom: 2px solid transparent;
		cursor: pointer;
		transition:
			color 0.15s,
			border-color 0.15s,
			background 0.15s;
		white-space: nowrap;
		font-family: inherit;
	}

	.tab:hover {
		color: var(--color-text);
		background: var(--color-surface-raised-strong);
	}

	.tab.active {
		color: var(--color-accent-strong);
		border-bottom-color: var(--color-accent-strong);
	}

	.tab :global(svg) {
		display: block;
		flex-shrink: 0;
	}
</style>
