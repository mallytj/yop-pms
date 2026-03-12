import { writable, derived } from "svelte/store";
import type { DragState, CellSelection } from "../types/planner_data.ts";


export interface PlannerStore {
  multiSelectMode: boolean;
  selectedCells: CellSelection[];
  dragState: DragState | null;
  showReservationModal: boolean;
}

function createPlannerStore() {
  const { subscribe, set, update } = writable<PlannerStore>({
    multiSelectMode: false,
    selectedCells: [],
    dragState: null,
    showReservationModal: false,
  });

  return {
    subscribe,
    toggleMultiSelect: () =>
      update((s) => ({ ...s, multiSelectMode: !s.multiSelectMode })),
    addSelection: (selection: CellSelection) =>
      update((s) => ({
        ...s,
        selectedCells: [...s.selectedCells, selection],
      })),
    removeSelection: (index: number) =>
      update((s) => ({
        ...s,
        selectedCells: s.selectedCells.filter((_, i) => i !== index),
      })),
    clearSelections: () => update((s) => ({ ...s, selectedCells: [] })),
    shiftSelections: (dayShift: number) =>
      update((s) => ({
        ...s,
        selectedCells: s.selectedCells.map((sel) => ({
          ...sel,
          startDay: sel.startDay + dayShift,
          endDay: sel.endDay + dayShift,
        })),
      })),
    setDragState: (dragState: DragState | null) =>
      update((s) => ({ ...s, dragState })),
    openReservationModal: () =>
      update((s) => ({ ...s, showReservationModal: true })),
    closeReservationModal: () =>
      update((s) => ({ ...s, showReservationModal: false, selectedCells: [] })),
    reset: () =>
      set({
        multiSelectMode: false,
        selectedCells: [],
        dragState: null,
        showReservationModal: false,
      }),
  };
}

export const plannerStore = createPlannerStore();

export const isMultiSelectMode = derived(
  plannerStore,
  ($s) => $s.multiSelectMode,
);
export const selectedCellsCount = derived(
  plannerStore,
  ($s) => $s.selectedCells.length,
);
export const hasActiveDrag = derived(
  plannerStore,
  ($s) => $s.dragState !== null,
);
