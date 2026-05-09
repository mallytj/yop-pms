# ADR 017: `stay_period` Time Semantics

## Status

**Proposed**

## Context

Reservation `stay_period` was originally specified as a `TSTZRANGE` with bounds
implicitly pinned to 00:00 in the property timezone — effectively a `daterange`
in TSTZ clothing. This created several frictions:

- The `housekeeping_buffer_minutes` property setting had no clean enforcement
  point: the ledger keys on `calendar_date`, so sub-day buffers had no place
  to live.
- Same-day turnover (guest A departs, guest B arrives same date) collapsed at
  the EXCLUDE-GIST level despite being legitimate operational reality.
- Edge cases R-RES-EDGE-017 (adjacent ranges) and R-RES-AVAIL-005 (housekeeping
  buffer) needed bespoke logic that didn't fit the data model.

A receptionist scenario crystallised the problem: guest A checks out 11:00,
housekeeping cleans, guest B arrives 15:00 — same date. Midnight-bound
TSTZRANGE either rejected this (overlap) or silently allowed it without
encoding the buffer.

## Decision

`reservation_item.stay_period` bounds carry **property check-in / check-out
timestamps**, not midnight:

- `lower(stay_period) = arrival_date @ property.default_checkin_time`
- `upper(stay_period) = departure_date @ property.default_checkout_time`

API request bodies accept `arrival_date` and `departure_date` as DATEs; the
server composes the TSTZRANGE using property defaults. Explicit timestamps
(early arrival / late checkout) are accepted only with the
`reservations:override_restrictions` permission.

`housekeeping_buffer_minutes` becomes **advisory only** — surfaced to staff
for scheduling context, not enforced at availability check or written as a
ledger row. Real same-day turnover protection emerges naturally from the gap
between `default_checkout_time` and `default_checkin_time`.

Ledger rows continue to key on `calendar_date`. The included date set is
`{ d : d ∈ [lower::date, upper::date) }` — departure date excluded (room is
sellable that night).

## Consequences

### ✅ Positive

- Same-day turnover encodes naturally; GIST EXCLUDE on `assigned_room_id` does
  the right thing without bespoke logic.
- One overlap mechanism (TSTZRANGE) handles every case — adjacent stays,
  overstays, late checkouts, early arrivals.
- Overstay detection is a clean `now() > upper(stay_period) + grace`
  comparison — no calendar-date trickery.
- API surface stays date-friendly for the common case.

### ⚠️ Negative

- Migration rewrites every existing `stay_period` value to use the new
  bounds.
- Documentation (flows, edge cases) must be updated to drop midnight
  assumptions.
- `housekeeping_buffer_minutes` enforcement (R-RES-AVAIL-005) is dropped —
  some integrators may have expected hard enforcement.

## Alternatives

- **Keep midnight bounds, add a `housekeeping_buffer_minutes` ledger row** —
  rejected: ledger granularity is per-day, sub-day rows pollute the model.
- **Switch to `daterange`** — rejected: loses future flexibility for sub-day
  inventory (hourly rates, day-use rooms).
- **Two columns: `stay_dates DATERANGE` + `requested_checkin_at TIMESTAMPTZ`**
  — rejected: doubles the overlap-check surface, inconsistent.

## References

- `/docs/requirements/reservations.md` §5 (R-RES-VALID-001)
- `/docs/CONTEXT.md` — `stay_period`, Housekeeping buffer
- ADR-013 — Locking & availability strategy
