<!-- lib/components/planner/SavedSelections.svelte -->
<script lang="ts">
  import type { CellSelection } from "$types/planner_data";
  interface Props {
    selectedCells?: CellSelection[];
    onRemoveSelection: (index: number) => void;
  }
  
  let { selectedCells = [], onRemoveSelection }: Props = $props();
</script>

{#each selectedCells as selection, idx (idx)}
  <div
    class="cell-selection-shadow"
    style="
      grid-row: {selection.startRoom + 2} / span {selection.endRoom -
      selection.startRoom +
      1};
      grid-column: {selection.startDay + 2} / span {selection.endDay -
      selection.startDay +
      1};
    "
  >
    <button
      class="remove-selection"
      onclick={() => onRemoveSelection(idx)}
      title="Remove selection"
    >
      ✕
    </button>
  </div>
{/each}

<style>
  .cell-selection-shadow {
    margin: var(--gap-sm) 0;
    position: relative;
    z-index: 5;
    border-radius: var(--radius-md);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
    z-index: 5;
    background: rgba(from var(--color-success) r g b / 0.2);
    border: var(--border-width-thin) solid var(--color-success);
    pointer-events: auto;
    box-shadow: var(--shadow-xs);
  }

  .remove-selection {
    position: absolute;
    top: var(--gap-sm);
    right: var(--gap-sm);
    width: 20px;
    height: 20px;
    background: var(--color-success);
    color: var(--color-light);
    border: none;
    border-radius: var(--radius-full);
    cursor: pointer;
    font-size: var(--font-size-xs);
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10;
  }

  .remove-selection:hover {
    background: rgb(from var(--color-success) r g b / 0.8);
  }
</style>
