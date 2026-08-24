<script lang="ts">
	import Block from './Block.svelte';

	export interface MaintenanceCellData {
		id: string;
		blockType: string;
	}

	interface Props {
		block: MaintenanceCellData;
		startIndex: number;
		span: number;
	}

	let { block, startIndex, span }: Props = $props();
	let label = $derived(
		block.blockType
			? block.blockType
					.replace(/_/g, ' ') // snake_case -> spaced words
					.replace(/^./, (char) => char.toUpperCase()) // capitalize first letter
			: 'Maintenance'
	);
</script>

<Block {startIndex} {span} class="maintenance-block" title={label}>
	<strong>{label}</strong>
</Block>

<style>
	:global(.maintenance-block) {
		--stripe-color: var(--color-danger-strong);
		--maintenance-stripe-gradient: repeating-linear-gradient(
			-45deg,
			transparent,
			transparent 4px,
			var(--stripe-color) 4px,
			var(--stripe-color) 5px
		);
		border: 1px solid var(--color-danger-strong);
		background: var(--color-danger-subtle);
		background-image: var(--maintenance-stripe-gradient);
		color: var(--color-text);
	}

	:global(.maintenance-block strong) {
		overflow: hidden;
		text-overflow: ellipsis;
		font-weight: var(--font-weight-bold);
	}
</style>
