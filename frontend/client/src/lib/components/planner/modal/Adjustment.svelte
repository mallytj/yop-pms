<!-- <script lang="ts">
  import { Button } from "$lib/components/ui";
  import Input from "$lib/components/ui/Input.svelte";
  import { formatPrice } from "$helpers/reservation_modal";
  import { onMount } from "svelte";

  export let title: string;
  export let base = 0;
  export let onUpdate: (newValue: number) => void;
  export let code: string;
  type AdjustmentMode = "fixed" | "percentage" | "override";

  let mode: AdjustmentMode = "fixed";
  let val: number = 0;
  let reason: string = "";
  let preview: number = base;
  let popoverEl: HTMLDivElement;

  $: preview = (() => {
    const valPounds = val * 100;
    switch (mode as AdjustmentMode) {
      case "fixed":
        return base + valPounds;
      case "percentage":
        return base + (base * val) / 100;
      case "override":
        return valPounds;
    }
  })();

  function close() {
    popoverEl?.hidePopover();
  }

  const modes = [
    { key: "fixed" as AdjustmentMode, label: "± £" },
    { key: "percentage" as AdjustmentMode, label: "± %" },
    { key: "override" as AdjustmentMode, label: "£" },
  ];
</script>

<div
  class="adjustment-popover"
  popover="auto"
  id={`popover-${code}`}
  bind:this={popoverEl}
  style={`position-anchor: --rate-plan-${code}`}
>
  <div class="header">
    <h4 class="popover-title">{title}</h4>
    <button class="close-btn" on:click={close}> &times; </button>
  </div>

  <div class="mode-tabs">
    {#each modes as m}
      <button
        class="mode-tab"
        class:active={m.key === mode}
        on:click={() => {
          mode = m.key;
          val = 0;
          reason = "";
        }}
      >
        {m.label}
      </button>
    {/each}
  </div>

  <div class="input-wrapper">
    <Input
      value={val === 0 ? undefined : val}
      placeholder={mode === "fixed" || "percentage" ? "-20" : String(base)}
      inputType="number"
      prefix={mode === "percentage" ? "%" : "£"}
      on:input={(e) => {
        const target = e.target as HTMLInputElement;
        val = Number(target.value);
      }}
    />

    <Input value={reason} inputType="text" placeholder="Reason" />
    {#if val !== 0}
      <div class="preview">
        <span class="preview-base">
          {formatPrice(base)}
          {#if mode !== "override"}
            <span class="preview-delta" class:negative={preview < base}>
              {mode === "percentage" ? `${val}%` : `${formatPrice(val * 100)}`}
            </span>
          {/if}
        </span>
        <span class="preview-final"> {formatPrice(preview)}</span>
      </div>
    {/if}

    <div class="actions">
      <Button variant="secondary" onClick={close} size="sm">Cancel</Button>
      <Button variant="primary" onClick={() => onUpdate(preview)} size="sm">
        Apply
      </Button>
    </div>
  </div>
</div>

<style>
  .adjustment-popover {
    z-index: 60;
    width: 250px;
    position: fixed;
    margin: 0;
    inset: auto;
    border: none;

    top: calc(anchor(bottom) + var(--gap-lg));
    justify-self: anchor-center;
    z-index: 100;
    position-try-fallbacks: --aside, --above;
    background: var(--bg-selected);
    border-radius: var(--radius-md);
    padding: var(--padding-card);
    box-shadow: var(--shadow-lg);
    border: var(--border-width-medium) solid var(--border-active);
  }

  @position-try --aside {
    left: calc(anchor(right) + var(--gap-sm));
    right: auto;
    top: auto;
    bottom: calc(anchor(bottom) - var(--gap-lg));
    align-self: anchor-center;
  }

  @position-try --above {
    top: auto;
    bottom: calc(anchor(top) + var(--gap-sm));
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--gap-sm);
    text-transform: uppercase;
    letter-spacing: var(--letter-spacing-widest);
  }

  .popover-title {
    font-weight: var(--font-weight-semibold);
    font-size: var(--font-size-xs);
    color: var(--fg-subtle);
  }

  .close-btn {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--fg-subtle);
    font-size: var(--font-size-lg);
    line-height: var(--line-height-none);
    padding: 0;
  }

  .mode-tabs {
    display: flex;
    gap: var(--gap-xs);
    margin-bottom: var(--gap-sm);
  }

  .mode-tab {
    flex: 1;
    padding: var(--padding-btn-sm);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-semibold);
    border-radius: var(--radius-md);
    cursor: pointer;
    border: var(--border-width-thin) solid var(--border-base);
    background: var(--btn-secondary-bg);
    color: var(--btn-secondary-fg);
    transition: var(--transition-fast);
  }

  .mode-tab.active {
    border-color: var(--btn-primary-bg);
    background: var(--btn-primary-bg);
    color: var(--btn-primary-fg);
  }

  .input-wrapper {
    display: flex;
    flex-direction: column;
    gap: var(--gap-sm);
  }

  .preview {
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: var(--bg-subtle);
    border-radius: var(--radius-md);
    padding: var(--padding-btn-sm);
    font-size: var(--font-size-xs);
    font-family: var(--font-data);
  }

  .preview-base {
    color: var(--fg-muted);
    display: flex;
    flex-direction: column;
  }

  .preview-delta {
    color: var(--color-success-bg);
  }
  .preview-delta::before {
    content: "+";
    margin-right: var(--gap-xs);
    color: var(--color-success-bg);
  }

  .preview-delta.negative {
    color: var(--color-danger-bg);
  }
  .preview-delta.negative::before {
    content: "-";
    color: var(--color-danger-bg);
  }

  .preview-final {
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
    margin-bottom: auto;
  }

  .preview-final::before {
    content: "=>";
    color: var(--fg-muted);
    margin-right: var(--gap-xs);
  }

  .actions {
    display: flex;
    gap: var(--gap-sm);
    justify-content: space-between;
  }
</style> -->
