<script lang="ts">
  import { Button } from "$lib/components/ui";
  // import {Adjustment} from "./";
  import { formatPrice } from "$lib/helpers/reservation_modal";

  interface Props {
    name: string;
    code: string;
    basePrice: number;
    finalPrice: number;
    description: string;
    selected: boolean;
    onClick: () => void;
  }

  let {
    name,
    code,
    basePrice,
    finalPrice = $bindable(),
    description,
    selected,
    onClick,
  }: Props = $props();

  let adjustment = $derived(finalPrice - basePrice);
  let isPositiveAdjustment = $derived(adjustment >= 0);

  function handleAdjustmentUpdate(newValue: number) {
    finalPrice = newValue;
  }
</script>

<div
  class="rate-plan"
  title={name}
  style="anchor-name: --rate-plan-{code};"
  data-selected={selected}
  onclick={onClick}
  role="button"
  tabindex="0"
  aria-label={name}
  onkeydown={(e) => {
    if (e.key === "Enter") {
      onClick();
    }
  }}
>
  <span class="selected-icon">&check;</span>
  <h5 class="rate-code">{code}</h5>
  <p class="rate-name">{name}</p>

  <div class="rate-prices">
    <div class="price-row">
      <span class="price-label">Base</span>
      <span class="price-value base">{formatPrice(basePrice)}</span>
    </div>

    <div class="price-row">
      <span class="price-label">Adjustment</span>
      <span
        class="price-value adjustment"
        class:positive={isPositiveAdjustment && adjustment !== 0}
        class:negative={!isPositiveAdjustment}
      >
        {isPositiveAdjustment
          ? adjustment === 0
            ? ""
            : "+"
          : "-"}{formatPrice(adjustment)}
      </span>
    </div>
    <div class="price-row price-row--final">
      <span class="price-label">Total</span>
      <span class="price-value total">
        {formatPrice(finalPrice)}
      </span>
    </div>
    <Button
      variant="secondary"
      dotted
      disabled={!selected}
      popoverTarget={`popover-${code}`}
    >
      &ShortDownArrow; Adjust</Button
    >
  </div>
</div>

<!-- <Adjustment
  title="Adjustment"
  base={basePrice}
  onUpdate={handleAdjustmentUpdate}
  {code}
/> -->

<style>
  .rate-plan {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--gap-sm);
    border: var(--border-width-medium) solid var(--border-base);
    border-radius: var(--radius-md);
    padding: var(--spacing-4);
    text-align: left;
    background-color: var(--bg-middle);
    cursor: pointer;
    transition: var(--transition-slow);
    box-shadow: var(--shadow-sm);
  }

  .rate-plan:hover {
    background-color: var(--bg-selected);
    box-shadow: var(--shadow-md);
  }

  .rate-plan[data-selected="true"] {
    background-color: var(--bg-selected);
    border-color: var(--border-active);
    box-shadow: var(--shadow-md);

    & .selected-icon {
      opacity: 1;
    }
  }

  .selected-icon {
    opacity: 0;
    position: absolute;
    z-index: 1;
    top: var(--gap-sm);
    right: var(--gap-sm);
    color: var(--border-active);
    font-size: var(--font-size-lg);
    font-weight: var(--font-weight-semibold);
    padding: var(--gap-xs);
    line-height: 1;
    transition: var(--transition-slow);
  }

  .rate-code {
    font-size: var(--font-size-md);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    line-height: var(--line-height-none);
  }

  .rate-name {
    color: var(--text-secondary);
    font-size: var(--font-size-xs);
    line-height: var(--line-height-none);
    border-bottom: 1px solid var(--border-base);
    padding-bottom: var(--gap-sm);
  }

  .price-value.adjustment.positive {
    color: var(--color-success-bg);
  }
  .price-value.adjustment.negative {
    color: var(--color-danger-bg);
  }

  .price-row {
    font-size: var(--font-size-xs);
    font-family: var(--font-data);
    font-weight: var(--font-weight-normal);
    color: var(--text-secondary);
    display: flex;
    justify-content: space-between;
  }

  .rate-prices {
    display: flex;
    flex-direction: column;
    gap: var(--gap-xs);
    padding-top: var(--gap-sm);
  }

  .price-row--final {
    color: var(--text-primary);
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-semibold);
  }
</style>
