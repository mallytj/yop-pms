<script lang="ts">
  import { tick } from "svelte";
  import type {
    Room,
    Booking,
    ReservationDraft,
    CellSelection,
    DragState,
  } from "$types/planner_data";
  import {
    initializeBookingDrag,
    initializeCellSelection,
    updateCellSelectionDrag,
    updateBookingDrag,
    finalizeCellSelection,
    finalizeBookingUpdate,
    setupAutoScroll,
    diffDays,
    addDays,
  } from "$helpers/planner";
  import { plannerStore } from "$stores/planner_store";
  import { ToolbarPanel, GridContainer, ReservationModal } from "./";

  // --- Props ---
  interface Props {
    rooms?: Room[];
    startDate: Date;
    endDate: Date;
    isLoading?: boolean;
    onLoadMore: () => void;
    onLoadPrevious: () => Promise<void>;
    updateBooking: (
      id: string,
      newStart: Date,
      newEnd: Date,
      newRoomId: string,
    ) => void;
    onCreateReservation: (drafts: ReservationDraft[]) => void;
  }
  let {
    rooms = [],
    startDate,
    endDate,
    isLoading = false,
    onLoadMore,
    onLoadPrevious,
    updateBooking,
    onCreateReservation,
  }: Props = $props();

  // --- Local State ---
  let scrollableElement: HTMLElement = $state() as HTMLElement;
  let autoScrollInterval: number | null = $state(null);
  let multiSelectMode = $state(false);
  let selectedCells: CellSelection[] = $state([]);
  let dragState: DragState | null = $state(null);
  let showReservationModal = $state(false);

  // --- Subscriptions ---
  plannerStore.subscribe((state) => {
    multiSelectMode = state.multiSelectMode;
    selectedCells = state.selectedCells;
    dragState = state.dragState;
    showReservationModal = state.showReservationModal;
  });

  // --- Grid Math ---
  const dayNameFormat = new Intl.DateTimeFormat("en-US", { weekday: "short" });
  let totalDays = $derived(
    startDate && endDate ? diffDays(endDate, startDate) + 1 : 0,
  );
  let headerDates = $derived(
    startDate && totalDays > 0
      ? Array.from({ length: totalDays }, (_, i) => addDays(startDate, i))
      : [],
  );

  // --- Event Handlers ---
  function handleBookingDragStart(
    booking: Booking,
    roomIndex: number,
    e: PointerEvent,
  ) {
    const target = e.target as HTMLElement;
    const resizeHandle = target.closest(".resize-handle");

    if (resizeHandle) {
      return;
    }

    const bbTarget = target.closest(".booking-block") as HTMLElement;
    if (!bbTarget) return;

    const rect = bbTarget.getBoundingClientRect();
    const parentRect = scrollableElement.getBoundingClientRect();
    const newDragState = initializeBookingDrag(
      booking,
      roomIndex,
      startDate,
      rect,
      parentRect,
      scrollableElement.scrollLeft,
      scrollableElement.scrollTop,
      e.clientX,
      e.clientY,
    );

    plannerStore.setDragState(newDragState);
  }

  function handleBookingResizeStart(
    booking: Booking,
    roomIndex: number,
    e: PointerEvent,
  ) {
    const bbTarget = (e.target as HTMLElement).closest(
      ".booking-block",
    ) as HTMLElement;
    if (!bbTarget) return;

    const rect = bbTarget.getBoundingClientRect();
    const parentRect = scrollableElement.getBoundingClientRect();

    const newDragState = initializeBookingDrag(
      booking,
      roomIndex,
      startDate,
      rect,
      parentRect,
      scrollableElement.scrollLeft,
      scrollableElement.scrollTop,
      e.clientX,
      e.clientY,
    );

    newDragState.mode = "resize";
    plannerStore.setDragState(newDragState);
  }

  function handleCellDragStart(
    dayIndex: number,
    roomIndex: number,
    e: PointerEvent,
  ) {
    if ((e.target as HTMLElement).closest(".booking-block")) {
      return;
    }
    e.preventDefault();

    const newDragState = initializeCellSelection(
      dayIndex,
      roomIndex,
      startDate,
      endDate,
      e.clientX,
      e.clientY,
    );

    plannerStore.setDragState(newDragState);
  }

  function handleDragMove(e: PointerEvent, delta: { dx: number; dy: number }) {
    if (!dragState) return;

    const newDragState = { ...dragState };
    newDragState.currentX += e.movementX;
    newDragState.currentY += e.movementY;

    // Handle auto-scroll
    const interval = setupAutoScroll(e.clientX, scrollableElement);
    if (interval) {
      if (autoScrollInterval) clearInterval(autoScrollInterval);
      autoScrollInterval = interval;
    }

    // Update based on mode
    if (dragState.mode === "select-cells") {
      const updated = updateCellSelectionDrag(
        newDragState,
        delta.dx,
        totalDays,
        rooms,
        startDate,
      );
      plannerStore.setDragState(updated);
    } else {
      const updated = updateBookingDrag(
        newDragState,
        delta.dx,
        delta.dy,
        startDate,
        rooms,
      );
      plannerStore.setDragState(updated);
    }
  }

  function handleDragEnd() {
    if (autoScrollInterval) {
      clearInterval(autoScrollInterval);
      autoScrollInterval = null;
    }

    if (!dragState) return;

    if (dragState.mode === "select-cells") {
      handleSelectionEnd(dragState);
      return;
    }

    handleBookingMoveEnd(dragState);
  }

  function handleSelectionEnd(dragState: DragState) {
    const selection = finalizeCellSelection(dragState, startDate, totalDays);

    if (multiSelectMode) {
      plannerStore.addSelection(selection);
      plannerStore.setDragState(null);
      return;
    }

    // Single selection mode - open modal immediately
    plannerStore.clearSelections();
    plannerStore.addSelection(selection);
    plannerStore.openReservationModal();
    plannerStore.setDragState(null);
  }

  function handleBookingMoveEnd(dragState: DragState) {
    const result = finalizeBookingUpdate(dragState, startDate, rooms);

    if (result) {
      updateBooking(
        dragState.item_id || "",
        result.finalStart,
        result.finalEnd,
        result.finalRoomId,
      );
    }

    plannerStore.setDragState(null);
  }

  function handleRemoveSelection(idx: number) {
    plannerStore.removeSelection(idx);
  }

  let hasScrolledRight = false;

  async function handleScroll() {
    if (!scrollableElement || isLoading) return;
    if (dragState) return;

    const { scrollLeft, scrollWidth, clientWidth } = scrollableElement;

    if (scrollLeft > 200) hasScrolledRight = true;

    const scrollPercentage = (scrollLeft + clientWidth) / scrollWidth;

    // Right edge → load future dates
    if (scrollPercentage > 0.75) {
      onLoadMore();
      return;
    }

    // Left edge → load past dates (only after user has scrolled right first)
    if (hasScrolledRight && scrollLeft < 200) {
      const oldScrollLeft = scrollableElement.scrollLeft;
      const oldScrollWidth = scrollableElement.scrollWidth;

      await onLoadPrevious();
      await tick();
      await tick();

      const addedWidth = scrollableElement.scrollWidth - oldScrollWidth;
      scrollableElement.scrollLeft = oldScrollLeft + addedWidth;
    }
  }
</script>

<div class="planner-container">
  <!-- Toolbar -->
  <ToolbarPanel
    {multiSelectMode}
    selectedCellCount={selectedCells.length}
    onToggleMultiSelect={() => plannerStore.toggleMultiSelect()}
    onOpenModal={() => plannerStore.openReservationModal()}
    onCancel={() => plannerStore.reset()}
  />

  <!-- Grid -->
  <GridContainer
    {rooms}
    {headerDates}
    {totalDays}
    {startDate}
    {dragState}
    {multiSelectMode}
    {selectedCells}
    {isLoading}
    {dayNameFormat}
    bind:scrollableElement
    onScroll={handleScroll}
    onCellDragStart={handleCellDragStart}
    onBookingDragStart={handleBookingDragStart}
    onBookingResizeStart={handleBookingResizeStart}
    onDragMove={handleDragMove}
    onDragEnd={handleDragEnd}
    onRemoveSelection={handleRemoveSelection}
  />

  <!-- Reservation Modal -->
  <!-- {#if showReservationModal && selectedCells.length > 0} -->
  <ReservationModal />

  <!-- {/if} -->
</div>

<style>
  .planner-container {
    height: 100%;
    width: 100%;
    min-width: 100vw;
    overflow: hidden;
    position: relative;
    background: var(--bg-app);
  }
</style>
