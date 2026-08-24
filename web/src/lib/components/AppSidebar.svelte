<script lang="ts">
	import { page } from '$app/state';
	import {
		ChartNoAxesCombined,
		Sparkles,
		FileText,
		Building2,
		CalendarDays,
		PanelLeftClose
	} from '@lucide/svelte';

	let collapsed = $state(false);

	const navItems = [
		{ label: 'Tape Chart', icon: ChartNoAxesCombined, href: '/tape-chart' },
		{ label: 'Housekeeping', icon: Sparkles, href: '/housekeeping' },
		{ label: 'Reports', icon: FileText, href: null },
		{ label: 'Banking', icon: Building2, href: null },
		{ label: 'Availability', icon: CalendarDays, href: null }
	];

	const path = $derived(page.url.pathname);
</script>

<aside data-collapsed={collapsed}>
	<nav>
		{#each navItems as item}
			{@const Icon = item.icon}
			{#if item.href}
				<a
					href={item.href}
					class="nav-item"
					class:active={path.startsWith(item.href)}
					title={collapsed ? item.label : undefined}
				>
					<Icon size={18} strokeWidth={1.5} />
					<span aria-hidden={collapsed}>{item.label}</span>
				</a>
			{:else}
				<button
					class="nav-item"
					disabled
					aria-disabled="true"
					title={collapsed ? item.label : undefined}
				>
					<Icon size={18} strokeWidth={1.5} />
					<span aria-hidden={collapsed}>{item.label}</span>
				</button>
			{/if}
		{/each}
	</nav>

	<button
		class="collapse-toggle"
		aria-expanded={!collapsed}
		aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
		onclick={() => (collapsed = !collapsed)}
	>
		<PanelLeftClose size={18} strokeWidth={1.5} />
	</button>
</aside>

<style>
	aside {
		width: var(--sidebar-width);
		flex-shrink: 0;
		border-right: 1px solid var(--color-border);
		padding: var(--spacing-lg);
		background: var(--color-bg);
		color: var(--color-text-secondary);
		font-size: var(--font-size-sm);
		line-height: 1.4;
		display: flex;
		flex-direction: column;
		gap: var(--spacing-md);
		box-sizing: border-box;
		transition:
			width 0.2s ease,
			padding-inline 0.2s ease;
	}

	aside[data-collapsed='true'] {
		width: var(--sidebar-width-collapsed, 44px);
		padding-inline: 6px;
	}

	.nav-item span {
		overflow: hidden;
		white-space: nowrap;
		max-width: 160px;
		opacity: 1;
		transition:
			max-width 0.2s ease,
			opacity 0.15s ease;
	}

	aside[data-collapsed='true'] .nav-item span {
		max-width: 0;
		opacity: 0;
	}

	aside[data-collapsed='true'] .nav-item,
	aside[data-collapsed='true'] .collapse-toggle {
		gap: 0;
		padding-inline: 6px;
	}

	.nav-item :global(svg),
	.collapse-toggle :global(svg) {
		flex-shrink: 0;
	}

	nav {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.nav-item {
		all: unset;
		display: flex;
		align-items: center;
		gap: var(--spacing-md);
		padding: 6px var(--spacing-md);
		border-radius: var(--radius-sm);
		cursor: pointer;
		text-decoration: none;
		color: inherit;
		transition: background var(--transition-fast);
	}

	.nav-item:hover:not(:disabled) {
		background: var(--color-border-hover);
	}

	.nav-item:disabled {
		cursor: not-allowed;
		opacity: 0.55;
	}

	.nav-item.active {
		background: var(--color-border);
		color: var(--color-text);
		font-weight: var(--font-weight-medium);
	}

	.collapse-toggle {
		all: unset;
		display: flex;
		align-items: center;
		gap: var(--spacing-md);
		padding: 6px var(--spacing-md);
		border-radius: var(--radius-sm);
		cursor: pointer;
		color: inherit;
		margin-top: auto;
		transition: background var(--transition-fast);
	}

	.collapse-toggle:hover {
		background: var(--color-border-hover);
	}
</style>
