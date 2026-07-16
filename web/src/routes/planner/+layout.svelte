<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { BedDouble, Wrench, Tag } from 'lucide-svelte';
	import { topBarStore } from '$stores/topbar.svelte';

	let { children } = $props();

	const TAPE_CHART_TABS = [
		{ id: 'reservations', label: 'Reservations', icon: BedDouble },
		{ id: 'maintenance', label: 'Maintenance', icon: Wrench },
		{ id: 'rates', label: 'Rates', icon: Tag }
	];

	$effect(() => {
		topBarStore.tabs = TAPE_CHART_TABS;
		topBarStore.active = $page.url.pathname.split('/')[2] ?? '';
		topBarStore.onchange = (id: string) => goto(`/planner/${id}`);
		return () => topBarStore.reset();
	});
</script>

{@render children()}
