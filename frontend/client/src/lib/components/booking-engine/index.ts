import RoomCard from "./RoomCard.svelte";
import BookingEngine from "./BookingEngine.svelte";
import Header from "./Header.svelte";
import OccupancySetter from "./OccupancySetter.svelte";
import RateRow from "./RateRow.svelte";
import StayDates from "./StayDates.svelte";

const BOE = {
  Header,
};

const Rates = {
  RoomCard,
  StayDates,
};

const RC = {
  OccupancySetter,
  RateRow,
};

export { BookingEngine, BOE, RC, Rates };
