<script lang="ts">
  import { draggable } from "$lib/actions/draggable";
  import { calculateGridPosition } from "$helpers/planner";
  import type { Booking, DragState } from "$types/planner_data";

  // Props
  interface Props {
    booking: Booking;
    roomIndex: number;
    dragState: DragState | null;
    startDate: Date;
    onDragStart: (booking: Booking, roomIndex: number, e: PointerEvent) => void;
    onResizeStart: (
      booking: Booking,
      roomIndex: number,
      e: PointerEvent,
    ) => void;
    onDragMove: (e: PointerEvent, delta: { dx: number; dy: number }) => void;
    onDragEnd: () => void;
  }

  let {
    booking,
    roomIndex,
    dragState,
    startDate,
    onDragStart,
    onResizeStart,
    onDragMove,
    onDragEnd,
  }: Props = $props();

  // Local State
  let bbTarget: HTMLElement;

  // Actions
  function handleResizeStart(e: PointerEvent) {
    e.preventDefault();
    e.stopPropagation();
    onResizeStart(booking, roomIndex, e);
  }
</script>

<div
  class="booking-block"
  class:opacity-0={dragState?.id === booking.reservation_id}
  style="
    --color: {booking.status_color};
    {calculateGridPosition(
    booking.check_in_date,
    booking.check_out_date,
    startDate,
  )}
    grid-row: {roomIndex + 2};
  "
  bind:this={bbTarget}
  use:draggable={{
    onStart: (e) => onDragStart(booking, roomIndex, e),
    onMove: onDragMove,
    onEnd: onDragEnd,
    onClick: (e) => {
      console.log("Booking clicked:", booking);
    },
    clickable: true,
  }}
  role="button"
  tabindex="0"
  title={booking.guest_name}
>
  <span class="truncate">{booking.guest_name}</span>
  <span class="truncate price">£{booking.stay_price_pence! / 100}</span>
  <div
    class="resize-handle"
    role="button"
    tabindex="0"
    aria-label="Resize booking"
    use:draggable={{
      onStart: handleResizeStart,
      onMove: onDragMove,
      onEnd: onDragEnd,
    }}
  ></div>
</div>

<style>
  @import "tailwindcss";
  .booking-block {
    --complimentary: hsl(from var(--color) h s calc(l * 0.5) / 0.6);

    background-color: var(--color);
    color: var(--complimentary);
    border: var(--border-width-thin) solid var(--complimentary);
    border-left: var(--border-width-thick) solid var(--complimentary);
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-normal);
    padding: var(--padding-btn-sm);
    margin: var(--gap-xs) 0;
    position: relative;
    z-index: 5;
    border-radius: var(--radius-md);
    overflow: visible;
    display: flex;
    flex-direction: column;
    justify-content: center;
    cursor: pointer;
    pointer-events: auto;
    transition: var(--transition-normal);
    &:hover {
      background-color: rgb(from var(--color) r g b / 0.7);
      box-shadow: var(--shadow-md);
    }
  }

  .price {
    font-weight: var(--font-weight-semibold);
  }

  .booking-block.opacity-0 {
    opacity: 0 !important;
    pointer-events: none;
  }

  .resize-handle {
    position: absolute;
    right: 0;
    top: 0;
    bottom: 0;
    z-index: 100;
    width: 15%;
    cursor: col-resize;
    transition: var(--transition-normal);

    &:hover {
      background-color: rgb(from var(--color-dark-muted) r g b / 0.1);
    }
  }

  .truncate {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
