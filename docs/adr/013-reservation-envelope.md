# ADR 013: Reservation Envelope

## Status

**Accepted**

`operations.reservations.stay_period_envelope TSTZRANGE NOT NULL` is the materialised union of its non-cancelled items' periods, computed as `[min(item.lower), max(item.upper))`. Recomputed in the same transaction as any item insert/update/delete that affects bounds; `version` column is bumped. GIST index on `(property_id, stay_period_envelope)` supports overlap filters; btree on `(property_id, lower(stay_period_envelope), id)` covers the default arrival-date cursor (ADR-008). `R-RES-VALID-002` (no earlier-than-today) is enforced on `lower(envelope)`.

Alternatives: pure JOIN-aggregate derivation (every list query expensive, awkward cursor keys), `GENERATED ALWAYS AS` column (cannot reference child tables), client-side join (pushes complexity to every consumer).

---

See: ADR-008 (cursor pagination), ADR-009 (state machine rollup), `docs/requirements/reservations.md` §5 (R-RES-VALID-001/002), §9 (list)
