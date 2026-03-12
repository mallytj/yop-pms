<script lang="ts">
  import type { ReservationItemDraft, UUID } from "$lib/types/booking_engine";
  import { getContext } from "svelte";

  import { RC } from ".";
  import Button from "../ui/Button.svelte";
  import { Copy } from "@lucide/svelte";
  import { useBookingStore } from "$lib/stores/booking.svelte";

  interface Props {
    item: ReservationItemDraft;
    idx: number;
  }

  const store = useBookingStore();

  let { item = $bindable(), idx }: Props = $props();

  $effect(() => {
    if (!item.selectedRatePlanId && store.plans.length > 0) {
      item.selectedRatePlanId = store.plans[0].id;
    }
  });
</script>

<article class="room-card-container">
  <div class="room-card">
    <div class="room-card-header">
      <h4>ROOM {item.assignedRoom?.room_name}</h4>

      <RC.OccupancySetter bind:item defaultAdults={2} defaultChildren={0} />

      {#if idx == 0}
        <div class="apply-all-btn-container ml-auto">
          <Button variant="secondary" onclick={(e) => store.applyFirstToAll()}>
            <Copy size="var(--font-size-md)" />Apply All
          </Button>
        </div>
      {/if}
    </div>
    <div class="rate-plans">
      <div role="radiogroup" aria-label="Rate Plans">
        {#each store.plans ?? [] as rp, rpIdx (rp.id)}
          <RC.RateRow
            ratePlan={rp}
            selected={rp.id === item.selectedRatePlanId}
            bind:selectedRateId={item.selectedRatePlanId}
            roomTypeId={item.bookedRoomTypeId}
            groupId={`room-${item.tempId}`}
          />
        {/each}
      </div>
    </div>
  </div>
</article>

<style>
  .room-card-container {
    display: grid;
    grid-template-rows: auto 0fr;
    transition: grid-template-rows 0.4s ease;
    overflow: visible;
    padding: var(--padding-card);
  }

  .room-card {
    border: var(--border-width-medium) solid var(--border-dim);
    border-radius: var(--radius-md);
    background-color: var(--bg-panel);
    display: flex;
    flex-direction: column;
    place-items: start;
    text-align: center;
  }

  .room-card-header {
    border-bottom: var(--border-width-medium) solid var(--border-dim);
    padding: var(--padding-card);
    display: flex;
    align-items: center;
    background-color: var(--bg-subtle);
    width: 100%;
    gap: var(--gap-xl);
  }

  .room-card-header h4 {
    font-size: var(--font-size-md);
    text-transform: uppercase;
    letter-spacing: var(--letter-spacing-extrawide);
    font-weight: var(--font-weight-bold);
    color: var(--fg-subtle);
  }

  .rate-plans {
    display: flex;
    flex-direction: column;
    width: 100%;
    overflow: visible;

    div[role="radiogroup"] {
      display: flex;
      flex-direction: column;
      width: 100%;
      overflow: visible;
    }
  }
</style>
