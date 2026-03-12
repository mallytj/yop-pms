<script lang="ts">
  import { dateToShortString } from "$helpers/planner/date_conversions";
  import Button from "$lib/components/ui/Button.svelte";
  import type { RateData, RatePlan } from "$types";
  import RatePlanCard from "./RatePlanCard.svelte";
  import { aggregateRatePlans } from "$helpers/reservation_modal";

  interface Props {
    rateData: RateData[];
  }


  let { rateData }: Props = $props();

  let showAllRates = $state(false);
  // svelte-ignore state_referenced_locally
  let selectedRatePlanId = $state(rateData[0]?.ratePlans[0]?.id);
  let selectedRatePlan = $derived(
    rateData
      .flatMap((d) => d.ratePlans)
      .find((p) => p.id === selectedRatePlanId) || rateData[0]?.ratePlans[0],
  );
  let aggRatePlans = $derived(aggregateRatePlans(rateData));
</script>

<section class="rates">
  <h2>Set Rates</h2>
  <Button variant="secondary" onclick={() => (showAllRates = !showAllRates)}
    >&ShortDownArrow; Show daily rates</Button
  >
  <div class="rates-container" class:expanded={showAllRates}>
    <div class="rate-card rate-card--full">
      <div class="rate-header">
        <h4>Full Stay</h4>
      </div>
      <div class="rate-plans">
        {#each aggRatePlans as rp, rpIdx (rp.id)}
          <RatePlanCard
            name={rp.name}
            code={rp.code}
            basePrice={rp.basePrice}
            finalPrice={rp.finalPrice}
            description={rp.description}
            selected={rp.id === selectedRatePlan.id}
            onClick={() => (selectedRatePlanId = rp.id)}
          />
        {/each}
      </div>
    </div>
  </div>
</section>

<style>
  .rates {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--gap-md);
    padding: var(--padding-card);
  }

  .rates h2 {
    font-size: var(--text-lg);
    font-weight: var(--font-weight-semibold);
    color: var(--color-dark);
  }

  .rates-grid {
    display: flex;
    flex-wrap: wrap;
    gap: var(--gap-md);
    width: 100%;
    max-height: 0;
    flex: 1 1 100%;
    transition: var(--transition-slow);
  }

  .rates-container {
    display: grid;
    grid-template-rows: auto 0fr;
    transition: grid-template-rows 0.4s ease;
    overflow: visible;
  }

  .rates-container.expanded {
    grid-template-rows: auto 1fr;
    gap: var(--gap-md);
    overflow: visible;

    &.rates-grid {
      max-height: 100%;
    }
  }

  .rate-card {
    border: var(--border-width-medium) solid var(--border-dim);
    border-radius: var(--radius-md);
    background-color: var(--bg-panel);
    padding: var(--padding-card);
    display: flex;
    flex-direction: column;
    place-items: start;
    text-align: center;
    gap: var(--gap-sm);
  }

  .rate-header h4 {
    font-size: var(--font-size-xs);
    text-transform: uppercase;
    letter-spacing: var(--letter-spacing-widest);
    font-weight: var(--font-weight-semibold);
    color: var(--fg-subtle);
    padding-bottom: var(--gap-md);
  }

  .rate-plans {
    display: flex;
    flex-direction: column;
    gap: var(--gap-sm);
    overflow: visible;
  }

  .rate-card--full {
    align-self: start;
    place-items: center;
    contain: layout;
    height: fit-content;
    width: 100%;
    overflow-y: scroll;
    overflow-x: clip;
  }

  .rate-card--full .rate-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
  }

  .rate-card--full .rate-plans {
    width: 100%;
    display: grid;
    grid-template-columns: repeat(3, minmax(200px, 1fr));
    gap: var(--gap-sm);
    grid-auto-columns: auto;
  }
</style>
