<script lang="ts">
  import BookingEngine from "$lib/components/booking-engine/BookingEngine.svelte";
  import type { RoomData } from "$lib/types/booking_engine";
  import "$lib/styles/app.css";
  import type { PageData } from "./$types";

  let { data } = $props<{ data: PageData }>();

  let { checkInDate, checkOutDate, ratePlans, rateMap } = data;

  let dummyRooms: RoomData[] = [
    {
      id: crypto.randomUUID(),
      room_id: crypto.randomUUID(),
      room_name: "001",
      room_type_id: "3df4cc45-0bad-4ee8-89f2-ca822cffb38b",
      room_type_code: "CO",
      room_type_name: "Cozy",
    },
    {
      id: crypto.randomUUID(),
      room_id: crypto.randomUUID(),
      room_name: "002",
      room_type_id: "4222118b-32e0-4a71-9320-e8750607b495",
      room_type_code: "DX",
      room_type_name: "Deluxe",
    },
  ];
</script>

<BookingEngine
  checkInDate={checkInDate ?? ""}
  checkOutDate={checkOutDate ?? ""}
  rooms={dummyRooms}
  ratePlans={ratePlans ?? []}
  rateMap={rateMap!}
></BookingEngine>
