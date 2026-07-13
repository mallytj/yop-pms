# ADR 009: State Machine Rollup

## Status

**Accepted**

Reservation status is derived from its items by a deterministic PL/pgSQL function called from both the application layer and an `AFTER UPDATE OF status` safety-net trigger. Rule (first match wins): all items in `cancelled`-equivalent → `cancelled`; all terminal and ≥1 `checked_out` → `checked_out`; ≥1 `checked_in` and no items in `booked` → `checked_in`; else unchanged. Terminal item states: `checked_out`, `no_show`, `cancelled`, `archived`. The `hold`→`confirmed` transition is reservation-level, not item-driven, and does not roll up.

Alternatives: reservation-only status (loses per-item outcomes), two independent state machines (semantic ambiguity), materialised-view rollup (stale window), generated column (cannot reference child tables), event sourcing (overshoot).

---

See: `docs/requirements/reservations.md` §7.3, `migrations/00001_initial_schemas_enums_functions_extensions.sql`, `internal/booking/state_machine.go`
