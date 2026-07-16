<script lang="ts">
	import { shellStore } from '$stores/shell.svelte';
	import { topBarStore } from '$stores/topbar.svelte';

	const tabs = $derived(topBarStore.tabs);
	const active = $derived(topBarStore.active);
	const onchange = $derived(topBarStore.onchange);
</script>

<nav class="topbar">
	<div class="topbar-left">
		<span class="topbar-title">{shellStore.topLeft}</span>
	</div>

	{#if tabs.length > 0}
		<div class="topbar-center">
			{#each tabs as tab (tab.id)}
				<button
					class="role-tab"
					class:active={active === tab.id}
					onclick={() => onchange(tab.id)}
				>
					{tab.label}
				</button>
			{/each}
		</div>
	{/if}

	<div class="topbar-right">
		<span>{shellStore.topRight}</span>
	</div>
</nav>

<style>
	.topbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: var(--topbar-height);
		border-bottom: 1px solid var(--color-border);
		padding: 0 var(--spacing-xl);
		font-size: var(--font-size-sm);
		color: var(--color-text-secondary);
	}

	.topbar-left {
		font-weight: 600;
		color: var(--color-text);
		min-width: 120px;
	}

	.topbar-center {
		display: flex;
		gap: var(--spacing-xs);
	}

	.topbar-right {
		min-width: 120px;
		text-align: right;
	}

	.role-tab {
		all: unset;
		padding: 6px var(--spacing-md);
		border-radius: var(--radius-sm);
		cursor: pointer;
		font-size: var(--font-size-sm);
		color: var(--color-text-secondary);
		transition: background var(--transition-fast), color var(--transition-fast);
	}

	.role-tab:hover {
		background: var(--color-border-hover);
	}

	.role-tab.active {
		background: var(--color-border);
		color: var(--color-text);
		font-weight: 500;
	}
</style>
