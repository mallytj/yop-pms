<script lang="ts">
	import './tape-chart.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { topBar, type TabDef } from '$stores/topbar.svelte';
	import { Calendar, Wrench, Landmark } from '@lucide/svelte';
	import type { Component } from 'svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	const TABS: TabDef[] = [
		{ id: '', label: 'Reservations', icon: Calendar as Component },
		{ id: 'maintenance', label: 'Maintenance', icon: Wrench as Component },
		{ id: 'rates', label: 'Rates', icon: Landmark as Component }
	];

	function syncTopBar() {
		topBar.tabs = TABS;
		topBar.active = page.url.pathname.split('/')[2] ?? '';
		topBar.onchange = (id: string) => goto(`/tape-chart/${id}`);
	}

	// Called eagerly (not only inside $effect) so SSR renders the tabs/title
	// on first paint — $effect never runs during server rendering, which
	// otherwise left the header empty until client-side hydration.
	syncTopBar();

	$effect(() => {
		syncTopBar();
		return () => topBar.reset();
	});
</script>

<div class="tape-chart-root">
	{@render children()}
</div>

<style>
	.tape-chart-root {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-height: 0;
	}
</style>
