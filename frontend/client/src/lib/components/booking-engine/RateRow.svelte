<script lang="ts">
  import {
    applyAdjustment,
    getDatesInRange,
    getTotalRate,
    strNumToCurrencyStr,
  } from "$helpers/booking-engine";
  import type { Adjustment, RatePlan, UUID } from "$types/booking_engine";
  import { formatPrice } from "$helpers/reservation_modal";
  import CurrencyAdjustment from "./CurrencyAdjustment.svelte";
  import { useBookingStore } from "$stores/booking.svelte";
  import Input from "../ui/Input.svelte";

  interface Props {
    ratePlan: RatePlan;
    selected: boolean;
    selectedRateId: UUID | null | undefined;
    roomTypeId: UUID;
    groupId: string;
  }

  let {
    ratePlan,
    selected,
    selectedRateId = $bindable(),
    roomTypeId,
    groupId,
  }: Props = $props();

  const store = useBookingStore();

  const dates = getDatesInRange(
    store.draft.globalCheckInDate,
    store.draft.globalCheckOutDate,
  );

  const totalRates = getTotalRate(
    roomTypeId,
    ratePlan.id,
    store.rateMap,
    dates,
  );

  let adjustment = $state<Adjustment>({
    value: null,
    type: "fixed_amount",
    reason: "",
  });

  let final = $derived(applyAdjustment(adjustment, totalRates));
  let delta = $derived(final - totalRates);

  function onBlur(e: FocusEvent) {
    const target = e.target as HTMLInputElement;
    let value: string | null = target.value;

    if (!Number(value)) {
      value = "";
    } else {
      value = strNumToCurrencyStr(value);
    }
  }
</script>

<label class="rate-row" data-selected={selected}>
  <input
    type="radio"
    name={groupId}
    id="{groupId}-{ratePlan.id}"
    value={ratePlan.id}
    bind:group={selectedRateId}
    checked={selected}
  />

  <div class="rate-info">
    <span class="code">{ratePlan.code}</span>
    <span class="name">{ratePlan.name}</span>
  </div>

  <div class="pricing">
    <div class="pricing-column pricing--base">
      <span class="pricing-label">Base</span>
      <span class="pricing-value">{formatPrice(totalRates)}</span>
    </div>
    <div class="pricing-column pricing--adjustment">
      <span class="pricing-label">Adjustment</span>
      <span class="pricing-value">
        <CurrencyAdjustment bind:adjustment />
      </span>
    </div>
    <div class="pricing-column pricing--final">
      <span class="pricing-label">Total</span>
      <span class="pricing-value">
        <Input
          inputType="number"
          onblur={onBlur}
          value={final / 100}
          prefix="£"
          suffix=""
          dp={2}
          fontSize="var(--font-size-md)"
        />
        <span class="delta" class:negative={delta < 0}>
          {formatPrice(delta)}
        </span>
      </span>
    </div>
  </div>
</label>
{#if delta > 0}
  <div class="reason-row">
    <label for="reason">REASON FOR ADJUSTMENT:</label>
    <select name="reason" id="reason" class="reason-select">
      <option value="walkin">Walkin Special</option>
      <option value="manager">Manager Approved</option>
      <option value="corp">Corporate</option>
      <option value="other">Other</option>
    </select>
  </div>
{/if}

<style>
  .rate-row {
    display: flex;
    width: 100%;
    background: var(--bg-app);
    align-items: center;
    padding: var(--padding-card);
    gap: var(--gap-md);
    color: var(--fg-middle);
    border-bottom: var(--border-width-thin) solid var(--border-dim);
  }

  .rate-row[data-selected="true"] {
    background-color: var(--bg-selected);
    border-color: var(--border-base);
    border-top: var(--border-width-thin) solid var(--border-base);
  }

  .pricing {
    margin-left: auto;
    display: grid;
    grid-template-columns: 1fr 2fr 1fr;
    gap: var(--gap-xl);
    color: var(--fg-subtle);
    align-items: flex-start;
    justify-content: center;
  }
  .pricing-label {
    color: var(--fg-muted);
    text-transform: uppercase;
    letter-spacing: var(--letter-spacing-wider);
    font-size: var(--font-size-sm);
    margin-right: auto;
  }

  .pricing-value {
    font-family: var(--font-data);
    font-weight: var(--font-weight-semibold);
    font-size: var(--font-size-md);
    margin-right: auto;
  }

  .rate-info {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }

  .code {
    font-weight: var(--font-weight-semibold);
    letter-spacing: var(--letter-spacing-wider);
    color: var(--fg-subtle);
  }

  .name {
    color: var(--fg-muted);
    letter-spacing: var(--letter-spacing-wide);
    font-size: var(--font-size-sm);
  }

  .delta {
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-semibold);
    letter-spacing: var(--letter-spacing-wider);
    color: var(--color-success-bg);

    &.negative {
      color: var(--color-danger-bg);
    }
  }

  .pricing-column {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .reason-row {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    background-color: var(--color-warning-fg);
    border: var(--border-width-thin) solid (--color-warning-bg);
    padding: var(--padding-card);
    color: var(--color-warning-bg);
    font-weight: var(--font-weight-semibold);
    letter-spacing: var(--letter-spacing-wider);
    width: 100%;
    margin-left: auto;
    gap: var(--gap-sm);
  }

  .reason-select {
    background-color: var(--bg-app);
    padding: var(--padding-btn-sm);
    border: var(--border-width-thin) solid var(--color-warning-bg);
    border-radius: var(--radius-md);
  }
</style>
