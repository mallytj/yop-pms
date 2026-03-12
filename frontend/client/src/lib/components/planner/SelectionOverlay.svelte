<!-- lib/components/planner/SelectionOverlay.svelte -->
<script lang="ts">
  import { fly } from "svelte/transition";
  import type { DragState } from "$types/planner_data";

  interface Props {
    dragState: DragState;
  }


  let { dragState }: Props = $props();
</script>

<div
  class="cell-selection-overlay"
  transition:fly={{ y: -20, duration: 100 }}
  style="
    grid-row: {dragState.shadowRowIndex} / span 1;
    grid-column: {dragState.shadowStartOffset} / span {dragState.shadowDuration};
  "
></div>

<style>
  .cell-selection-overlay {
    z-index: 50 !important;
    background-color: var(--bg-middle);
    border: var(--border-width-thin) solid var(--border-active);
    border-radius: var(--radius-md);
    position: relative;
    margin: var(--gap-sm) 0;
    z-index: 5;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
</style>
