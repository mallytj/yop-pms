<script lang="ts">
  import { buildRateMap } from "$helpers/booking-engine";
  import type {
    BookingReferenceData,
    RateMapResponse,
    RatePlan,
    ReservationDraft,
    ReservationItemDraft,
    RoomData,
  } from "$types/booking_engine";
  import { BookingStore, setBookingContext } from "$stores/booking.svelte";
  import { BOE, Rates } from ".";

  interface Props {
    checkInDate: string;
    checkOutDate: string;
    rooms: RoomData[];
    ratePlans: RatePlan[];
    rateMap: RateMapResponse;
  }

  const { checkInDate, checkOutDate, rooms, ratePlans, rateMap }: Props =
    $props();

  let store = $state<BookingStore | null>(null);
  let referenceData = $state<BookingReferenceData | null>(null);

  $effect(() => {
    if (ratePlans && rateMap && rooms) {
      referenceData = { ratePlans, rateMap: buildRateMap(rateMap) };

      const items: ReservationItemDraft[] = rooms.map((room) => ({
        tempId: crypto.randomUUID(),
        selectedRatePlanId: null,
        bookedRoomTypeId: room.room_type_id,
        assignedRoomId: room.id,
        assignedRoom: room,
        adults: 2,
        children: 0,
        dailyRates: [],
      }));

      const initalReservationDraft: ReservationDraft = {
        globalCheckInDate: checkInDate,
        globalCheckOutDate: checkOutDate,
        items: items,
      };

      if (!store) {
        store = setBookingContext(
          initalReservationDraft,
          referenceData?.rateMap ?? new Map(),
          referenceData?.ratePlans ?? [],
        );
      }
    }
  });

  let items = $derived(store?.draft.items);
</script>

{#if store}
  <div class="booking-engine">
    <BOE.Header />

    <main class="booking-engine-main">
      <Rates.StayDates {checkInDate} {checkOutDate} />

      {#each items as item, i (item.tempId)}
        <Rates.RoomCard bind:item={items![i]} idx={i} />
      {/each}
    </main>
  </div>
{/if}

<style>
  .booking-engine {
    height: 100%;
    overflow-y: scroll;
  }
  .booking-engine-main {
    height: 100%;
  }
</style>
