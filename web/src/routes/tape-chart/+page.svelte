<script lang="ts">
	import type { PageData } from './$types.js';
	import TapeChartGrid from './_components/TapeChart/TapeChartGrid.svelte';
	import TapeChartEmptyState from './_components/TapeChart/_components/TapeChartEmptyState.svelte';
	import { fetchTapeChart } from '$lib/api/tape-chart.js';
	import { mergeTapeData } from './_components/TapeChart/_utils/mergeTapeData.js';
	import {
		normalizeTapeData,
		type NormalizedTapeChartData
	} from './_components/TapeChart/_utils/normalizeTapeData.js';

	let { data }: { data: PageData } = $props();

	let accumulatedData = $state<NormalizedTapeChartData | null>(null);
	let loadError = $state<string | null>(null);
	let loading = $state(false);

	// `data.from`/`data.to` resolve synchronously (see +page.server.ts), so
	// the grid can render its date header immediately. `data.tapeChart` is
	// streamed — room/reservation data fills in once it resolves.
	let shellData = $derived<NormalizedTapeChartData>({
		from: data.from,
		to: data.to,
		roomTypes: [],
		reservations: [],
		maintenanceBlocks: [],
		inventory: []
	});

	$effect(() => {
		accumulatedData = null;
		loadError = null;
		data.tapeChart
			.then((tapeChart) => {
				accumulatedData = normalizeTapeData(tapeChart);
			})
			.catch((err: unknown) => {
				loadError = err instanceof Error ? err.message : 'Failed to load tape chart';
			});
	});

	async function handleLoadMore(from: string, to: string) {
		if (loading || !accumulatedData) return;
		loading = true;

		try {
			const result = await fetchTapeChart(from, to, 'shallow');
			const incoming = result.data;
			if (!incoming) return;

			accumulatedData = mergeTapeData(accumulatedData, incoming);
		} catch (err) {
			console.error('Failed to load more tape chart data:', err);
			throw err;
		} finally {
			loading = false;
		}
	}
</script>

<div class="tape-chart-page">
	<div class="grid-wrap">
		{#if accumulatedData}
			<TapeChartGrid data={accumulatedData} onLoadMore={handleLoadMore} {loading} />
		{:else if loadError}
			<TapeChartEmptyState title="Failed to load tape chart" hint={loadError} />
		{:else}
			<TapeChartGrid data={shellData} loading={true} />
		{/if}
	</div>
</div>

<style>
	.tape-chart-page {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-height: 0;
		gap: var(--spacing-sm);
	}

	.grid-wrap {
		flex: 1;
		min-height: 0;
		position: relative;
	}
</style>
