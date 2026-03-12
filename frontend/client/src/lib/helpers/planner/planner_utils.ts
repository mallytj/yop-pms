import {
  parseLocalYMD,
  dateToString,
  diffDays,
  addDays,
} from "./date_conversions.ts";
// --- Grid Math ---
/**
 * Calculates the CSS grid position (column start and span) for a booking.
 *
 * @param {string} bookingStart - The start date of the booking.
 * @param {string} bookingEnd - The end date of the booking.
 * @param {Date} gridStartDate - The start date of the grid.
 * @returns {string} The CSS grid-column attribute string for positioning.
 */
function calculateGridPosition(
  bookingStart: string,
  bookingEnd: string,
  gridStartDate: Date,
): string {
  const start = parseLocalYMD(bookingStart);
  const end = parseLocalYMD(bookingEnd);

  const startOffset = diffDays(start, gridStartDate) + 2; // +2 for Column headers
  const duration = Math.max(1, diffDays(end, start));

  return `grid-column: ${startOffset} / span ${duration};`;
}

export {
  calculateGridPosition,
  parseLocalYMD,
  dateToString,
  diffDays,
  addDays,
};
