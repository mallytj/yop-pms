import { writable, derived } from "svelte/store";

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

function createReservationStore() {
  const { subscribe, set, update } = writable({
    step: "rates" as "rates" | "guests" | "review",
    currentGuestPage: 0,
    isSubmitting: false,
  });

  return {
    subscribe,
    setStep: (step: "rates" | "guests" | "review") =>
      update((s) => ({ ...s, step })),
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
    setGuestPage: (page: number) =>
      update((s) => ({ ...s, currentGuestPage: page })),
    nextGuestPage: () =>
      update((s) => ({ ...s, currentGuestPage: s.currentGuestPage + 1 })),
    prevGuestPage: () =>
      update((s) => ({
        ...s,
        currentGuestPage: Math.max(0, s.currentGuestPage - 1),
      })),
    setSubmitting: (submitting: boolean) =>
      update((s) => ({ ...s, isSubmitting: submitting })),
    reset: () =>
      set({
        step: "rates",
        currentGuestPage: 0,
        isSubmitting: false,
      }),
  };
}

export const reservationStore = createReservationStore();
