<script lang="ts">
	import type { ReservationCellData } from './RoomAvailabilityCell.svelte';
	import Block from './Block.svelte';

	interface Props {
		reservation: ReservationCellData;
		startIndex: number;
		span: number;
	}

	let { reservation, startIndex, span }: Props = $props();

	let accentVar = $derived(`var(--color-reservation-accent-${reservation.accentIndex ?? 0})`);
	let isHold = $derived(reservation.status === 'on_hold' || reservation.status === 'hold');
	let blockClass = $derived(isHold ? 'reservation-block hold' : 'reservation-block');

	function formatPricePence(pricePence: number): string {
		return new Intl.NumberFormat('en-GB', {
			style: 'currency',
			currency: 'GBP',
			minimumFractionDigits: 0
		}).format(pricePence / 100);
	}

	let price = $derived(
		reservation.pricePence === undefined ? '' : formatPricePence(reservation.pricePence)
	);
</script>

<Block
	{startIndex}
	{span}
	class={blockClass}
	style={`--accent: ${accentVar};`}
	title="{reservation.guestName} · {reservation.code}{price ? ` · ${price}` : ''}"
>
	<strong>{reservation.guestName}</strong>
	<span>{price || reservation.code}</span>
</Block>

<style>
	:global(.reservation-block) {
		flex-direction: column;
		justify-content: center;
		align-items: stretch;
		border: 1px solid var(--accent, var(--color-booking-border));
		background: color-mix(in oklch, var(--accent, var(--color-accent)) 26%, var(--color-surface));
		box-shadow: var(--shadow-sm);
		color: var(--color-text);
	}
	:global(.reservation-block strong),
	:global(.reservation-block span) {
		overflow: hidden;
		text-overflow: ellipsis;
	}
	:global(.reservation-block strong) {
		font-weight: var(--font-weight-bold);
	}
	:global(.reservation-block span) {
		color: var(--color-text-secondary);
	}
	:global(.reservation-block.hold) {
		background: var(--color-warning-subtle);
		border-color: var(--color-warning-strong);
		border-style: dashed;
	}
</style>
