export interface EdgeLoadCheckInput {
	hasScrolledHorizontally: boolean;
	loading: boolean;
	pendingLoadDirection: 'past' | 'future' | null;
	columns: { index: number }[];
	totalDates: number;
	edgeThreshold: number;
}

/**
 * Decides whether the grid should fetch more past/future days given the
 * currently visible virtualized columns.
 *
 * @remarks
 * Pulled out of the component's `$effect` as a pure function: Svelte only
 * tracks reactive reads that actually execute, so a guard clause that
 * short-circuits before reading a dependency silently unsubscribes the
 * effect from it. Keeping the decision here means the effect can force-read
 * every dependency unconditionally and delegate the branching to something
 * ordinary unit tests can cover.
 */
export function nextEdgeLoadDirection({
	hasScrolledHorizontally,
	loading,
	pendingLoadDirection,
	columns,
	totalDates,
	edgeThreshold
}: EdgeLoadCheckInput): 'past' | 'future' | null {
	if (!hasScrolledHorizontally || loading || pendingLoadDirection || columns.length === 0) {
		return null;
	}

	const firstVisibleIndex = columns[0].index;
	const lastVisibleIndex = columns[columns.length - 1].index;

	if (firstVisibleIndex <= edgeThreshold) return 'past';
	if (lastVisibleIndex >= totalDates - 1 - edgeThreshold) return 'future';
	return null;
}

/**
 * Positions the initial viewport at `today` instead of the loaded range's
 * start.
 *
 * @remarks
 * Landing at index 0 pins the scrollbar thumb to its track minimum —
 * physically impossible to drag further left — even though a lookback
 * buffer of already-loaded days sits there. Starting at `today` puts that
 * buffer to the left of the viewport as real scroll headroom.
 */
export function initialScrollLeftForToday(
	dates: string[],
	today: string,
	cellWidth: number
): number {
	const todayIndex = dates.indexOf(today);
	// `todayIndex * cellWidth` is 0 whether todayIndex is 0 or negative (not
	// found), so `> 0` vs `>= 0` here is unobservable — verified equivalent.
	// Stryker disable next-line EqualityOperator
	return todayIndex > 0 ? todayIndex * cellWidth : 0;
}
