/**
 * Formats a given price in pence to a string represented in pounds (£)
 *
 * @param {number} pence - The price amount in pence.
 * @returns {string} The formatted price string (e.g., £10.00).
 * @example formatPrice(1000) => £10.00
 */
function formatPrice(pence: number): string {
  return `${pence < 0 ? "-" : ""}£${Math.abs(pence / 100).toFixed(2)}`;
}

export { formatPrice };
