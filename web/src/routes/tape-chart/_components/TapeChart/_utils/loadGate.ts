export type LoadDirection = 'past' | 'future';

export interface LoadGate {
	readonly pending: LoadDirection | null;
	/** Claims the gate for `direction`; returns false if a load is already pending. */
	start(direction: LoadDirection): boolean;
	/**
	 * Call once the load settles. Always releases after a cooldown rather
	 * than immediately: an operator holding the scrollbar thumb at the edge
	 * keeps re-satisfying the edge check on every render, so even a
	 * successful load must wait out the cooldown — otherwise it refires the
	 * next batch instantly and the operator gets a burst of loads instead of
	 * one. `succeeded` is unused today but kept so callers can still
	 * distinguish outcomes if the cooldown ever needs to differ per case.
	 */
	settle(succeeded: boolean): void;
}

export function createLoadGate(
	cooldownMs: number,
	scheduleAfterCooldown: (fn: () => void, ms: number) => void = setTimeout
): LoadGate {
	let pending: LoadDirection | null = null;
	return {
		get pending() {
			return pending;
		},
		start(direction) {
			if (pending) return false;
			pending = direction;
			return true;
		},
		settle() {
			scheduleAfterCooldown(() => {
				pending = null;
			}, cooldownMs);
		}
	};
}
