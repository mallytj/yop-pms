// lib/actions/draggable.ts
export interface DragOptions {
  onStart?: (e: PointerEvent) => void;
  onMove?: (e: PointerEvent, delta: { dx: number; dy: number }) => void;
  onEnd?: (e: PointerEvent) => void;
  clickable?: boolean;
  onClick?: (e: PointerEvent) => void;
}

export function draggable(element: HTMLElement, options: DragOptions = {}) {
  let startX = 0;
  let startY = 0;
  let startTime = 0;
  const TIME_THRESHOLD = 200; // ms

  function handlePointerDown(e: PointerEvent) {
    startX = e.clientX;
    startY = e.clientY;
    startTime = e.timeStamp;

    options.onStart?.(e);

    function handlePointerMove(moveEvent: PointerEvent) {
      const dx = moveEvent.clientX - startX;
      const dy = moveEvent.clientY - startY;
      options.onMove?.(moveEvent, { dx, dy });
    }

    function handlePointerUp(upEvent: PointerEvent) {
      const duration = upEvent.timeStamp - startTime;
      if (options.clickable && duration < TIME_THRESHOLD) {
        options.onClick?.(upEvent);
      }
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
      window.removeEventListener("pointercancel", handlePointerUp);
      options.onEnd?.(upEvent);
    }

    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", handlePointerUp);
    window.addEventListener("pointercancel", handlePointerUp);
  }

  element.addEventListener("pointerdown", handlePointerDown);

  return {
    destroy() {
      element.removeEventListener("pointerdown", handlePointerDown);
    },
  };
}
