# ADR 012: Stay Period Time Semantics

## Status

**Accepted**

`reservation_item.stay_period` bounds carry property check-in / check-out timestamps, not midnight: `lower = arrival_date @ property.default_checkin_time`, `upper = departure_date @ property.default_checkout_time`. API accepts `arrival_date` / `departure_date` as DATEs; server composes the TSTZRANGE using property defaults. Explicit timestamps (early arrival / late checkout) require the `reservations:override_restrictions` permission. `housekeeping_buffer_minutes` is advisory only — surfaced to staff but not enforced; same-day turnover protection emerges naturally from the gap between default checkout and check-in. Ledger rows key on `calendar_date`; included dates are `{d : d ∈ [lower::date, upper::date)}` (departure date excluded).

Alternatives: midnight bounds + ledger buffer row (sub-day rows pollute model), `daterange` (loses sub-day flexibility), two columns `stay_dates` + `requested_checkin_at` (doubles overlap surface).

---

See: `docs/requirements/reservations.md` §5 (R-RES-VALID-001), `docs/CONTEXT.md` (`stay_period`)
