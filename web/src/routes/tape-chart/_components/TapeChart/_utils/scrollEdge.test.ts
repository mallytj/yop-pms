import { describe, it, expect } from 'vitest';
import { nextEdgeLoadDirection, initialScrollLeftForToday } from './scrollEdge.js';
import type { EdgeLoadCheckInput } from './scrollEdge.js';

const middleOfRangeInput: EdgeLoadCheckInput = {
	hasScrolledHorizontally: true,
	loading: false,
	pendingLoadDirection: null,
	columns: [{ index: 10 }, { index: 11 }, { index: 12 }],
	totalDates: 50,
	edgeThreshold: 5
};

function givenEdgeInput(overrides: Partial<EdgeLoadCheckInput>): EdgeLoadCheckInput {
	return { ...middleOfRangeInput, ...overrides };
}

describe('nextEdgeLoadDirection', () => {
	it('returns null before the operator has scrolled', () => {
		const input = givenEdgeInput({ hasScrolledHorizontally: false });

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBeNull();
	});

	it('returns null while a fetch is already loading', () => {
		const input = givenEdgeInput({ loading: true });

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBeNull();
	});

	it('returns null while a load direction is already pending', () => {
		const input = givenEdgeInput({ pendingLoadDirection: 'future' });

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBeNull();
	});

	it('returns null when there are no visible columns', () => {
		const input = givenEdgeInput({ columns: [] });

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBeNull();
	});

	it('returns null when the visible range is comfortably inside both edges', () => {
		const direction = nextEdgeLoadDirection(middleOfRangeInput);

		expect(direction).toBeNull();
	});

	it('returns "past" when the first visible column is within the threshold of index 0', () => {
		const input = givenEdgeInput({ columns: [{ index: 4 }, { index: 5 }, { index: 6 }] });

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBe('past');
	});

	it('returns "past" at the exact threshold boundary', () => {
		const input = givenEdgeInput({ columns: [{ index: 5 }] });

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBe('past');
	});

	it('does not load when unscrolled, even if the visible columns sit at the past edge', () => {
		const input = givenEdgeInput({
			hasScrolledHorizontally: false,
			columns: [{ index: 2 }, { index: 3 }]
		});

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBeNull();
	});

	it('does not load while a fetch is in flight, even if the visible columns sit at the past edge', () => {
		const input = givenEdgeInput({
			loading: true,
			columns: [{ index: 2 }, { index: 3 }]
		});

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBeNull();
	});

	it('does not load while a direction is already pending, even if the visible columns sit at the past edge', () => {
		const input = givenEdgeInput({
			pendingLoadDirection: 'future',
			columns: [{ index: 2 }, { index: 3 }]
		});

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBeNull();
	});

	it('returns "future" at the exact threshold boundary', () => {
		// totalDates=50 -> last index 49; threshold 5 -> boundary is index 44 exactly
		const input = givenEdgeInput({ columns: [{ index: 44 }] });

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBe('future');
	});

	it('returns "future" when the last visible column is within the threshold of the final date', () => {
		// totalDates=50 -> last index 49; threshold 5 -> triggers at index >= 44
		const input = givenEdgeInput({ columns: [{ index: 44 }, { index: 45 }] });

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBe('future');
	});

	it('prefers "past" when both edges are simultaneously within threshold (tiny loaded range)', () => {
		const input = givenEdgeInput({
			totalDates: 8,
			columns: [{ index: 2 }, { index: 3 }]
		});

		const direction = nextEdgeLoadDirection(input);

		expect(direction).toBe('past');
	});
});

describe('initialScrollLeftForToday', () => {
	const loadedDates = ['2026-08-01', '2026-08-02', '2026-08-03', '2026-08-04', '2026-08-05'];
	const cellWidth = 96;

	it('positions the viewport at today, leaving loaded days to its left as scroll headroom', () => {
		const scrollLeft = initialScrollLeftForToday(loadedDates, '2026-08-03', cellWidth);

		expect(scrollLeft).toBe(2 * cellWidth);
	});

	it('returns 0 when today is the first loaded date (no lookback buffer)', () => {
		const scrollLeft = initialScrollLeftForToday(loadedDates, '2026-08-01', cellWidth);

		expect(scrollLeft).toBe(0);
	});

	it('returns 0 when today is outside the loaded range', () => {
		const scrollLeft = initialScrollLeftForToday(loadedDates, '2026-09-01', cellWidth);

		expect(scrollLeft).toBe(0);
	});

	it('returns 0 for an empty date list', () => {
		const scrollLeft = initialScrollLeftForToday([], '2026-08-03', cellWidth);

		expect(scrollLeft).toBe(0);
	});
});
