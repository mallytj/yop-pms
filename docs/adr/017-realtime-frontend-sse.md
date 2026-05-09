# ADR 017: Real-Time Frontend Updates via SSE

## Status

**Proposed**

## Context

**What is the problem we are solving?**

Property staff routinely view the same reservation, calendar, or arrivals list at the same time on different machines. When one user mutates state (changes dates, assigns a room, takes payment, cancels), every other open UI must reflect that change without a manual refresh. Without push, stale UIs cause double-bookings, contradictory check-ins, and lost trust.

The backend already emits authoritative change events:

- ADR-010 — `LISTEN/NOTIFY` listener (`internal/platform/events`) consumes channels like `reservation_changes`, `availability_changes`, `outbox_dead_lettered` and drives reactive cache invalidation.
- Migration `00005_planner_notifier.sql` and `internal/booking/res_handler_example.go` `pg_notify('reservation_changes', …)` on every reservation/item/guest mutation. Payload already carries `property_id`, `record_id`, `operation`, stay range, source table.

So the change feed exists; what is missing is a transport that pushes it to browsers.

Constraints:

- Auth is cookie/JWT via existing Chi middleware — any new transport must reuse it, not invent a parallel auth handshake.
- We are multi-tenant — a client must only receive events for properties it can access.
- Backend is single-process today (one Go binary). Horizontal scaling is plausible but not imminent.
- We do not need client→server messaging on this channel — mutations continue to go through the REST/OpenAPI surface.
- Frontend is SvelteKit 5 with Runes; we want a Svelte-idiomatic store, not a global event bus.

## Decision

**We will push backend change events to the browser using Server-Sent Events (SSE) over plain HTTP, fanning out from the existing `events.Listener`.**

### Endpoint

`GET /api/v1/stream` — single SSE endpoint per authenticated session.

- Auth: existing session middleware (cookie/JWT). No subprotocol negotiation.
- Query params: `topics=reservations,availability` (subscription filter; default = all topics the user is permitted to see).
- Response: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`.
- Each event:
  ```
  id: <monotonic-ulid>
  event: reservation.changed
  data: {"property_id":"…","record_id":"…","op":"UPDATE","at":"…"}
  ```
- Heartbeat: `: ping\n\n` every 25 s to keep idle proxies open and let the client detect dead connections.
- Resume: client reconnects with `Last-Event-ID`; server replays from an in-memory ring buffer (last ~1000 events per topic, ~5 min retention). On overflow → emit `event: resync` so the client re-fetches via REST.

### Server architecture

```
PostgreSQL ──pg_notify──▶ events.Listener ──▶ Hub (in-process pub/sub)
                                                │
                                                ├─▶ SSE conn (user A, props {P1})
                                                ├─▶ SSE conn (user B, props {P1, P2})
                                                └─▶ SSE conn (user C, props {P2})
```

- New `internal/platform/realtime` package holds the `Hub`. The Hub is the single subscriber to relevant `events.Listener` channels and fans out to per-connection buffered channels.
- Per-connection authorization: on connect we resolve the user's permitted `property_id` set once; the Hub filters every event by that set before delivery.
- Slow consumers: bounded buffer (e.g. 64 events). Overflow drops the connection with `event: resync` so the client refetches state — never block the Hub.

### Payload policy — change notification, not entity snapshot

The SSE event carries **only** `{property_id, record_id, op, at}` — enough to identify what changed. The client fetches the new state via the existing REST endpoint (which is already cache-warm from ADR-010). Reasons:

- Avoids leaking fields the recipient is not authorized to see.
- Avoids divergence between SSE shape and REST shape (one source of truth: the OpenAPI contract).
- Keeps payloads tiny so the ring buffer stays cheap.

### Topics (initial)

| Topic           | Source channel        | Triggered by                                    |
| --------------- | --------------------- | ----------------------------------------------- |
| `reservations`  | `reservation_changes` | reservation/item/guest mutations (mig 00005)    |
| `availability`  | `availability_changes`| holds, ledger writes (ADR-013)                  |
| `outbox_alerts` | `outbox_dead_lettered`| ADR-012 dead-letter signal (admins only)        |

Topic names are stable strings; new ones added by extending the Hub's subscription list.

### Multi-instance scaling (deferred, not blocking)

Today: single Go process, in-process Hub. Acceptable.

When we scale out: each instance subscribes to the same PG channels independently — `LISTEN/NOTIFY` natively fans out to every backend listener, so no extra broker is needed for *event delivery*. The only cross-instance gap is the resume ring buffer (a client reconnecting to a different pod cannot replay). At that point we replace the in-memory ring with Redis Streams (`XADD`/`XREAD`) keyed per topic. This swap is local to the `realtime` package — no API change.

### Frontend (SvelteKit 5 + Runes)

A single connection per tab, exposed as a Rune-based store and per-resource subscriptions.

```ts
// web/src/lib/realtime/stream.svelte.ts
import { browser } from '$app/environment';

type ChangeEvent = {
  property_id: string;
  record_id: string;
  op: 'INSERT' | 'UPDATE' | 'DELETE';
  at: string;
};

class RealtimeStream {
  status = $state<'idle' | 'open' | 'reconnecting' | 'closed'>('idle');
  #es: EventSource | null = null;
  #subs = new Map<string, Set<(e: ChangeEvent) => void>>();

  connect() {
    if (!browser || this.#es) return;
    this.#es = new EventSource('/api/v1/stream', { withCredentials: true });
    this.#es.onopen = () => (this.status = 'open');
    this.#es.onerror = () => (this.status = 'reconnecting'); // browser auto-retries
    this.#es.addEventListener('resync', () => {
      // notify all subs to refetch
      for (const [, fns] of this.#subs) fns.forEach((fn) => fn({ op: 'UPDATE' } as ChangeEvent));
    });
    this.#es.addEventListener('reservation.changed', (ev) => this.#dispatch('reservations', ev));
    this.#es.addEventListener('availability.changed', (ev) => this.#dispatch('availability', ev));
  }

  on(topic: string, fn: (e: ChangeEvent) => void) {
    if (!this.#subs.has(topic)) this.#subs.set(topic, new Set());
    this.#subs.get(topic)!.add(fn);
    return () => this.#subs.get(topic)!.delete(fn);
  }

  #dispatch(topic: string, ev: MessageEvent) {
    const data = JSON.parse(ev.data) as ChangeEvent;
    this.#subs.get(topic)?.forEach((fn) => fn(data));
  }

  disconnect() {
    this.#es?.close();
    this.#es = null;
    this.status = 'closed';
  }
}

export const realtime = new RealtimeStream();
```

Usage in a reservation page:

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { realtime } from '$lib/realtime/stream.svelte';
  import { fetchReservation } from '$lib/api/reservations';

  let { id }: { id: string } = $props();
  let reservation = $state(await fetchReservation(id));

  onMount(() => {
    realtime.connect();
    return realtime.on('reservations', async (e) => {
      if (e.record_id === id) reservation = await fetchReservation(id);
    });
  });
</script>
```

Single connection is initialized in the root layout (`+layout.svelte`); pages register topic subscriptions on mount and clean up on destroy. Browsers automatically reconnect with `Last-Event-ID`; on `resync` the local store refetches via REST.

## Consequences

### ✅ Positive

- **Reuses existing infra** — no new broker, no new auth path; drops in behind Chi and reuses `events.Listener` + reactive cache.
- **One source of truth** — clients refetch via REST, so SSE never drifts from the OpenAPI contract.
- **Tenant safe** — per-connection property filter enforced server-side; clients cannot subscribe to data they cannot read.
- **Cheap at current scale** — one HTTP/2 connection per tab, one goroutine per connection, payloads in single-digit bytes.
- **Browser-native resume** — `Last-Event-ID` reconnect with no client library.
- **Forward-compatible** — Redis Streams swap is internal to the `realtime` package when we shard.

### ⚠️ Negative

- **HTTP/1.1 connection cap** — six SSE tabs per origin can starve other requests on legacy HTTP/1.1; we mitigate by serving over HTTP/2 in production (already the case behind our reverse proxy).
- **Goroutine + buffer per client** — bounded but non-zero; needs a metric (open connections, dropped slow consumers) and a connection ceiling.
- **No client→server messaging on this channel** — collaborative cursor/typing presence will need a separate WebSocket later. Acceptable trade-off.
- **In-memory resume buffer is per-process** — a reconnect that lands on a different pod after we scale out will trigger a `resync` until the Redis Streams swap.
- **Browser `EventSource` lacks header customization** — auth must be cookie-based or via query token; we already use cookies, so no change.

## Alternatives Considered

- **WebSockets** — Bidirectional and lower per-message overhead, but we have nothing to send client→server on this channel and it forces a second auth path, a subprotocol, and a heavier client. Hold for future presence/collab features.
- **Long polling** — Works through any proxy but burns a request per cycle and has worse latency. SSE strictly dominates for one-way push.
- **Polling with ETags** — Simple but every open tab hits the API on a fixed interval regardless of change rate; wastes capacity at peak and still feels stale at low rates.
- **Direct `pg_notify` over Postgres LISTEN from the browser** — Not possible; browsers cannot speak the Postgres protocol, and exposing it would shred our auth model.
- **Push via Redis pub/sub from the browser via a proxy** — Adds infra without improving on SSE-over-HTTP for the one-way case.

## References

- [ADR-010 — Reactive Cache Invalidation](010-reactive-cache-invalidation.md)
- [ADR-012 — Transactional Outbox Worker](012-transactional-outbox-worker.md)
- [ADR-013 — Locking & Availability Strategy](013-locking-availability-strategy.md)
- `internal/platform/events/listener.go` — PG LISTEN reconnect loop
- `migrations/00005_planner_notifier.sql` — `reservation_changes` channel + payload shape
- [MDN — Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
- [WHATWG HTML — EventSource](https://html.spec.whatwg.org/multipage/server-sent-events.html)
