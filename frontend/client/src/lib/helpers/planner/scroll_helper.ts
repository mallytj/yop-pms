// lib/helpers/scrollHelper.ts
export interface ScrollOptions {
  element: HTMLElement;
  onScroll?: (progress: number) => void;
  loadMoreThreshold?: number;
}

/**
 * Sets up auto-scrolling for an element based on mouse position relative to boundaries.
 *
 * @param {number} mouseX - The current X coordinate of the mouse.
 * @param {HTMLElement} element - The scrollable container element.
 * @param {number} [speed=15] - The speed of scrolling per interval tick.
 * @param {number} [sensorDistance=50] - The padding from edge to trigger auto-scroll.
 * @returns {number | null} The interval ID string if auto-scrolling was triggered, or null.
 */
export function setupAutoScroll(
  mouseX: number,
  element: HTMLElement,
  speed: number = 15,
  sensorDistance: number = 50,
): number | null {
  const rect = element.getBoundingClientRect();

  if (mouseX > rect.right - sensorDistance) {
    return window.setInterval(() => {
      element.scrollLeft += speed;
    }, 16);
  } else if (mouseX < rect.left + sensorDistance) {
    return window.setInterval(() => {
      element.scrollLeft -= speed;
    }, 16);
  }

  return null;
}

/**
 * Handles checking scroll progress on a container and loading more content if near the end.
 *
 * @param {HTMLElement} scrollElement - The HTML element managing the scroll.
 * @param {() => void} onLoadMore - The callback function to execute when the threshold is reached.
 * @param {number} [threshold=0.75] - The scroll depth ratio required to trigger loading.
 */
export function handleContainerScroll(
  scrollElement: HTMLElement,
  onLoadMore: () => void,
  threshold: number = 0.75,
): void {
  console.log("Scroll event triggered");
  if (!scrollElement) return;

  const { scrollLeft, scrollWidth, clientWidth } = scrollElement;

  // Only trigger when we've scrolled to the right AND we're near the end
  const scrollPercentage = (scrollLeft + clientWidth) / scrollWidth;

  console.log(
    `Scroll: ${scrollLeft}, Width: ${scrollWidth}, Client: ${clientWidth}, Percentage: ${(scrollPercentage * 100).toFixed(1)}%`,
  );

  if (scrollPercentage > threshold) {
    console.log("Near end - calling onLoadMore");
    onLoadMore();
  }
}
