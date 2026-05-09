# ADR 013: Locking and Availability Strategy

## Status

**Accepted**

## Context

The reservation domain has a hard correctness requirement: no room may be sold to two parties for overlapping dates. This must hold under arbitrary concurrency (web booking engine, staff drag-to-create, OTA webhook, reactivation of a cancelled booking) and arbitrary failure modes (crashed handlers, abandoned web checkouts, network retries).

Three sub-problems sit underneath:

1. **Per-room overlap** — two reservations cannot pin the same physical room on overlapping dates.
2. **Per-room-type availability** — a request for a room type at a date when no rooms of that type remain must be rejected.
3. **In-flight protection** — while a guest is filling in the booking form (or a staff member is dragging across a calendar), the inventory must not be claimed by someone else.

The naive design is: lock a room when reservation creation starts, release on success/fail/timeout, separate from any reservation state. This produces two parallel mechanisms (reservation state machine + room lock state) that drift, deadlock, and require their own crash-recovery story.

## Decision

The reservation `hold` state itself **is** the lock. There is no separate "room lock" entity. Three rules together provide the correctness guarantee.

### 1. The ledger is the single source of truth for room availability

`inventory.room_inventory_ledger` has `UNIQUE(room_id, calendar_date)`. Every fact that blocks a room on a date — confirmed booking, hold, online checkout in progress, maintenance block, decommissioning — writes a row to this table. Availability queries read this single table. There is no separate "current bookings" view that has to be reconciled with maintenance.

Status enum:

- `sold` — committed booking, ledger row tied to a reservation
- `on_hold` — guest checkout flow in progress, ledger row tied to a `checkout_session` with `expires_at`
- `maintenance` — blocked by an `inventory.maintenance_blocks` row (ADR-013 + M3)
- `decommissioned` — room permanently removed from inventory
- `available` — implicit (absence of row)

### 2. Reservation creation auto-pins a specific room at hold time

When a hold reservation is created with only a room type (no specific room chosen — typical for guest web booking), the system selects an available room of that type using a deterministic policy (`SELECT id FROM rooms ... FOR UPDATE SKIP LOCKED` ordered by room number) and writes ledger rows for each calendar date in `stay_period`.

This allows concurrent booking of the same room type. If two guests book the last two rooms, Guest A locks room 101, and Guest B skips 101 to lock 102. Both succeed atomically.

The `reservation_items.assigned_room_id` column stays NULL until staff explicitly confirms the assignment. The ledger row is the implementation detail; the assignment column is the guest-visible commitment. The ledger row may be moved to a different room of the same type without changing `assigned_room_id` — useful when housekeeping shuffles the floor plan.

Auto-pinning collapses the type-level race ("which of N rooms in this type") into a per-room race, which the DB can resolve atomically.

### 3. The DB enforces per-room overlap at insert time

Two constraints together provide the guarantee:

- `EXCLUDE USING GIST (assigned_room_id WITH =, stay_period WITH &&) WHERE (deleted_at IS NULL AND assigned_room_id IS NOT NULL)` on `operations.reservation_items` — prevents two items pinning the same room with overlapping ranges (post-confirmation case).
- `UNIQUE(room_id, calendar_date)` on `inventory.room_inventory_ledger` — prevents two ledger rows existing for the same room and date (the dominant pre-confirmation case under the hold model).

Both checks fire at insert time; the loser of a concurrent attempt receives a constraint violation that the handler maps to 409 Conflict with the conflicting dates listed.

### 4. Per-room-type capacity is enforced at the application layer

Per-room-type overflow ("I want a Double for next weekend, are any left?") cannot be expressed as a single DB constraint without locking the entire type. Instead, availability is computed as:

```
available_count(type, date) =
    count(rooms WHERE type=...) - count(ledger rows WHERE room.type=... AND calendar_date=date AND status in ('sold','on_hold','maintenance','decommissioned'))
```

The handler checks this aggregate before attempting an insert. Because the auto-pin (rule 2) immediately creates the ledger row, the window between the check and the insert is small, and any actual race is caught by the DB constraint (rule 3). The aggregate check is best-effort — it gives early rejection with a useful error message; the DB has the final word.

### 5. Crash recovery via TTL worker

Holds expire via the worker described in `R-RES-INTEG-007`. Per-source TTL is configurable (`website_hold_ttl_seconds`, `internal_hold_ttl_seconds`). Stale holds are cancelled by the worker, ledger rows deleted, NOTIFY emitted. Recovery is automatic; no separate dead-lock detection logic is needed.

## Consequences

### ✅ Positive

- **Single mechanism** — One state machine governs a reservation's lifecycle. No separate room-lock state to keep in sync.
- **DB-enforced correctness** — Per-room overlap is a constraint violation, not application logic. Cannot be bypassed by a buggy handler.
- **Maintenance integrates cleanly** — Blocking a room for repair is the same primitive as selling it: a ledger row.
- **Crash recovery is the same problem as expiry** — A worker process exists anyway to release abandoned web checkouts; the same loop covers crashed staff sessions.
- **No advisory locks** — The DB's existing transactional guarantees suffice; no `pg_advisory_lock`, no Redis `SETNX`-style serialisation.

### ⚠️ Negative

- **Auto-pin requires room selection policy** — "Lowest room number" is deterministic but suboptimal in some scenarios (e.g. clustering bookings to free up a maintenance-friendly block of rooms). Policy is pluggable but the default may need tuning.
- **Type-level capacity check has a small race window** — Resolved at insert via the DB, but the early rejection at the type level is not authoritative. A request that passes the aggregate check may still fail at insert under heavy concurrency. Acceptable: 409 with retry guidance.
- **Reactivation re-checks availability** — Cancelled reservations release ledger rows; reactivating a cancellation requires re-acquiring them. May fail if the dates were rebooked. Surfaced as 409.
- **Expired hold deletes ledger rows** — A worker that prematurely expires a still-paying website checkout (clock skew, slow payment provider) silently releases inventory. Mitigated by setting `website_hold_ttl_seconds` generously vs. payment provider timeouts.

## Alternatives Considered

- **Separate "room lock" table with TTL** — Rejected. Two parallel state machines for a single physical fact (this room is committed for these dates). Drift between the lock table and reservation table is a constant source of bugs.
- **`SELECT ... FOR UPDATE` on rooms during booking** — Rejected. Serialises every concurrent booking attempt across the property even when they are for different rooms. Throughput collapses on a busy day.
- **Redis distributed lock per room/date** — Rejected. Redis is transient state only (ADR-008); a Redis-acquired lock that cannot be observed by the DB violates the single-source-of-truth principle. Any availability query reading the DB would see stale state.
- **Eventual consistency, accept double bookings, resolve manually** — Rejected. Hospitality industry tolerates 0% double-booking; the financial and reputational cost of a single incident exceeds any latency saving.
- **Optimistic check-then-insert without DB-level constraint** — Rejected. Race window is fundamentally unbounded; only DB constraints are reliable.

## References

- `migrations/00004_reservations_finance.sql` — Existing EXCLUDE constraint on `reservation_items` and `room_inventory_ledger` UNIQUE constraint
- `migrations/00007_reservation_api_prep.sql` — M3 adds `maintenance` ledger status + maintenance_block_id FK
- `docs/requirements/reservations.md` — R-RES-AVAIL-001 through R-RES-AVAIL-012, R-RES-INTEG-007
- ADR-008: Redis Caching Layer (why Redis is not used for locking)
- ADR-012: Transactional Outbox Worker (companion crash-recovery primitive)
