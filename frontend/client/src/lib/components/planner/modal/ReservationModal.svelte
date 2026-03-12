<script lang="ts">
  import { reservationStore } from "$stores/reservation_modal_store";
  import { onMount } from "svelte";
  import { Modal } from ".";
  import type { RateData, RatePlan } from "$lib/types";
  import { addDays } from "$lib/helpers/planner";

  let dialogElement: HTMLDialogElement;
  const today = new Date();

  let dummyRatePlans: RatePlan[] = [
    {
      id: "1",
      basePrice: 10000,
      finalPrice: 10000,
      name: "Rate Plan 1",
      description: "Rate Plan 1",
      code: "RP1",
    },
    {
      id: "2",
      basePrice: 10000,
      finalPrice: 10000,
      name: "Rate Plan 2",
      description: "Rate Plan 2",
      code: "RP2",
    },
    {
      id: "3",
      basePrice: 10000,
      finalPrice: 5000,
      name: "Rate Plan 3",
      description: "Rate Plan 3",
      code: "RP3",
    },
  ];

  let dummyRateData: RateData[] = [
    {
      id: "1",
      calendarDate: today,
      ratePlans: dummyRatePlans,
    },
    {
      id: "2",
      calendarDate: addDays(today, 1),
      ratePlans: dummyRatePlans,
    },
    {
      id: "3",
      calendarDate: addDays(today, 2),
      ratePlans: dummyRatePlans,
    },
  ];

  onMount(() => {
    dialogElement?.showModal();
  });

  function handleCancel() {
    dialogElement?.close();
  }
</script>

<dialog
  class="reservation-dialog"
  bind:this={dialogElement}
  on:close={handleCancel}
>
  <Modal.Header />

  {#if $reservationStore.step === "rates"}
    <Modal.Rates rateData={dummyRateData} />
  {/if}
  <!-- 
  {#if $reservationStore.step === "guests"}
    <Modal.Guests />
  {/if}

  {#if $reservationStore.step === "review"}
    <Modal.Review />
  {/if} -->
</dialog>

<style>
  :global(dialog::backdrop) {
    background: rgba(0, 0, 0, 0.6);
  }
  dialog {
    border: none;
    border-radius: 12px;
    box-shadow: var(--shadow-lg);
    max-width: 750px;
    width: 90%;
    height: clamp(50vh, 90vh, 600px);
    display: flex;
    flex-direction: column;
    overflow: scroll;
    position: fixed;
    top: 50%;
    z-index: 50;
    left: 50%;
    opacity: 0;
    transform: translate(-50%, -50%);
  }

  dialog[open] {
    opacity: 1;
    animation: slideIn 0.3s ease-out;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translate(-50%, -50%) scale(0.95);
    }
    to {
      opacity: 1;
      transform: translate(-50%, -50%) scale(1);
    }
  }
</style>
