# ADR 007: Locking & Availability

## Status

**Accepted**

The reservation `hold` state is the lock — no separate room-lock entity. Three rules provide correctness: (1) `inventory.room_inventory_ledger` is the single source of truth for room availability (`UNIQUE(room_id, calendar_date)`), with `sold`/`on_hold`/`maintenance`/`decommissioned`/implicit-`available`; (2) hold creation auto-pins the lowest available room of the requested type via `SELECT ... FOR UPDATE SKIP LOCKED` and writes ledger rows for each `stay_period` date; (3) `EXCLUDE USING GIST (assigned_room_id WITH =, stay_period WITH &&) WHERE (deleted_at IS NULL AND assigned_room_id IS NOT NULL)` on `reservation_items` catches overlap at insert time. Per-room-type capacity is best-effort at the app layer; the DB has the final word.

Alternatives: separate room-lock table (drift), `SELECT ... FOR UPDATE` on rooms (serialises), Redis distributed lock (transient), eventual consistency (zero double-book tolerance), app-only check-then-insert (unbounded race).

---

See: `migrations/00004_reservations_finance.sql`, `migrations/00007_reservation_api_prep.sql`, `docs/requirements/reservations.md` (R-RES-AVAIL-001..012, R-RES-INTEG-007)
