# ADR 011: Real-Time Frontend via SSE

## Status

**Accepted**

Two-part push system: (1) PostgreSQL triggers on `operations.reservations`, `operations.reservation_items`, `inventory.room_inventory_ledger`, and `pricing.booked_daily_rates` fire `pg_notify` with minimal `{table, op, id, property_id}` payloads on every mutation — bypass-proof. (2) `internal/platform/realtime` Hub subscribes to the existing `events.Listener`, fans out to per-connection buffered channels filtered by `property_id`, emits `event: resync` on overflow or post-reconnect, and heartbeats `: ping` every 25s. Endpoint `GET /v1/sse`; auth via `StubAuth` query-param fallback until cookie auth lands.

Alternatives: Go-sent `NOTIFY` (bypassable, fragile), WebSockets (bidirectional overhead unused), long polling (latency), polling with ETags (still stale).

---

See: `internal/platform/events/listener.go`, `internal/platform/realtime/hub.go`, `web/src/lib/realtime/stream.svelte.ts`
