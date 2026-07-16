<script lang="ts">
	import TopBar from '$components/TopBar.svelte';
	import AppSidebar from '$components/AppSidebar.svelte';
	import '../app.css';

	let { children } = $props();
</script>

<TopBar />

<div class="layout">
	<AppSidebar />
	<main class="body">
		{@render children()}
	</main>
</div>

<style>
	:global(body) {
		margin: 0;
		background: var(--color-bg);
		color: var(--color-text);
		font-family: var(--font-sans);
		font-size: var(--font-size-base);
		-webkit-font-smoothing: antialiased;
	}

	.layout {
		flex: 1;
		display: flex;
	}

	.body {
		flex: 1;
		padding: var(--spacing-xl) var(--spacing-2xl);
		display: flex;
		flex-direction: column;
		gap: var(--spacing-lg);
	}
</style>
