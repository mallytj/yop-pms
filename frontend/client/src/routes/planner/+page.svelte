<!-- src/routes/planner/+page.svelte -->

<script lang="ts">
  import { onMount } from "svelte";
  import { Planner } from "$components/planner";
  import type { Room, Booking, PlannerData } from "$types/planner_data";
  import Toaster from "$components/Toaster.svelte";
  import { addToast } from "$lib/stores/toast_store";
  import { plannerStore } from "$lib/stores/planner_store";
  import { diffDays } from "$helpers/planner";
  import "$lib/styles/app.css";
  import Spinner from "$lib/components/ui/Spinner.svelte";

  let roomMap = new Map<string, Room>();
  let loading = false;
  let error = "";

  let gridStartDate: Date | null = null;
  let gridEndDate: Date | null = null;

  const CHUNK_DAYS = 30;

  async function fetchPlannerData(
    start: string,
    end: string,
  ): Promise<PlannerData | null> {
    const res = await fetch(
      `http://localhost:8080/v1/planner?startDate=${start}&endDate=${end}`,
    );
    if (!res.ok) throw new Error("API Failed");
    return await res.json();
  }

  async function loadNextChunk() {
    if (loading) return;
    loading = true;

    try {
      let fetchStart: Date;
      let fetchEnd: Date;

      if (!gridEndDate) {
        // First load
        fetchStart = new Date();
        fetchEnd = new Date(fetchStart);
        fetchEnd.setDate(fetchEnd.getDate() + CHUNK_DAYS);
      } else {
        // Subsequent load
        fetchStart = new Date(gridEndDate);
        fetchStart.setDate(fetchStart.getDate() + 1);
        fetchEnd = new Date(fetchStart);
        fetchEnd.setDate(fetchEnd.getDate() + CHUNK_DAYS);
      }

      const dateToString = (d: Date) => d.toISOString().split("T")[0];
      const data = await fetchPlannerData(
        dateToString(fetchStart),
        dateToString(fetchEnd),
      );

      if (!gridStartDate && data?.start_date) {
        gridStartDate = new Date(data.start_date);
      }
      if (data?.end_date) {
        gridEndDate = new Date(data.end_date);
      }

      // Merge rooms (your original logic)
      data?.rooms.forEach((newRoom: Room) => {
        if (roomMap.has(newRoom.room_id)) {
          const existing = roomMap.get(newRoom.room_id)!;
          const existingIds = new Set(
            existing.reservations?.map((r: Booking) => r.reservation_id),
          );
          const fresh = newRoom.reservations?.filter(
            (r: Booking) => !existingIds.has(r.reservation_id),
          );

          roomMap.set(newRoom.room_id, {
            ...existing,
            reservations: [...(existing.reservations || []), ...(fresh || [])],
          });
        } else {
          roomMap.set(newRoom.room_id, newRoom);
        }
      });

      roomMap = new Map(roomMap);
    } catch (e) {
      console.error(e);
      addToast("Failed to load data", "error");
    } finally {
      loading = false;
    }
  }

  async function loadPreviousChunk() {
    if (loading || !gridStartDate) return;
    loading = true;

    try {
      const fetchEnd = new Date(gridStartDate);
      fetchEnd.setDate(fetchEnd.getDate() - 1);
      const fetchStart = new Date(fetchEnd);
      fetchStart.setDate(fetchStart.getDate() - CHUNK_DAYS);

      // Limit: don't load more than 1 year back
      const minDate = new Date();
      minDate.setFullYear(minDate.getFullYear() - 1);
      if (fetchStart < minDate) fetchStart.setTime(minDate.getTime());
      if (fetchStart >= fetchEnd) return;

      const dateToString = (d: Date) => d.toISOString().split("T")[0];
      const data = await fetchPlannerData(
        dateToString(fetchStart),
        dateToString(fetchEnd),
      );
      if (!data) return;

      const newStartDate = new Date(data.start_date);
      const shiftDays = diffDays(gridStartDate!, newStartDate);

      // Shift stored CellSelection indices so they still point at the same dates
      plannerStore.shiftSelections(shiftDays);

      gridStartDate = newStartDate;

      // Merge rooms — prepend new reservations before existing ones
      data.rooms.forEach((newRoom: Room) => {
        if (roomMap.has(newRoom.room_id)) {
          const existing = roomMap.get(newRoom.room_id)!;
          const existingIds = new Set(
            existing.reservations?.map((r: Booking) => r.reservation_id),
          );
          const fresh = newRoom.reservations?.filter(
            (r: Booking) => !existingIds.has(r.reservation_id),
          );
          roomMap.set(newRoom.room_id, {
            ...existing,
            reservations: [...(fresh || []), ...(existing.reservations || [])],
          });
        } else {
          roomMap.set(newRoom.room_id, newRoom);
        }
      });

      roomMap = new Map(roomMap);
    } catch (e) {
      console.error(e);
      addToast("Failed to load previous data", "error");
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadNextChunk();
  });

  async function handleBookingUpdate(
    id: string,
    newStart: Date,
    newEnd: Date,
    newRoomId: string,
  ) {
    // Your original logic here...
    let existingBooking: Booking | null = null;
    let oldRoomId: string | null = null;

    for (const [roomId, room] of roomMap) {
      const booking = room.reservations?.find(
        (r) => r.reservation_item_id === id,
      );
      if (booking) {
        existingBooking = booking;
        oldRoomId = roomId;
        break;
      }
    }

    if (!existingBooking || !oldRoomId) {
      console.warn(`Booking ${id} not found`);
      return;
    }

    const dateToString = (d: Date) => d.toISOString().split("T")[0];
    const newStartStr = dateToString(newStart);
    const newEndStr = dateToString(newEnd);

    const roomChanged = oldRoomId !== newRoomId;
    const datesChanged =
      existingBooking.check_in_date !== newStartStr ||
      existingBooking.check_out_date !== newEndStr;

    if (!roomChanged && !datesChanged) return;

    const previousRoomMap = roomMap;

    const optimisticBooking: Booking = {
      ...existingBooking,
      check_in_date: newStartStr,
      check_out_date: newEndStr,
    };

    roomMap = applyBookingUpdate(roomMap, {
      booking: optimisticBooking,
      oldRoomId,
      newRoomId,
      roomChanged,
    });

    try {
      const res = await fetch(
        `http://localhost:8080/v1/reservation_item/${id}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            assigned_room_id: newRoomId,
            check_in_date: newStart.toISOString(),
            check_out_date: newEnd.toISOString(),
          }),
        },
      );

      if (!res.ok) throw new Error(`API returned ${res.status}`);
      console.log("✅ Update successful");
    } catch (e) {
      addToast("Failed to update booking. Please try again.", "error");
      roomMap = previousRoomMap;
    }
  }

  function applyBookingUpdate(
    currentMap: Map<string, Room>,
    params: {
      booking: Booking;
      oldRoomId: string;
      newRoomId: string;
      roomChanged: boolean;
    },
  ): Map<string, Room> {
    const { booking, oldRoomId, newRoomId, roomChanged } = params;
    const newMap = new Map(currentMap);
    const bookingId = booking.reservation_item_id;

    if (roomChanged) {
      const oldRoom = newMap.get(oldRoomId);
      if (oldRoom) {
        newMap.set(oldRoomId, {
          ...oldRoom,
          reservations:
            oldRoom.reservations?.filter(
              (r) => r.reservation_item_id !== bookingId,
            ) ?? [],
        });
      }

      const newRoom = newMap.get(newRoomId);
      if (newRoom) {
        newMap.set(newRoomId, {
          ...newRoom,
          reservations: [...(newRoom.reservations ?? []), booking],
        });
      }
    } else {
      const room = newMap.get(oldRoomId);
      if (room) {
        newMap.set(oldRoomId, {
          ...room,
          reservations:
            room.reservations?.map((r) =>
              r.reservation_item_id === bookingId ? booking : r,
            ) ?? [],
        });
      }
    }

    return newMap;
  }

  $: sortedRooms = [...roomMap.values()].sort((a, b) =>
    a.room_name.localeCompare(b.room_name),
  );
</script>

<main>
  {#if error}
    <div class="error">{error}</div>
  {/if}
  <Toaster />

  {#if gridStartDate && gridEndDate}
    <Planner
      rooms={sortedRooms}
      startDate={gridStartDate}
      endDate={gridEndDate}
      isLoading={loading}
      onLoadMore={loadNextChunk}
      onLoadPrevious={loadPreviousChunk}
      updateBooking={handleBookingUpdate}
      onCreateReservation={() => console.log("Create Reservation")}
    />
  {:else}
    <div class="center-screen"><Spinner size="lg" /></div>
  {/if}
</main>

<style>
  main {
    height: 100vh;
    overflow: hidden;
  }
  .center-screen {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100%;
  }
  .error {
    color: var(--color-danger);
    text-align: center;
    padding: var(--padding-card);
  }
</style>
