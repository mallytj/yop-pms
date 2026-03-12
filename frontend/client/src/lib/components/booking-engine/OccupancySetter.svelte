<script lang="ts">
  import { UsersRound } from "@lucide/svelte";
  import Stepper from "../ui/Stepper.svelte";
  import {
    useBookingStore,
    type BookingStore,
  } from "$lib/stores/booking.svelte";
  import type { ReservationItemDraft } from "$lib/types/booking_engine";

  interface Props {
    defaultAdults: number;
    defaultChildren: number;
    item: ReservationItemDraft;
  }
  
  const store = useBookingStore();

  let { defaultAdults, defaultChildren, item = $bindable() }: Props = $props();
</script>

<div class="occupancy-setter">
  <span class="prefix-icon"
    ><UsersRound color="var(--fg-subtle)" size="var(--font-size-md)" /></span
  >

  <div class="stepper stepper--adults">
    <label for="adults-stepper">Adults</label>
    <Stepper
      id="adults-stepper"
      max={4}
      min={1}
      defaultVal={defaultAdults}
      onchange={(v) => store?.updateOccupancy(item.tempId, v, item.children!)}
    />
  </div>
  <span class="separator"></span>

  <div class="stepper stepper--children">
    <label for="children-stepper">Children</label>
    <Stepper
      id="children-stepper"
      max={2}
      min={0}
      defaultVal={defaultChildren}
      onchange={(v) => store?.updateOccupancy(item.tempId, item.adults!, v)}
    />
  </div>
</div>

<style>
  :root {
    --bg-setter: var(--palette-stone-50);
  }
  .occupancy-setter {
    display: flex;
    align-items: center;
    background-color: var(--bg-setter);
    border: var(--border-width-thin) solid var(--border-base);
    padding: var(--padding-btn);
    border-radius: var(--radius-md);
    color: var(--fg-subtle);
  }

  .stepper {
    display: flex;
    place-items: center;
    gap: var(--gap-sm);
    padding: 0 var(--gap-sm);
  }

  .stepper--adults {
    border-right: var(--border-width-thin) solid var(--border-base);
  }

  .prefix-icon {
    margin-right: var(--gap-sm);
  }

  span.separator {
    width: 2px;
    height: 100%;
    background-color: var(--border-base);
  }

  label {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-semibold);
    letter-spacing: var(--letter-spacing-wider);
  }
</style>
