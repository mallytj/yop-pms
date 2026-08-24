<script lang="ts">
	import { dayNumber, isWeekend, monthLabel, weekdayLabel } from '$helpers/dates.js';

	interface Props {
		date: string;
		today: string;
	}

	let { date, today }: Props = $props();
	let isCurrentDay = $derived(date === today);
</script>

<div
	class="header-cell"
	class:weekend={isWeekend(date)}
	class:today={isCurrentDay}
	role="columnheader"
	title={date}
>
	<span class="date-month" class:show={true}>{monthLabel(date)}</span>
	<span class="date-day">{dayNumber(date)}</span>
	<span class="date-wd">{weekdayLabel(date)}</span>
</div>

<style>
	.header-cell {
		/* Shared by .date-month and .date-wd, the two secondary date labels. */
		--header-cell-meta-font-size: var(--font-size-4xs);
		position: relative;
		flex: 0 0 var(--cell-w);
		width: var(--cell-w);
		min-width: var(--cell-w);
		max-width: var(--cell-w);
		height: var(--header-h);
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--spacing-xs);
		padding: var(--spacing-xs) 0 var(--spacing-sm);
		background: var(--color-surface-raised);
		border-right: 1px solid var(--color-border);
		border-bottom: 1px solid var(--color-border);
	}

	.header-cell.weekend {
		background: var(--color-highlight-weekend);
	}
	.header-cell.today {
		background: var(--color-highlight-today);
	}
	.header-cell.today::after {
		content: '';
		position: absolute;
		top: 0;
		left: 4px;
		right: 4px;
		height: var(--spacing-xs);
		border-radius: 2px;
		background: var(--color-accent);
	}

	.date-month {
		font-size: var(--header-cell-meta-font-size);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-muted);
		line-height: 1;
		text-transform: uppercase;
		text-wrap: balance;
		opacity: 0;
	}
	.date-month.show {
		opacity: 1;
	}
	.date-day {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-bold);
		color: var(--color-text);
		line-height: 1.1;
		font-variant-numeric: tabular-nums;
	}
	.header-cell.today .date-day {
		color: var(--color-accent-strong);
	}
	.date-wd {
		font-size: var(--header-cell-meta-font-size);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-muted);
		line-height: 1;
		text-transform: uppercase;
	}
</style>
