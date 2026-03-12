<script lang="ts">
  import { Calendar, MoveRight } from "@lucide/svelte";
  import Input from "../ui/Input.svelte";
  import { getDaysBetween } from "$lib/helpers/booking-engine/utils";

  interface Props {
    checkInDate: string;
    checkOutDate: string;
  }

  let { checkInDate, checkOutDate }: Props = $props();

  let stayLength = $derived(getDaysBetween(checkInDate, checkOutDate));
</script>

<section class="stay-dates">
  <div class="row">
    <div class="dates">
      <span class="icon"><Calendar color="var(--border-base)" /></span>
      <p class="prefix">Stay Dates</p>
      <span class="date">
        <Input inputType="date" value={checkInDate} />
      </span>
      <span class="icon"><MoveRight color="var(--border-base)" /></span>
      <span class="date">
        <Input inputType="date" value={checkOutDate} />
      </span>
      <span class="stay-length">{stayLength} nights</span>
    </div>
  </div>
</section>

<style>
  .stay-dates {
    width: 100%;
    margin-top: var(--gap-sm);
    padding: var(--padding-card);
    padding-bottom: var(--gap-md);
  }

  .row {
    display: flex;
    padding: var(--padding-card);
    border-radius: var(--radius-md);
    border: var(--border-width-medium) solid var(--border-dim);
    background-color: var(--bg-subtle);
    flex-direction: column;
    justify-items: center;
    gap: var(--gap-lg);
  }

  .prefix {
    color: var(--fg-subtle);
    font-size: var(--font-size-md);
    letter-spacing: var(--letter-spacing-extrawide);
    text-transform: uppercase;
    margin: 0;
    height: auto;
    place-items: center;
    font-weight: var(--font-weight-bold);
  }

  .stay-length {
    font-size: var(--font-size-sm);
    color: var(--fg-subtle);
    background-color: var(--bg-panel);
    padding: var(--padding-btn-sm);
    border-radius: var(--radius-md);
    letter-spacing: var(--letter-spacing-wide);
    margin-left: auto;
    font-weight: var(--font-weight-semibold);
    border: var(--border-width-thin) solid var(--border-base);
  }

  .dates {
    display: flex;
    gap: var(--gap-md);
    justify-content: center;
    align-items: center;
  }

  .date {
    font-weight: var(--font-weight-bold);
  }
</style>
