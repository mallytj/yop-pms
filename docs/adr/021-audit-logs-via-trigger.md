# ADR 021: Audit Logs via Database Trigger

## Status

**Proposed**

## Context

The booking implementation plan proposed inserting audit log entries via
application code inside each `ExecuteTx` callback — an explicit
`INSERT INTO internal.outbox_events (type=audit_log, ...)` per mutation.
This approach has two problems:

1. **Fragile coverage.** Every mutation handler must remember to insert
   the audit row. If a future mutation (or a worker sweep, or an admin
   fix) forgets, the audit trail is incomplete with no compile-time or
   runtime warning.
2. **Wrong mechanism.** `internal.outbox_events` (ADR-012) is designed
   for async work delivery — emails, webhooks, provider void calls. It
   has retry backoff, dead-letter signalling, and a poll loop. None of
   that applies to an immutable audit record. Conflating audit with
   async delivery creates confusion about which events are durable and
   which are ephemeral delivery attempts.

Meanwhile, `auth.audit_logs` already exists (migration 00002) with the
exact schema needed — `action`, `entity`, `entity_id`, `changes JSONB`,
`user_id`, `property_id`, `created_at`. It was designed for triggers
from the start but sits unused.

## Decision

**Every INSERT, UPDATE, or DELETE on `operations.reservations` and
`operations.reservation_items` writes an audit row to `auth.audit_logs`
via a PostgreSQL trigger (`AFTER INSERT OR UPDATE OR DELETE FOR EACH ROW`).**

- The trigger function extracts `OLD`/`NEW` values, builds a `changes`
  JSONB diff, and inserts into `auth.audit_logs` with `entity='reservation'`
  or `entity='reservation_item'`.
- The calling context must set `app.current_user_id` before the mutation
  so the trigger can record who made the change. Application code does
  this via `qtx.SetCurrentUserID(ctx, userID)` inside `ExecuteTx`.
- `internal.outbox_events` remains for async delivery only (confirmation
  emails, cancellation notices, provider void calls) — per ADR-012.

This decision covers only `reservations` and `reservation_items`. Other
domains (folios, guests, room inventory) will adopt the same pattern
when their APIs are built.

## Consequences

### ✅ Positive

- **Guaranteed coverage.** The trigger fires for every mutation regardless
  of code path — application handler, worker sweep, migration, admin
  tool, direct SQL. Nothing bypasses it.
- **Separation of concerns.** `outbox_events` = async delivery.
  `audit_logs` = immutable record. Clear, distinct tables with clear,
  distinct purposes.
- **Schema reuse.** `auth.audit_logs` was built for this; no new table
  needed.
- **Zero application code per mutation.** Handlers don't need to
  construct audit rows. The trigger does it.

### ⚠️ Negative

- **User context coupling.** The trigger reads `app.current_user_id` from
  a session variable. If application code forgets to set it before a
  mutation, the audit row has `user_id=NULL`. Mitigation: `ExecuteTx`
  enforces this in a single place; all mutations go through it.
- **PL/pgSQL maintenance.** The trigger function is harder to unit-test
  than Go code. Mitigation: the trigger is simple (extract OLD/NEW,
  build JSONB diff, insert); integration tests verify it end-to-end.
- **Worker sweeps.** Background workers (hold expiry, archival) don't
  have a real user. They set `app.current_user_id` to a reserved
  `system` user UUID.
- **Changes JSONB semantics.** The trigger must decide what constitutes
  a meaningful change. Every column change is recorded; the diff format
  should be documented.

## Alternatives Considered

- **Application-code audit (Go handler inserts)**
  — Rejected: fragile, every handler must remember to do it. Workers
  and admin tools need separate audit paths. Easy to miss.

- **Outbox-as-audit (PLAN's original approach)**
  — Rejected: conflation of async delivery and immutable record. ADR-012
  explicitly scopes outbox to async work. Retry semantics don't apply
  to audit.

- **Event sourcing (full append-only log)**
  — Rejected: massive overshoot for a 5-property PMS. Re-evaluate if
  audit/replay needs grow significantly.

- **Separate audit service (microservice)**
  — Rejected: adds infra dependency. Trigger runs in the same transaction
  as the mutation — no partial audit.

## References

- Migration 00002 (`auth.audit_logs` schema, `auth.audit_log_action`,
  `auth.audit_log_entity` enums)
- ADR-012 (Transactional Outbox Worker — scoped to async delivery)
- ADR-015 (State Machine Rollup — trigger pattern already used for
  item→reservation status)
- `internal/booking/PLAN.md` Phase 0 (M18 migration entry)
- `internal/platform/db/tx.go` (`ExecuteTx` must call
  `qtx.SetCurrentUserID` before mutations)
