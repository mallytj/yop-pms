# ADR 019: Reservation `stay_period_envelope`

## Status

**Proposed**

## Context

R-RES-VALID-001 permits multi-item reservations whose items have differing
`stay_period`s. This makes the reservation itself a parent of N independently
dated children with no canonical "stay" of its own. Two operational needs
collide with that:

- Receptionist dashboards / list filters operate on the reservation level
  ("arriving today", "in-house", "departing tomorrow") — they need a single
  date range to filter on.
- Cursor pagination (ADR-014) sorted by arrival date needs an indexable key
  on the reservation row, not a join into items.

Without a reservation-level period, every list query joins `reservation_items`
and aggregates — slow, complicates cursor encoding, prevents clean GIST
indexing.

## Decision

`operations.reservations` carries a materialised `stay_period_envelope
TSTZRANGE NOT NULL`, computed as

```
[ min(item.lower) , max(item.upper) )
```

across all non-cancelled items. The envelope is recomputed within the same
transaction as any item insert / update / delete that affects bounds, and
the reservation `version` is bumped.

A GIST index on `(property_id, stay_period_envelope)` supports overlap
filters; a btree on `(property_id, lower(stay_period_envelope), id)` covers
the default arrival-date cursor (ADR-014).

R-RES-VALID-002 ("`lower(stay_period)` not earlier than today") is enforced
on `lower(envelope)` for create / extend operations — single check, single
authorization gate.

## Consequences

### ✅ Positive

- O(1) list / cursor queries against a reservation-level index.
- Single source of truth for "Mrs Smith's stay" on dashboards.
- Validation rules collapse to one expression per reservation, not per item.
- Cancellation of an item (item-level cancel) automatically tightens the
  envelope on the next recompute.

### ⚠️ Negative

- Redundant data — envelope must be kept in sync with items. Bug surface.
- Trigger or service-layer code on every item mutation.
- Migration backfill required for existing rows.

## Alternatives

- **Pure derivation via JOIN aggregate** — rejected: every list query
  expensive; cursor pagination keys awkward.
- **Postgres `GENERATED ALWAYS AS` column** — rejected: cannot reference child
  tables; would need triggers anyway.
- **No reservation-level period; UI joins items client-side** — rejected:
  pushes complexity to every consumer.

## References

- ADR-014 — Cursor pagination
- ADR-015 — State machine rollup
- `/docs/requirements/reservations.md` §5 (R-RES-VALID-001, R-RES-VALID-002), §9 (list)
- `/docs/CONTEXT.md` — `stay_period`
