<script lang="ts">
	import type { Snippet } from 'svelte';

	// Every block's left/right edge stays this many pixels inside its own
	// [startIndex, startIndex+span) date span. Letting either edge bleed past
	// the span makes an unrelated block in the next room-availability column
	// (e.g. a maintenance block starting the day a reservation checks out)
	// read as an overlap that isn't real.
	const GUTTER_PX = 4;

	interface Props {
		startIndex: number;
		span: number;
		class?: string;
		style?: string;
		title?: string;
		children: Snippet;
	}

	let { startIndex, span, class: className = '', style = '', title, children }: Props = $props();
</script>

<div
	class="tape-block {className}"
	style="left: calc(var(--room-w) + {startIndex} * var(--cell-w) + {GUTTER_PX}px); width: calc({span} * var(--cell-w) - {GUTTER_PX *
		2}px); {style}"
	role="button"
	{title}
>
	{@render children()}
</div>

<style>
	.tape-block {
		position: absolute;
		top: 3px;
		z-index: 3;
		box-sizing: border-box;
		height: calc(var(--row-h) - 6px);
		display: flex;
		padding: 0 var(--spacing-sm);
		border-radius: var(--radius-sm);
		font-size: var(--font-size-3xs);
		white-space: nowrap;
		overflow: hidden;
		cursor: pointer;
	}
</style>
