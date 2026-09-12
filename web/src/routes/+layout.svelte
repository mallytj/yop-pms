<script lang="ts">
	import TopBar from '$components/TopBar.svelte';
	import AppSidebar from '$components/AppSidebar.svelte';
	import '../app.css';

	let { children } = $props();
</script>

<div class="app-shell">
	<div class="layout">
		<AppSidebar />
		<main class="body">
			{@render children()}
		</main>
	</div>

	<!--
		Rendered after `.layout` (so nested route layouts — which push tab
		config into the shared topBar store — run first during SSR) and
		reordered to the top visually with `order`. Rendering it first in
		markup would read the store before a route had a chance to set it,
		leaving the header title/tabs blank until client-side hydration.
	-->
	<TopBar />
</div>

<style>
	:global(body) {
		background: var(--color-bg);
		color: var(--color-text);
		font-family: var(--font-sans);
		font-size: var(--font-size-base);
		-webkit-font-smoothing: antialiased;
	}

	.app-shell {
		flex: 1;
		display: flex;
		flex-direction: column;
		height: 100vh;
	}

	.app-shell > :global(.top-bar) {
		order: -1;
	}

	.layout {
		flex: 1;
		min-height: 0;
		display: flex;
	}

	.body {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		min-height: 0;
	}
</style>
