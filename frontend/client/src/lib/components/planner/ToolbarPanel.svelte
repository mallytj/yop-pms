<script lang="ts">
  import { Button } from "../ui/";

  // Props
  interface Props {
    multiSelectMode: boolean;
    selectedCellCount: number;
    onToggleMultiSelect: () => void;
    onOpenModal: () => void;
    onCancel: () => void;
  }
  let {
    multiSelectMode,
    selectedCellCount,
    onToggleMultiSelect,
    onOpenModal,
    onCancel,
  }: Props = $props();
</script>

<div class="toolbar">
  {#if !multiSelectMode}
    <Button onclick={onToggleMultiSelect} variant="primary">
      Multi-Select Mode
    </Button>
  {:else}
    <div class="multi-select-active">
      <span>
        {selectedCellCount} area{selectedCellCount !== 1 ? "s" : ""} selected
      </span>
      <Button variant="primary" onclick={onOpenModal}>
        Done ({selectedCellCount})
      </Button>
      <Button variant="secondary" onclick={onCancel}>Cancel</Button>
    </div>
  {/if}
</div>

<style>
  .toolbar {
    padding: var(--padding-card);
    background-color: var(--bg-middle);
    color: var(--text-primary);
    border-bottom: var(--border-width-thin) solid var(--border-base);
    display: flex;
    gap: var(--gap-md);
    overflow: hidden;
    align-items: center;
    position: sticky;
    top: 0;
    z-index: 50;
  }

  .multi-select-active {
    display: flex;
    gap: var(--gap-md);
    align-items: center;
    flex: 1;
  }

  .multi-select-active span {
    font-weight: var(--font-weight-semibold);
    flex: 1;
  }
</style>
