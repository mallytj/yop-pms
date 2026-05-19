# Price Fetching Flow

How a nightly rate is resolved for a given `(property_id, rate_plan_id, room_type_id, calendar_date)`.

---

## The Three Tiers

```
Tier 1 — daily_price_grid   (explicit per-day override)
Tier 2 — seasonal_rates     (date-range + day-of-week override)
Tier 3 — base_rates         (day-of-week fallback)
```

Resolution is **waterfall**: stop at the first tier that has a row. If no tier matches, the date has no price and is treated as unavailable.

---

## Tier Detail

### Tier 1 — `pricing.daily_price_grid`

Keyed on `(property_id, rate_plan_id, room_type_id, calendar_date)`.

An explicit row for the exact calendar date. Written by:
- Revenue manager setting or overriding a specific date
- OTA channel sync writing per-date prices
- The booking engine's rate matrix when a receptionist edits a specific day

Fields consumed:
| Column | Use |
|--------|-----|
| `base_price_pence` | Rate for that night |
| `min_los_restriction` | Minimum stay length gating availability |
| `max_los_restriction` | Maximum stay length gating availability |
| `is_available` | Hard block — even if price exists, date closed-out |

A row with `is_available = false` blocks the date regardless of price.

---

### Tier 2 — `pricing.seasonal_rates`

Keyed on `(property_id, rate_plan_id, room_type_id, day_of_week)` scoped to `override_period` (`TSTZRANGE`).

Applies when the calendar date falls within the range **and** matches the `day_of_week` (0 = Sunday … 6 = Saturday). Multiple seasonal records cannot overlap for the same `(property_id, room_type_id, rate_plan_id, day_of_week)` — enforced by a GiST EXCLUDE constraint.

Fields consumed: same as Tier 1 minus `is_available` (seasonal rates do not close-out dates).

---

### Tier 3 — `pricing.base_rates`

Keyed on `(property_id, rate_plan_id, room_type_id, day_of_week)`.

The standing weekly template. One row per day-of-week per `(room_type, rate_plan)` pair. No date range — always active unless superseded by Tier 1 or 2.

Fields consumed: `base_price_pence`, `min_los_restriction`, `max_los_restriction`.

---

## Resolution Algorithm

```
input: property_id, rate_plan_id, room_type_id, calendar_date

dow = day_of_week(calendar_date)   -- 0–6

1. SELECT from daily_price_grid
   WHERE property_id = $1
     AND rate_plan_id = $2
     AND room_type_id = $3
     AND calendar_date = $4
     AND deleted_at IS NULL
   → if found: use this row (check is_available first)

2. SELECT from seasonal_rates
   WHERE property_id = $1
     AND rate_plan_id = $2
     AND room_type_id = $3
     AND day_of_week = dow
     AND $4 <@ override_period      -- date inside range
     AND deleted_at IS NULL
   → if found: use this row

3. SELECT from base_rates
   WHERE property_id = $1
     AND rate_plan_id = $2
     AND room_type_id = $3
     AND day_of_week = dow
     AND deleted_at IS NULL
   → if found: use this row

4. No row found → date has no price; treat as unavailable
```

A single query can implement steps 1–3 with `LEFT JOIN` + `COALESCE` + `ROW_NUMBER` over a priority column, or as three CTEs with `UNION ALL LIMIT 1 ORDER BY tier`.

---

## Derived Rate Plans

`pricing.rate_plans` supports a parent/child hierarchy. A derived plan stores `parent_rate_plan_id` and `derivation_rule` (`{type: "percentage"|"fixed", value: N}`).

Resolution for a derived plan:

1. Run the waterfall for the **parent** plan to get `base_price_pence`.
2. Apply `derivation_rule`:
   - `percentage`: `ROUND(base_price_pence * (1 + value / 100.0))`
   - `fixed`: `base_price_pence + value` (value may be negative)
3. Result is the effective price for the derived plan.

LOS restrictions from the parent row apply unless the derived plan has its own explicit grid/seasonal/base row (which overrides entirely).

---

## LOS Enforcement

After resolving a price row, check each night of the requested stay:

```
∀ night ∈ stay:
  min_los_restriction(night) ≤ stay_length ≤ max_los_restriction(night)
```

Violation on any single night rejects the entire stay (per R-RES-AVAIL-008). `min_los_restriction` defaults to `1`; `max_los_restriction` defaults to `365` when NULL.

---

## Snapshot at Booking Time

When a reservation item is created (or rates recomputed), one `pricing.booked_daily_rates` row is written per night:

| Column | Source |
|--------|--------|
| `base_price_pence` | Result of waterfall above |
| `rate_plan_id` | The resolved plan ID (may differ per night if plan changed) |
| `adjustment` | NULL at creation; set by staff rate override (`PATCH .../booked-rates`) |
| `final_price_pence` | `base_price_pence` + applied adjustment (computed in Go) |

`booked_daily_rates` is a **snapshot** — subsequent changes to `daily_price_grid` / `seasonal_rates` / `base_rates` do not retroactively alter confirmed bookings. Recomputation is explicit (PATCH item `rate_plan_id` or the booked-rates endpoint).

---

## Closed-Out Dates

Only Tier 1 (`daily_price_grid.is_available = false`) closes a date. Tier 2 and Tier 3 have no availability flag — absence of a row at those tiers falls through to the next tier, not a block.

To close a date without overriding price: insert a `daily_price_grid` row copying the current effective price but with `is_available = false`.

---

## Related

- Schema: `migrations/00003_inventory_pricing.sql` §5–§6
- Snapshot table: `migrations/00004_reservations_finance.sql` §5
- Rate override flow: R-RES-CRUD-016 in `docs/requirements/reservations.md`
- LOS + availability: R-RES-AVAIL-008, R-RES-RATE-005
