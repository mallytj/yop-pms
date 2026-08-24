import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createLoadGate } from './loadGate.js';

const COOLDOWN_MS = 2000;

function givenGate() {
	return createLoadGate(COOLDOWN_MS);
}

describe('createLoadGate', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('has no pending load before start() is called', () => {
		const gate = givenGate();

		expect(gate.pending).toBeNull();
	});

	it('start() claims the gate and records the requested direction', () => {
		const gate = givenGate();

		const claimed = gate.start('future');

		expect(claimed).toBe(true);
		expect(gate.pending).toBe('future');
	});

	it('start() refuses a second call while one is already pending', () => {
		const gate = givenGate();
		gate.start('future');

		const claimedSecond = gate.start('past');

		expect(claimedSecond).toBe(false);
		expect(gate.pending).toBe('future');
	});

	it('settle(false) keeps pending set until the cooldown fully elapses', () => {
		const gate = givenGate();
		gate.start('future');
		gate.settle(false);

		vi.advanceTimersByTime(COOLDOWN_MS - 1);

		expect(gate.pending).toBe('future');
	});

	it('settle(false) clears pending once the cooldown fully elapses', () => {
		const gate = givenGate();
		gate.start('future');
		gate.settle(false);

		vi.advanceTimersByTime(COOLDOWN_MS);

		expect(gate.pending).toBeNull();
	});

	it('allows a new load once the failure cooldown has elapsed', () => {
		const gate = givenGate();
		gate.start('past');
		gate.settle(false);
		vi.advanceTimersByTime(COOLDOWN_MS);

		const claimed = gate.start('future');

		expect(claimed).toBe(true);
		expect(gate.pending).toBe('future');
	});

	it('settle(true) keeps pending set until the cooldown fully elapses', () => {
		const gate = givenGate();
		gate.start('future');
		gate.settle(true);

		vi.advanceTimersByTime(COOLDOWN_MS - 1);

		expect(gate.pending).toBe('future');
	});

	it('settle(true) clears pending once the cooldown fully elapses', () => {
		const gate = givenGate();
		gate.start('future');
		gate.settle(true);

		vi.advanceTimersByTime(COOLDOWN_MS);

		expect(gate.pending).toBeNull();
	});

	it('allows a new load once the success cooldown has elapsed', () => {
		const gate = givenGate();
		gate.start('past');
		gate.settle(true);
		vi.advanceTimersByTime(COOLDOWN_MS);

		const claimed = gate.start('future');

		expect(claimed).toBe(true);
		expect(gate.pending).toBe('future');
	});
});
