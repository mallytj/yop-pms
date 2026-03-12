/**
 * Converts a Date object to a YYYY-MM-DD string format.
 *
 * @param {Date} date - The date to convert.
 * @returns {string} The formatted local date string.
 */
function dateToString(date: Date): string {
  // Convert to local date string in "YYYY-MM-DD" format
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0"); // Months are 0-indexed
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

/**
 * Parses a YYYY-MM-DD string into a Date object.
 *
 * @param {string | undefined} dateStr - The date string to parse.
 * @returns {Date} The parsed Date object.
 */
function parseLocalYMD(dateStr: string | undefined): Date {
  const [y, m, d] = dateStr?.split("-").map(Number) ?? [0, 0, 0];
  return new Date(y, m - 1, d);
}

/**
 * Converts a Date object to a short string format (e.g., "Jan 1").
 *
 * @param {Date} date - The date to convert.
 * @returns {string} The formatted short date string.
 */
function dateToShortString(date: Date): string {
  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
  });
}

/**
 * Calculates the difference in days between two Date objects.
 *
 * @param {Date} dateA - The first date.
 * @param {Date} dateB - The second date.
 * @returns {number} The difference in days.
 */
function diffDays(dateA: Date, dateB: Date): number {
  const a = new Date(dateA.getFullYear(), dateA.getMonth(), dateA.getDate());
  const b = new Date(dateB.getFullYear(), dateB.getMonth(), dateB.getDate());
  const msPerDay = 1000 * 60 * 60 * 24;
  return Math.round((a.getTime() - b.getTime()) / msPerDay);
}

/**
 * Adds a specific number of days to a Date object.
 *
 * @param {Date} date - The starting date.
 * @param {number} days - The number of days to add.
 * @returns {Date} A new Date object with the added days.
 */
function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

export { dateToString, parseLocalYMD, diffDays, addDays, dateToShortString };
