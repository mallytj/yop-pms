<!-- lib/components/planner/GridContainer.svelte -->
<script lang="ts">
  import { fly } from "svelte/transition";
  import type {
    Room,
    Booking,
    CellSelection,
    DragState,
  } from "$types/planner_data";

  import GridHeader from "./GridHeader.svelte";
  import RoomRow from "./RoomRow.svelte";
  import SelectionOverlay from "./SelectionOverlay.svelte";
  import SavedSelections from "./SavedSelections.svelte";

  // Props
  interface Props {
    rooms?: Room[];
    headerDates?: Date[];
    totalDays?: number;
    startDate: Date;
    dragState?: DragState | null;
    multiSelectMode?: boolean;
    selectedCells?: CellSelection[];
    isLoading?: boolean;
    dayNameFormat: Intl.DateTimeFormat;
    scrollableElement: HTMLElement;
    onScroll: () => void;
    onCellDragStart: (
      dayIndex: number,
      roomIdx: number,
      e: PointerEvent,
    ) => void;
    onBookingDragStart: (
      booking: Booking,
      roomIdx: number,
      e: PointerEvent,
    ) => void;
    onBookingResizeStart: (
      booking: Booking,
      roomIdx: number,
      e: PointerEvent,
    ) => void;
    onDragMove: (e: PointerEvent, delta: { dx: number; dy: number }) => void;
    onDragEnd: () => void;
    onRemoveSelection: (index: number) => void;
  }
  let {
    rooms = [],
    headerDates = [],
    totalDays = 0,
    startDate,
    dragState = null,
    multiSelectMode = false,
    selectedCells = [],
    isLoading = false,
    dayNameFormat,
    scrollableElement = $bindable(),
    onScroll,
    onCellDragStart,
    onBookingDragStart,
    onBookingResizeStart,
    onDragMove,
    onDragEnd,
    onRemoveSelection,
  }: Props = $props();
</script>

<div class="grid-container" bind:this={scrollableElement} onscroll={onScroll}>
  <div
    class="grid"
    style="grid-template-columns: 200px repeat({totalDays}, 100px);"
  >
    <!-- Grid Headers -->
    <GridHeader {headerDates} {dayNameFormat} />

    <!-- Rooms and Bookings -->
    {#each rooms as room, roomIdx (room.room_id)}
      <RoomRow
        {room}
        {roomIdx}
        {totalDays}
        {startDate}
        {dragState}
        scrollableContainer={scrollableElement}
        onCellDragStart={(dayIdx: number, e: PointerEvent) =>
          onCellDragStart(dayIdx, roomIdx, e)}
        onBookingDragStart={(booking, e) =>
          onBookingDragStart(booking, roomIdx, e)}
        onBookingResizeStart={(booking, e) =>
          onBookingResizeStart(booking, roomIdx, e)}
        {onDragMove}
        {onDragEnd}
      />
    {/each}

    <!-- Selection preview (active drag) -->
    {#if dragState?.mode === "select-cells"}
      <SelectionOverlay {dragState} />
    {/if}

    <!-- Multi-select saved selections -->
    {#if multiSelectMode}
      <SavedSelections {selectedCells} {onRemoveSelection} />
    {/if}
  </div>

  <!-- Loading overlay -->
  <div class="loading-overlay" style={`opacity: ${isLoading ? 1 : 0}`}>
    Loading more dates...
  </div>
</div>

<style>
  .grid-container {
    height: 100%;
    width: 100%;
    overflow: auto;
    position: relative;
  }

  .grid {
    display: grid;
    grid-auto-rows: 5rem;
    width: max-content;
  }

  .loading-overlay {
    position: fixed;
    bottom: 20px;
    right: 20px;
    background: var(--color-success);
    color: var(--color-light);
    padding: var(--padding-btn);
    border-radius: var(--radius-md);
    transition: var(--transition-fast);
    z-index: 100;
  }
</style>
