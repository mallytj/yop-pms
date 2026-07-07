# ADR 017: Real-Time Frontend Updates via SSE

## Status

**Accepted**

## Context

Backend emits change events via PostgreSQL `LISTEN/NOTIFY` (`reservation_changes`,
`staff_alerts`), consumed by `internal/platform/events/listener.go` with auto-
reconnect. What's missing: a transport that pushes these events to open browser
tabs, without polling, and a reliable NOTIFY source that never misses a mutation.

## Decision

**Two-part system:**

1. **PostgreSQL triggers** fire `pg_notify` on every `INSERT`/`UPDATE`/`DELETE`
   to `operations.reservations`, `operations.reservation_items`,
   `inventory.room_inventory_ledger`, and `pricing.booked_daily_rates`. Payload
   is minimal: `{table, op, id, property_id}`. Triggers cannot be bypassed by
   workers, admin tools, or direct SQL — replaces all Go-sent `NOTIFY` calls.

2. **`internal/platform/realtime` Hub** registers as a handler on the existing
   `events.Listener` (no own pgx connection). Simple mutex guards the client map.
   Subscribe, OnEvent, Resync — no select loop.

   - Hub receives structured `events.Event` from Listener, fans out to per-
     connection buffered channels filtered by `property_id`.
   - Bounded buffer (64 events). Overflow → `event: resync` instead of blocking.
   - `Hub.Resync()` sends `event: resync` to all clients — called from
     Listener's `onReconnect` callback after every reconnect to cover the
     disconnect gap.
   - Heartbeat `: ping\n\n` every 25s via ticker inside `Subscribe`.
   - Endpoint: `GET /v1/sse` (inside `/v1`; idempotency middleware skips GET).
   - Auth: `EventSource` cannot set custom HTTP headers. StubAuth falls back
     to `?property_id=` query param — validates resolved property against
     the authenticated session. Guest UUIDs rejected. Real auth PR switches
     to cookie auth (`EventSource` sends cookies with `withCredentials: true`).

   Frontend: SvelteKit opens `EventSource('/v1/sse?property_id=...')` in root
   layout. Per-resource subscriptions refetch on change. On `event: resync`
   refetch via REST. Browser native auto-reconnect.

Alternatives considered: Go-sent NOTIFY (fragile, easy to miss mutations),
WebSockets (bidirectional overhead — no client→server messages needed here),
long polling (worse latency), polling with ETags (still stale).

---

See: `internal/platform/events/listener.go`, `internal/platform/realtime/hub.go`,
`migrations/NNN-sse-notify-triggers.sql`
