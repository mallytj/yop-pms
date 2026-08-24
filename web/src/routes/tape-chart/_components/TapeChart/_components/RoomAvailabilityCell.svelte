<script lang="ts">
	import { isMonthStart, isWeekend } from '$helpers/dates.js';

	export interface ReservationCellData {
		code: string;
		guestName: string;
		pricePence?: number;
		status?: string;
		accentIndex?: number;
	}

	interface Props {
		roomName: string;
		date: string;
		status: string;
		reservation?: ReservationCellData;
		today: string;
	}

	let { roomName, date, status, reservation, today }: Props = $props();
	// Converts pence to whole pounds and renders as GBP currency (e.g. 12345 -> "£123"),
	// rounded to the nearest pound with no decimal places.
	const priceFormatter = new Intl.NumberFormat('en-GB', {
		style: 'currency',
		currency: 'GBP',
		minimumFractionDigits: 0
	});
	let reservationPrice = $derived(
		reservation?.pricePence === undefined ? '' : priceFormatter.format(reservation.pricePence / 100)
	);
	let isHold = $derived(status === 'on_hold' || status === 'hold');
</script>

<div
	class="cell"
	class:hold={isHold}
	class:today={date === today}
	class:weekend={isWeekend(date)}
	class:month-start={isMonthStart(date)}
	role="gridcell"
	aria-label="{roomName} · {date} · {reservation?.code || status || 'available'}{reservationPrice
		? ` · ${reservationPrice}`
		: ''}"
	title="{roomName} · {date} · {reservation?.code || status || 'available'}{reservationPrice
		? ` · ${reservationPrice}`
		: ''}"
></div>

<style>
	.cell {
		flex: 0 0 var(--cell-w);
		width: var(--cell-w);
		min-width: var(--cell-w);
		max-width: var(--cell-w);
		height: var(--row-h);
		border-right: 1px solid var(--color-border-subtle);
		border-bottom: 1px solid var(--color-border-subtle);
		background: var(--color-surface-raised);
	}
	.cell.weekend {
		background: var(--color-highlight-weekend);
	}
	.cell.today {
		background: var(--color-highlight-today);
	}
	.cell.month-start {
		box-shadow: inset 2px 0 0 0 var(--color-border-strong);
	}
</style>
