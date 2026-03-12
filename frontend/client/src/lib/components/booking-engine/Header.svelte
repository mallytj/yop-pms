<script lang="ts">
  import { Button } from "$components/ui";
  import { type Step } from "$types/booking_engine";
  import { useBookingStore } from "$stores/booking.svelte";

  const store = useBookingStore();
</script>

<div class="modal-header">
  <h2>New Reservation(s)</h2>

  <div class="steps">
    {#each ["rates", "guests", "review"] as s, i}
      <Button
        variant={store.step == s ? "primary" : "secondary"}
        onclick={() => store.setStep(s as Step)}
      >
        <span class="step-number">{i + 1}</span>
        <span class="step-label">{s.charAt(0).toUpperCase() + s.slice(1)}</span>
      </Button>
    {/each}
  </div>
</div>

<style>
  .modal-header {
    display: flex;
    flex-direction: column;
    background-color: var(--bg-subtle);
    align-items: center;
    padding: var(--padding-card);
    gap: var(--gap-sm);
  }

  .modal-header h2 {
    font-family: var(--font-ui);
    font-weight: var(--font-weight-semibold);
    font-size: var(--font-size-xl);
    color: var(--text-secondary);
  }

  .steps {
    display: flex;
    gap: var(--gap-sm);
  }

  .step-number {
    font-weight: var(--font-weight-bold);
  }
</style>
