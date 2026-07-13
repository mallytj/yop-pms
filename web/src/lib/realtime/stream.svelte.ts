/**
 * Realtime SSE stream — Svelte 5 Rune-based EventSource wrapper.
 *
 * Core Requirements: [R-RES-INTEG-003], [ADR-011]
 *
 * Usage:
 *   // In root +layout.svelte (once per tab)
 *   import { realtime } from '$lib/realtime/stream.svelte';
 *   onMount(() => {
 *     realtime.connect('a0eebc99-...');
 *     return () => realtime.disconnect();
 *   });
 *
 *   // In any component — skip fetch if not in view, skip if version matches
 *   onMount(() => {
 *     return realtime.on('reservations', async (e) => {
 *       const local = findReservation(e.record_id);
 *       if (!local) return;                 // not in current view — skip
 *       if (local.version === e.version) return; // already fresh — skip
 *       reservation = await fetchReservation(e.record_id);
 *     });
 *   });
 */

import { browser } from '$app/environment';

export type ChangeOp = 'INSERT' | 'UPDATE' | 'DELETE';

export type ChangeEvent = {
	property_id: string;
	record_id: string;
	op: ChangeOp;
	at: string;
	version?: number; // present on reservations + reservation_items
	resync?: boolean; // true when server signals full resync
};

export type Topic = 'reservations' | 'availability' | 'rates' | 'staff';

// Subscriber pattern: check the entity is loaded in current view before fetching.
// If findReservation(e.record_id) returns null, the record isn't in view — skip.
// This avoids over-fetching for reservations not shown on screen. No separate
// Set or special payload needed — the loaded data IS the filter. New records
// outside current view appear on next navigation, filter change, or resync.
/** Base URL for the SSE endpoint. Override in dev with VITE_API_URL. */
const SSE_BASE_URL = import.meta.env.VITE_API_URL ?? '';

class RealtimeStream {
	/** Connection status */
	status = $state<'idle' | 'open' | 'reconnecting' | 'closed'>('idle');

	/** Active EventSource instance */
	#es: EventSource | null = null;

	/** Map of topic → Set of subscriber callbacks */
	#subs = new Map<Topic, Set<(e: ChangeEvent) => void>>();

	/** Connected property ID */
	#propertyId = '';

	/**
	 * Open the SSE connection. Must be called once per tab (typically in
	 * +layout.svelte). If already connected, this is a no-op.
	 */
	connect(propertyId: string) {
		if (!browser || this.#es) return;
		this.#propertyId = propertyId;

		const url = `${SSE_BASE_URL}/v1/sse?property_id=${encodeURIComponent(propertyId)}`;
		this.#es = new EventSource(url, { withCredentials: true });

		this.#es.onopen = () => {
			this.status = 'open';
		};

		this.#es.onerror = () => {
			if (this.#es?.readyState === EventSource.CLOSED) {
				this.status = 'closed';
				this.#es = null;
			} else {
				this.status = 'reconnecting';
			}
		};

		this.#es.addEventListener('resync', () => {
			// Notify all topic subscribers to refetch.
			for (const [, fns] of this.#subs) {
				fns.forEach((fn) =>
					fn({
						property_id: this.#propertyId,
						record_id: '',
						op: 'UPDATE',
						at: new Date().toISOString(),
						resync: true
					})
				);
			}
		});

		// Generic change event — all tables map here.

		// Map server event names to internal topics.
		const eventMap: Record<string, Topic> = {
			'reservation.changed': 'reservations',
			'availability.changed': 'availability',
			'rate.changed': 'rates',
			'staff.alert': 'staff'
		};

		for (const [eventName, topic] of Object.entries(eventMap)) {
			this.#es.addEventListener(eventName, (ev: MessageEvent) => {
				let data: ChangeEvent;
				try {
					data = JSON.parse(ev.data);
				} catch (err) {
					console.error('[realtime] invalid json:', ev.data, err);
					return;
				}
				this.#dispatch(topic, data);
			});
		}
	}

	/**
	 * Subscribe to a topic. Returns an unsubscribe function.
	 * Safe to call before connect() — subscriptions are buffered.
	 */
	on(topic: Topic, fn: (e: ChangeEvent) => void): () => void {
		if (!this.#subs.has(topic)) {
			this.#subs.set(topic, new Set());
		}
		this.#subs.get(topic)!.add(fn);

		return () => {
			this.#subs.get(topic)?.delete(fn);
		};
	}

	/** Close the SSE connection. */
	disconnect() {
		this.#es?.close();
		this.#es = null;
		this.status = 'closed';
	}

	#dispatch(topic: Topic, data: ChangeEvent) {
		this.#subs.get(topic)?.forEach((fn) => {
			try {
				fn(data);
			} catch (err) {
				console.error('[realtime] subscriber error:', err);
			}
		});
	}
}

/** Singleton instance — one connection per browser tab. */
export const realtime = new RealtimeStream();
