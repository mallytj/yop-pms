# ADR 010: Guest-Aware Hold TTLs

## Status

**Accepted**

Hold-expiry worker applies tiered TTLs based on reservation source and guest presence: `website` source uses a short TTL (15-30 min) matching payment provider checkout windows; `internal` source with no attached guest uses a short grace (30-60 min); `internal` with an attached Guest ID uses a long grace (12-24 h) to support phone bookings and operational flexibility. TTL values come from `property_settings.{source}_hold_ttl_seconds`. ADR-004's hold-as-lock + auto-pin guarantees the worker only needs to release ledger rows.

Alternatives: single universal TTL (too risky for website, too short for staff), manual release only (permanent inventory blocks on forgotten drafts).

---

See: `docs/requirements/reservations.md` (R-RES-INTEG-007), `internal/booking/workers.go`
