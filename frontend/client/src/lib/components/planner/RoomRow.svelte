<!-- lib/components/planner/RoomRow.svelte -->
<script lang="ts">
  import { fly } from "svelte/transition";
  import type { Room, Booking, DragState } from "$types/planner_data";
  import EmptyCell from "./EmptyCell.svelte";
  import BookingBlock from "./BookingBlock.svelte";

  // Props
  interface Props {
    room: Room;
    roomIdx: number;
    totalDays: number;
    startDate: Date;
    dragState?: DragState | null;
    scrollableContainer: HTMLElement;
    onCellDragStart: (dayIndex: number, e: PointerEvent) => void;
    onBookingDragStart: (booking: Booking, e: PointerEvent) => void;
    onBookingResizeStart: (booking: Booking, e: PointerEvent) => void;
    onDragMove: (e: PointerEvent, delta: { dx: number; dy: number }) => void;
    onDragEnd: () => void;
  }
  let {
    room,
    roomIdx,
    totalDays,
    startDate,
    dragState = null,
    scrollableContainer,
    onCellDragStart,
    onBookingDragStart,
    onBookingResizeStart,
    onDragMove,
    onDragEnd,
  }: Props = $props();
</script>

<!-- Room Header -->
<div
  class="room-cell fixed-left sticky"
  style="grid-row: {roomIdx + 2}; background-color: {roomIdx % 2 == 0
    ? 'var(--bg-front)'
    : 'var(--bg-subtle)'}"
>
  <strong>{room.room_name}</strong>
  <small>{room.room_type_code}</small>
</div>

<!-- Empty Cells -->
{#each Array(totalDays) as _, dayIdx}
  <div style="grid-column: {dayIdx + 2}; grid-row: {roomIdx + 2};">
    <EmptyCell
      dayIndex={dayIdx}
      roomIndex={roomIdx}
      onDragStart={(dayIndex, roomIdx, e) => onCellDragStart(dayIndex, e)}
      {onDragMove}
      {onDragEnd}
    />
  </div>
{/each}

<!-- Bookings -->
{#each room.reservations as booking (booking.reservation_id)}
  <BookingBlock
    {booking}
    roomIndex={roomIdx}
    {dragState}
    {startDate}
    onDragStart={(b, roomIdx, e) => onBookingDragStart(b, e)}
    onResizeStart={(b, roomIdx, e) => onBookingResizeStart(b, e)}
    {onDragMove}
    {onDragEnd}
  />
{/each}

<!-- Shadow for move/resize -->
{#if dragState && dragState.mode !== "select-cells" && dragState.shadowRowIndex === roomIdx + 2}
  <div
    class="shadow opacity-50 pointer-events-none"
    transition:fly={{ y: -20, duration: 100 }}
    style="
      grid-column: {dragState.shadowStartOffset} / span {dragState.shadowDuration};
      grid-row: {dragState.shadowRowIndex};
    "
  ></div>
{/if}

<!-- Row line divider -->
<div
  class="row-line"
  style="grid-column: 1 / -1; grid-row: {roomIdx + 2};"
></div>

<style>
  .room-cell {
    position: sticky;
    left: 0;
    z-index: 60;
    background: var(--bg-app);
    border-right: var(--border-width-thin) solid var(--border-base);
    border-bottom: var(--border-width-thin) solid var(--border-base);
    display: flex;
    height: 100%;
    flex-direction: column;
    justify-content: center;
    padding: var(--padding-btn);
    grid-column: 1;
  }

  .shadow {
    margin: var(--gap-sm) 0;
    background-color: var(--bg-middle);
    box-shadow: var(--shadow-sm);
    position: relative;
    z-index: 5;
    border-radius: var(--radius-md);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .shadow.opacity-50 {
    opacity: 0.5;
  }

  .shadow.pointer-events-none {
    pointer-events: none;
  }

  .row-line {
    border-bottom: var(--border-width-thin) solid var(--border-base);
    height: var(--border-width-thin);
    pointer-events: none;
    z-index: 0;
    grid-column: 1 / -1;
  }
</style>
