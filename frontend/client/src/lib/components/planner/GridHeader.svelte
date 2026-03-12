<script lang="ts">
interface Props {
  headerDates: Date[];
  dayNameFormat: Intl.DateTimeFormat;
}
  let {
    headerDates,
    dayNameFormat,
  }: Props = $props();
</script>

<div class="header-cell sticky-corner">Rooms</div>
{#each headerDates as date}
  <div class="header-cell date">
    <span>{dayNameFormat.format(date)}</span>
    <span class="font-bold">{date.getDate()}</span>
  </div>
{/each}

<style>
  .header-cell {
    background: var(--bg-app);
    border-bottom: var(--border-width-thin) solid var(--border-dim);
    border-right: var(--border-width-thin) dashed var(--border-dim);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    font-size: var(--font-size-xs);
  }

  .header-cell.date {
    position: sticky;
    top: 0;
    z-index: 20;
  }

  .sticky-corner {
    position: sticky;
    top: 0;
    left: 0;
    z-index: 70;
    background: var(--bg-app);
    border-right: var(--border-width-thin) solid var(--border-dim);
    border-bottom: var(--border-width-thin) solid var(--border-dim);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .font-bold {
    font-weight: var(--font-weight-semibold);
  }
</style>
