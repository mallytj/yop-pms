# ADR 010: Reactive Cache Invalidation

## Status

**Accepted**

## Context

TTL-only caching is insufficient for a PMS. Availability and pricing data must reflect reality immediately after a mutation — a guest booking a room should remove that availability from the cache now, not when a 1-hour TTL expires. Serving stale availability risks showing guests rooms that are already taken.

The system already has PostgreSQL `LISTEN/NOTIFY` triggers in place (`migrations/00005_planner_notifier.sql`), originally built for the tape chart planner. These triggers fire on every meaningful state change across reservations, guests, and daily rates — exactly the events that should drive cache invalidation.

TTL alone also creates unnecessary database pressure: short TTLs mean frequent expiry and re-fetching of data that hasn't changed.

## Decision

We use **PostgreSQL `LISTEN/NOTIFY` as the primary cache invalidation mechanism**. TTLs become a safety net only.

### How it works

1. **Database triggers** fire `pg_notify` on INSERT, UPDATE, and DELETE across reservations, guests, and daily rates. Each notification carries a typed payload: the operation, property, record ID, and the affected date range.

2. **A dedicated listener connection** (outside the pool — `LISTEN` blocks a connection) subscribes to notification channels and dispatches events to registered handlers. Handlers are decoupled from the listener: they receive a parsed event and decide how to act on it.

3. **Handlers perform surgical invalidation** using the payload's property ID and date range to target only the affected cache keys. Different cache namespaces may use different invalidation strategies — for example, per-day keys (availability) can be invalidated exactly, while range-keyed entries (e.g. planner views) require an overlap check to find all affected keys.

4. **TTLs are long** (24h+) since they are a fallback, not the clock driving freshness.

5. **Reconnect flush** — on listener reconnect, flush the entire cache namespace. During a disconnect window any number of mutations could have been missed; a full flush is the safest recovery. Cache misses are acceptable — load times without cache are fast enough that a temporary flush causes no user-visible degradation.

## Consequences

### ✅ Positive

- **Immediate consistency** — Cache reflects database state within milliseconds of a mutation
- **Long TTLs** — Reduced database pressure; cache entries live until something actually changes
- **Surgical invalidation** — Payload includes property_id + date range; only affected keys are cleared
- **Reuses existing infrastructure** — Triggers already exist; no new migrations needed
- **Decoupled** — Events package has no knowledge of cache; cache handler has no knowledge of how notifications are sent
- **Reconnect safety** — Full cache flush on reconnect ensures no stale data survives a disconnect gap

### ⚠️ Negative

- **Dedicated connection** — One persistent `*pgx.Conn` outside the pool; another thing to manage and reconnect
- **Missed notifications** — If listener is down, mutations during that window are invisible to the listener; mitigated by full flush on reconnect
- **At-least-once, not exactly-once** — `pg_notify` is fire-and-forget; a notification can be missed but never duplicated. Idempotent invalidation (`Invalidate` is safe to call multiple times) means this is acceptable
- **Reconnect thundering herd** — Full cache flush causes a brief spike of DB reads as cache warms back up; acceptable given low base load
- **Payload size limit** — `pg_notify` payloads are capped at 8000 bytes; current payload is well within limits

## Alternatives Considered

- **Short TTLs only (e.g., 30s)** — Rejected because:
  - Still serves stale data within the TTL window
  - High database pressure from constant expiry and re-fetching
  - No relationship between data changing and cache clearing

- **Invalidate in service layer only** — Rejected as sole mechanism because:
  - Relies on developer discipline; easy to miss a code path
  - Doesn't handle mutations from other sources (admin tools, migrations, direct DB access)
  - LISTEN/NOTIFY catches all mutations regardless of origin

- **Redis keyspace notifications** — Rejected because:
  - Notifies on cache events (expiry, eviction), not on data mutations
  - Source of truth is PostgreSQL, not Redis

- **Polling the database for changes** — Rejected because:
  - Constant query overhead regardless of whether anything changed
  - Higher latency than NOTIFY (poll interval vs immediate)

## References

- `internal/platform/events/` — Listener implementation
- `internal/platform/cache/` — Cache client and invalidation handlers
- `migrations/00005_planner_notifier.sql` — Existing trigger definitions
- ADR-008: Redis Caching Layer (cache client design)
- [PostgreSQL LISTEN/NOTIFY docs](https://www.postgresql.org/docs/current/sql-listen.html)
