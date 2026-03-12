<script lang="ts">
  import type { Adjustment } from "$lib/types/booking_engine";
  import {Input} from "$components/ui";

  interface Props {
    placeholder?: string;
    label?: string;
    adjustment: Adjustment;
    currencySymbol?: string;
  }

  let {
    label,
    adjustment = $bindable({
      value: 0,
      type: "fixed_amount",
      reason: "",
    }),
    currencySymbol = "£",
  }: Props = $props();

  // Derived state for the Input component configuration
  const isPercent = $derived(adjustment.type === "percentage");
  const prefix = $derived(isPercent ? "" : currencySymbol);
  const suffix = $derived(isPercent ? "%" : "");
  const dp = $derived(isPercent ? 0 : 2);
</script>

<div class="adjustment-container">
  {#if label}
    <label class="field-label" for="adjustment-input">{label}</label>
  {/if}

  <div class="control-group">
    <div class="input-wrapper">
      <Input
        inputType="number"
        placeholder={adjustment.type === "fixed_amount" ? "0.00" : "0"}
        {prefix}
        {suffix}
        {dp}
        onblur={(e) => {
          const value = (e.target as HTMLInputElement).value;
          adjustment.value = parseFloat(value);
        }}
        bind:value={adjustment.value!}
      />
    </div>

    <fieldset class="segmented-control">
      <legend class="sr-only">Adjustment Type</legend>

      <label class="segment" class:active={adjustment.type === "fixed_amount"}>
        <input
          type="radio"
          bind:group={adjustment.type}
          value="fixed_amount"
          oninput={() => {
            adjustment.value = null;
          }}
        />
        <span>{currencySymbol}</span>
      </label>

      <label class="segment" class:active={adjustment.type === "percentage"}>
        <input
          type="radio"
          bind:group={adjustment.type}
          value="percentage"
          oninput={() => {
            adjustment.value = null;
          }}
        />
        <span>%</span>
      </label>
    </fieldset>
  </div>
</div>

<style>
  .adjustment-container {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs, 0.5rem);
    width: 100%;
  }

  .field-label {
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-bold);
    color: var(--fg-subtle);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .control-group {
    display: flex;
    align-items: stretch;
    gap: var(--spacing-xs, 4px);
    background-color: var(--bg-panel);
    padding: var(--spacing-xs, 4px);
    border-radius: var(--radius-md);
    border: var(--border-width-thin) solid var(--border-base);
  }

  /* Target the Input.svelte inner group to remove its border 
     since the container now handles the 'box' feel */
  .control-group :global(.input-group) {
    border: none;
    background: transparent;
  }

  .input-wrapper {
    flex: 1;
  }

  .segmented-control {
    display: flex;
    border: none;
    padding: 0;
    margin: 0;
    background: var(--bg-subtle);
    border-radius: var(--radius-sm);
    overflow: hidden;
  }

  .segment {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0 var(--spacing-md, 12px);
    cursor: pointer;
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-semibold);
    color: var(--fg-subtle);
    transition: all var(--transition-fast);
    user-select: none;
  }

  /* Hide the radio button but keep it accessible for screen readers */
  .segment input {
    position: absolute;
    opacity: 0;
    width: 0;
    height: 0;
  }

  .segment:hover {
    color: var(--text-primary);
  }

  .segment.active {
    background-color: var(--bg-selected);
    color: var(--text-active);
    box-shadow: var(--shadow-sm);
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    border: 0;
  }
</style>
