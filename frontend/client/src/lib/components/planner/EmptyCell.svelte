<script lang="ts">
  import { draggable } from "$lib/actions/draggable";

  interface Props {
    dayIndex: number;
    roomIndex: number;
    onDragStart: (dayIndex: number, roomIndex: number, e: PointerEvent) => void;
    onDragMove: (e: PointerEvent, delta: { dx: number; dy: number }) => void;
    onDragEnd: () => void;
  }
  
  let {
    dayIndex,
    roomIndex,
    onDragStart,
    onDragMove,
    onDragEnd,
  }: Props = $props();
</script>

<div
  class="empty-cell"
  use:draggable={{
    onStart: (e: PointerEvent) => onDragStart(dayIndex, roomIndex, e),
    onMove: onDragMove,
    onEnd: onDragEnd,
  }}
  role="button"
  tabindex="0"
  aria-label="Empty cell for creating reservation"
></div>

<style>
  .empty-cell {
    width: 100%;
    height: 100%;
    cursor: crosshair;
    user-select: none;
  }

  .empty-cell:hover {
    background: var(--bg-app);
  }
</style>
