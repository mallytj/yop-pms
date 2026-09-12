<script lang="ts">
	import { DatePicker } from 'bits-ui';
	import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from '@lucide/svelte';
	import { type DateValue, getLocalTimeZone, parseDate, today } from '@internationalized/date';
	import { addDays } from '$helpers/dates.js';
	import { DEFAULT_LOOKBACK_DAYS, TAPE_CHART_WINDOW_DAYS } from '$lib/api/tape-chart.js';

	interface Props {
		from: string;
		onRangeChange: (from: string, to: string) => void;
		/** Fired on Today click, in addition to onRangeChange, so a parent can re-scroll an already-loaded grid back to today's column. */
		onToday?: () => void;
	}

	let { from, onRangeChange, onToday }: Props = $props();

	function changeStart(newFrom: string) {
		if (!newFrom) return;
		onRangeChange(newFrom, addDays(newFrom, TAPE_CHART_WINDOW_DAYS));
	}

	function jumpToToday() {
		const todayKey = today(getLocalTimeZone()).toString();
		changeStart(addDays(todayKey, -DEFAULT_LOOKBACK_DAYS));
		onToday?.();
	}

	let startValue = $derived<DateValue | undefined>(from ? parseDate(from) : undefined);
</script>

<div class="date-picker" role="group" aria-label="Tape Chart dates">
	<DatePicker.Root
		value={startValue}
		onValueChange={(value) => {
			if (value) changeStart(value.toString());
		}}
		granularity="day"
		weekdayFormat="short"
		numberOfMonths={2}
		pagedNavigation
	>
		<div class="date-field">
			<DatePicker.Label class="date-label">Start</DatePicker.Label>
			<DatePicker.Input aria-label="Tape Chart start date" class="date-input">
				{#snippet children({ segments })}
					{#each segments as { part, value }, index (index)}
						<DatePicker.Segment {part} class="segment">{value}</DatePicker.Segment>
					{/each}
				{/snippet}
			</DatePicker.Input>
			<DatePicker.Trigger class="trigger" aria-label="Open calendar">
				<CalendarIcon size={14} />
			</DatePicker.Trigger>
		</div>
		<DatePicker.Content class="content" sideOffset={6}>
			<DatePicker.Calendar class="calendar">
				{#snippet children({ months, weekdays })}
					<div class="calendar-months">
						{#each months as month (month.value.toString())}
							<div class="calendar-month">
								<DatePicker.Header class="calendar-header">
									<DatePicker.PrevButton class="nav-btn">
										<ChevronLeft size={16} />
									</DatePicker.PrevButton>
									<div class="heading-controls">
										<DatePicker.MonthSelect class="month-select" monthFormat="long" />
										<DatePicker.YearSelect class="year-select" />
									</div>
									<DatePicker.NextButton class="nav-btn">
										<ChevronRight size={16} />
									</DatePicker.NextButton>
								</DatePicker.Header>
								<DatePicker.Grid class="grid">
									<DatePicker.GridHead>
										<DatePicker.GridRow class="grid-row">
											{#each weekdays as weekday (weekday)}
												<DatePicker.HeadCell class="head-cell"
													>{weekday.slice(0, 2)}</DatePicker.HeadCell
												>
											{/each}
										</DatePicker.GridRow>
									</DatePicker.GridHead>
									<DatePicker.GridBody>
										{#each month.weeks as weekDates, weekIndex (weekIndex)}
											<DatePicker.GridRow class="grid-row">
												{#each weekDates as date (date.toString())}
													<DatePicker.Cell {date} month={month.value} class="cell">
														<DatePicker.Day class="day">{date.day}</DatePicker.Day>
													</DatePicker.Cell>
												{/each}
											</DatePicker.GridRow>
										{/each}
									</DatePicker.GridBody>
								</DatePicker.Grid>
							</div>
						{/each}
					</div>
				{/snippet}
			</DatePicker.Calendar>
		</DatePicker.Content>
	</DatePicker.Root>
	<div class="presets" aria-label="Shift date range">
		<button class="today" type="button" onclick={jumpToToday}>Today</button>
	</div>
</div>

<style>
	.date-picker {
		display: flex;
		align-items: center;
		gap: var(--spacing-md);
		min-width: 0;
	}

	.date-field {
		display: inline-flex;
		align-items: center;
		gap: var(--spacing-xs);
		height: 28px;
		padding: 0 var(--spacing-md);
		border: 1px solid var(--color-border-strong);
		border-radius: 4px;
		background: var(--color-surface);
		color: var(--color-text);
	}

	.date-field :global(.date-label) {
		font-size: var(--font-size-2xs);
		padding: var(--spacing-control);
		color: var(--color-text-secondary);
	}

	.date-field :global(.date-input) {
		display: inline-flex;
		align-items: center;
		font-size: var(--font-size-xs);
	}

	.date-field :global(.segment) {
		padding: 0 1px;
	}

	.date-field :global(.segment[data-placeholder]) {
		color: var(--color-text-secondary);
	}

	.date-field :global(.trigger) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		color: var(--color-text-secondary);
		background: none;
		border: none;
		height: auto;
		min-width: 0;
		padding: 0;
		cursor: pointer;
	}

	.date-field :global(.trigger:hover) {
		color: var(--color-text);
	}

	button {
		font: inherit;
	}

	:global(.content) {
		z-index: 200;
	}

	:global(.calendar) {
		padding: var(--spacing-sm);
		background: var(--color-surface);
		border: 1px solid var(--color-border-strong);
		border-radius: 6px;
		box-shadow: var(--shadow-md);
	}

	:global(.calendar-months) {
		display: flex;
		gap: var(--spacing-md);
	}

	:global(.calendar-header) {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--spacing-xs);
		margin-bottom: var(--spacing-xs);
	}

	:global(.calendar .heading-controls) {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	:global(.calendar .month-select),
	:global(.calendar .year-select) {
		font: inherit;
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text);
		background: none;
		border: 1px solid transparent;
		border-radius: 4px;
		padding: 2px 4px;
		cursor: pointer;
	}

	:global(.calendar .month-select:hover),
	:global(.calendar .year-select:hover) {
		border-color: var(--color-border-strong);
		background: var(--color-surface-overlay);
	}

	:global(.calendar .nav-btn) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		height: 24px;
		width: 24px;
		min-width: 0;
		padding: 0;
		background: none;
		border: none;
		color: var(--color-text-secondary);
		cursor: pointer;
	}

	:global(.calendar .nav-btn:hover) {
		color: var(--color-text);
	}

	:global(.calendar .grid) {
		border-collapse: collapse;
	}

	:global(.calendar .head-cell) {
		font-size: var(--font-size-3xs);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-secondary);
		padding: 2px 4px;
	}

	:global(.calendar .day) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		height: 26px;
		width: 26px;
		border-radius: 4px;
		font-size: var(--font-size-xs);
		color: var(--color-text);
		cursor: pointer;
	}

	:global(.calendar .day:hover) {
		background: var(--color-surface-overlay);
	}

	:global(.calendar .day[data-selected]) {
		background: var(--color-accent-strong);
		color: var(--color-accent-subtle);
	}

	:global(.calendar .day[data-disabled]) {
		opacity: 0.35;
		cursor: not-allowed;
	}

	:global(.calendar .day[data-outside-month]) {
		opacity: 0.4;
	}

	.presets {
		display: inline-flex;
		gap: 4px;
	}

	button {
		height: 28px;
		min-width: 32px;
		padding: 0 6px;
		border: 1px solid var(--color-border-strong);
		border-radius: 4px;
		background: var(--color-surface-raised-strong);
		color: var(--color-text-secondary);
		font-size: var(--font-size-2xs);
		cursor: pointer;
	}

	button:hover:not(:disabled) {
		background: var(--color-surface-overlay);
	}

	button:disabled {
		cursor: not-allowed;
		opacity: 0.45;
	}

	button.today {
		color: var(--color-accent-strong);
		background: var(--color-accent-subtle);
		border-color: var(--color-accent-border);
	}

	:global(.date-picker button:focus-visible) {
		outline: 2px solid var(--color-accent-ring);
		outline-offset: 1px;
	}

	@media (max-width: 900px) {
		.date-picker {
			flex-wrap: wrap;
		}
	}
</style>
