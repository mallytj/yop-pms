import { writable } from "svelte/store";

export interface ReservationBase {
  id: string;
  roomId: string;
  roomName: string;
  checkInDate: string;
  checkOutDate: string;
  nights: number;
}

export interface RateData {
  id: string;
  ratePerNight: number;
  totalPrice: number;
}

export interface GuestData {
  id: string;
  guestName: string;
  guestEmail: string;
  guestPhone: string;
}

type Step = "rates" | "guests" | "review";

function createReservationStore() {
  const { subscribe, set, update } = writable({
    step: "rates" as Step,
  });

  return {
    subscribe,
    setStep: (step: Step) => update((s) => ({ ...s, step })),
    nextStep: () =>
      update((s) => ({
        ...s,
        step:
          s.step === "rates"
            ? "guests"
            : s.step === "guests"
              ? "review"
              : "rates",
      })),
    previousStep: () =>
      update((s) => ({
        ...s,
        step:
          s.step === "review"
            ? "guests"
            : s.step === "guests"
              ? "rates"
              : "rates",
      })),
    reset: () =>
      set({
        step: "rates",
      }),
  };
}

export const reservationStore = createReservationStore();
export type { Step };
