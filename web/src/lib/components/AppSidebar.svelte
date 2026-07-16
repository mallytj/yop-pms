<script lang="ts">
	import { page } from '$app/stores';
	import { ChartNoAxesCombined, Sparkles, FileText, Building2, CalendarDays } from 'lucide-svelte';

	const navItems = [
		{ label: 'Tape Chart', icon: ChartNoAxesCombined, href: '/planner/reservations' },
		{ label: 'Housekeeping', icon: Sparkles, href: '/housekeeping' },
		{ label: 'Reports', icon: FileText, href: null },
		{ label: 'Banking', icon: Building2, href: null },
		{ label: 'Availability', icon: CalendarDays, href: null }
	];

	const path = $derived($page.url.pathname);
</script>

<aside>
	<nav>
		{#each navItems as item}
			{@const Icon = item.icon}
			{#if item.href}
				<a href={item.href} class="nav-item" class:active={path.startsWith(item.href)}>
					<Icon size={18} strokeWidth={1.5} />
					{item.label}
				</a>
			{:else}
				<button class="nav-item">
					<Icon size={18} strokeWidth={1.5} />
					{item.label}
				</button>
			{/if}
		{/each}
	</nav>
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

	.nav-item:hover {
		background: var(--color-border-hover);
	}

	.nav-item.active {
		background: var(--color-border);
		color: var(--color-text);
		font-weight: 500;
	}
</style>
