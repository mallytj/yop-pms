# ADR 014: Audit Logs via Database Trigger

## Status

**Accepted**

Every `INSERT`/`UPDATE`/`DELETE` on `operations.reservations` and `operations.reservation_items` writes an audit row to `auth.audit_logs` via an `AFTER ... FOR EACH ROW` trigger. The trigger builds a `changes JSONB` diff and uses `app.current_user_id` (set by `qtx.SetCurrentUserID` inside `ExecuteTx`) for the `user_id` column. System workers (hold expiry, archival) set it to a reserved `system` UUID. Coverage is bypass-proof: application handler, worker sweep, migration, admin tool, direct SQL — all fire the trigger. `internal.outbox_events` (ADR-006) is async delivery only.

Alternatives: app-code audit (fragile, every handler must remember), outbox-as-audit (conflates delivery with record), event sourcing (overshoot), separate audit microservice (infra + partial-audit window).

---

See: `migrations/00002_*` (auth.audit_logs schema), `internal/platform/db/tx.go`, `docs/requirements/reservations.md`
