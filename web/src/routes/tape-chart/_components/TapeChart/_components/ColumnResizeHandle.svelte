<script lang="ts">
	interface Props {
		width: number;
		onResize: (width: number) => void;
		minWidth?: number;
		maxWidth?: number;
		label: string;
	}

	let { width, onResize, minWidth = 100, maxWidth = 320, label }: Props = $props();

	let dragStartX = 0;
	let dragStartWidth = 0;

	function onPointerMove(event: PointerEvent) {
		const newWidth = dragStartWidth + (event.clientX - dragStartX);
		const clamp = (max: number, min: number, val: number) => Math.min(max, Math.max(min, val));

		const nextWidth = clamp(maxWidth, minWidth, newWidth);
		onResize(nextWidth);
	}

	function onPointerUp() {
		window.removeEventListener('pointermove', onPointerMove);
		window.removeEventListener('pointerup', onPointerUp);
	}

	function onPointerDown(event: PointerEvent) {
		dragStartX = event.clientX;
		dragStartWidth = width;

		window.addEventListener('pointermove', onPointerMove);
		window.addEventListener('pointerup', onPointerUp);
	}
</script>

<!-- role="separator" is keyboard-focusable by spec here (it drives pointer-based resize, not a click/key action), so Svelte's non-interactive-tabindex rule doesn't apply. -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
	class="column-resize-handle"
	role="separator"
	aria-orientation="vertical"
	aria-label={label}
	tabindex="0"
	onpointerdown={onPointerDown}
></div>

<style>
	.column-resize-handle {
		position: absolute;
		top: 0;
		right: 0;
		width: 6px;
		height: 100%;
		cursor: col-resize;
		touch-action: none;
		z-index: 1;
	}
	.column-resize-handle:hover,
	.column-resize-handle:focus-visible {
		background: var(--color-accent);
		outline: none;
	}
</style>
