# Reservation API — Outstanding Structure Changes

Tracks schema, index, and endpoint changes from the 2026-05-09 design pass.
Companion to ADR-018/019/020. Tick items as they land in migrations / SQLC /
handlers.

> Project is **greenfield** — no existing prod data. "Backfill" tasks are
> trivial. Migrations may still need ordering for dev resets.

## Schema

### Migrations to add / amend

- [ ] **M4 amend** — `operations.property_settings`: add
  - `default_checkin_time TIME NOT NULL DEFAULT '15:00'`
  - `default_checkout_time TIME NOT NULL DEFAULT '11:00'`
  - `late_checkout_grace_minutes INT NOT NULL DEFAULT 60`
  - `cancellation_auth_amount_pence INT` (used by ADR-019; null = one
    night's rate)
- [ ] **M5 (new + amend)** —
  - `CREATE TYPE operations.ota_action AS ENUM ('create','modify','cancel')`
  - `operations.ota_inbound_messages`: confirm columns
    `(channel_id, channel_message_id, processed_at, response_jsonb,
    action operations.ota_action NOT NULL DEFAULT 'create')`
  - `modify` rows always dead-letter; `cancel` rows dead-letter until
    `channel_reservation_id` lookup column lands (deferred to OTA PR — see
    `docs/requirements/ota-channels.md`).
- [ ] **M6 (new)** — `operations.reservations.stay_period_envelope
  TSTZRANGE NOT NULL`. Greenfield → no backfill. Add CHECK
  `lower(envelope) < upper(envelope)`. (ADR-020)
- [ ] **M7 (new)** — extend `operations.reservation_item_status` enum with
  `'overstay'`. App-code transitions: `checked_in → overstay` (worker),
  `overstay → checked_in` (extend, actor), `overstay → checked_out`
  (force, actor).
- [ ] **M8 (new)** — rename `operations.checkout_sessions` →
  `operations.payment_authorizations`. Columns: `provider TEXT`,
  `auth_id TEXT`, `expires_at TIMESTAMPTZ`, `captured_at TIMESTAMPTZ`,
  `voided_at TIMESTAMPTZ`. Implementation deferred to finance PR
  (ADR-019). Schema may also defer; if so, leave the rename out until the
  finance PR.
- [ ] **M9!** — **DESTRUCTIVE, ATOMIC.** Single transaction. Rewrites every
  `operations.reservations.stay_period` from midnight bounds to
  property-time bounds (`default_checkin_time`, `default_checkout_time`).
  Greenfield so no prod fallout, but irreversible without migration log.
  The bang in the migration name flags the atomicity requirement to anyone
  running `make goose-circle`. (ADR-018)

- [ ] **M12 (new)** — add `daily_room_capacity INT CHECK (daily_room_capacity > 0)` (nullable = unlimited) to `pricing.daily_price_grid`. Enables rate plan capacity restrictions per calendar date (e.g. seasonal package capped at 3 rooms/day). Check at booking/rate-change time: count existing `booked_daily_rates` rows for `(rate_plan_id, calendar_date)` across property. Override via `reservations:override_rate_plan_capacity`.
- [ ] **M11 (new)** — add `'pending_cancellation'` to `operations.reservation_status` enum. Finance PR owns `pending_cancellation → cancelled` transition (fee settlement gate). Room re-bookable immediately on entry to this state.
- [ ] **M10 (new)** — fix `pricing.booked_daily_rates` unique constraint: drop `UNIQUE (reservation_item_id, calendar_date)`, replace with `UNIQUE (reservation_item_id, calendar_date) WHERE (deleted_at IS NULL)`. Required for soft-delete + re-insert pattern (§2.6 rate change, §3.4 early checkout, §2.1 date change). Per convention in CLAUDE.md.

## Indexes

- [ ] `CREATE INDEX ON operations.reservations USING GIST
  (property_id, stay_period_envelope)` — overlap filter.
- [ ] `CREATE INDEX ON operations.reservations
  (property_id, lower(stay_period_envelope), id)` — default arrival
  cursor (ADR-014 + ADR-020).
- [ ] `CREATE INDEX ON operations.reservations
  (property_id, created_at DESC, id)` — `?sort=created_at` cursor.
- [ ] `CREATE INDEX ON operations.reservations
  (property_id, upper(stay_period_envelope), id)` — departure-date
  filters.
- [ ] `CREATE INDEX ON operations.reservation_items (status)
  WHERE status = 'overstay'` — overstay sweep worker (M7).
- [ ] Re-validate the existing item-level
  `EXCLUDE GIST (assigned_room_id WITH =, stay_period WITH &&)` against
  the new TSTZ bounds. Greenfield = clean re-create.

## Triggers / service-layer hooks

- [ ] Recompute reservation `stay_period_envelope` in the same tx as any
  item insert / update / delete. Bump reservation `version`. (ADR-020)
- [ ] Worker: `overstay` sweep — every N minutes select
  `status='checked_in' AND now() > upper(stay_period) +
  property.late_checkout_grace_minutes * interval '1 minute'`
  for update skip locked, set `status='overstay'`. JOIN
  `property_settings` on `property_id`. (R-RES-WORKER-005)

## Endpoints (Swagger comments to write)

### New endpoints

- [ ] `POST /api/v1/reservations/{id}/confirm` — explicit hold confirm
  (`reservations:confirm` perm). Idempotent on `confirmed` (200 no-op).
- [ ] `POST /api/v1/reservations/{id}/items` — add item to non-terminal
  reservation (`reservations:add_item` perm). 409 if reservation terminal.
- [ ] `PATCH /api/v1/reservations/{id}/items/{item_id}/booked-rates` —
  per-night rate override (`reservations:rate_override` perm).
- [ ] `GET /api/v1/reservations/{id}/cancellation-quote` — stub returning
  `{computed_fee_pence: null, policy: "deferred", currency: <property.currency>}`
  until finance PR.
- [ ] `GET /api/v1/reservations/{id}/folios/{folio_id}` — paginated
  transactions (sub-resource per lean envelope).
- [ ] `GET /api/v1/reservations/{id}/items/{item_id}/booked-rates` —
  per-night rate read.

### Changed endpoints

- [ ] `GET /api/v1/reservations` — replace `date_from`/`date_to` with:
  `arriving_from`, `arriving_to`, `departing_from`, `departing_to`,
  `in_house_on`, `stay_overlaps_from`, `stay_overlaps_to`. Default
  sort = `arrival_date` asc.
- [ ] All POST responses (create) return the lean reservation envelope
  shape (= `GET /{id}` shape).
- [ ] Action endpoints (`/cancel`, `/confirm`, `/checkin`, `/checkout`,
  `/reactivate`, `/no-show`) return minimal `{id, version, status}`.
- [ ] Action endpoints implement §7.4 idempotency rules — constructive
  200 no-op vs destructive 409.

### Removed / superseded

- [x] Drop `R-RES-AVAIL-005` housekeeping-buffer enforcement (buffer is
  advisory only). Done in spec; no code change required yet.

## Spec doc edits (`reservations.md`)

All landed in this pass. Open items below are follow-ups that surfaced in
review:

- [ ] §6.5 — write rationale into `authorization.md` for why
  `reservations:confirm` is distinct from `reservations:create` (junior
  staff hold, senior confirm).
- [ ] §8 — define error catalog body shapes for new edges (058, 059, 061)
  alongside R-RES-VALID-013 already noted in flow 4.1.
- [ ] §13 — when M8 lands, retire `checkout_session` references in
  ADR-013 (mark partially superseded by ADR-019).

## Flows (`flows/reservations.md`)

All major flow edits landed (1.1 auth, 1.2 worker void, 1.3 confirm perm,
1.4 walk-in payload + rate plan, 2.7 add-item, 3.6 overstay, 4.1 cancel
guard). Open items:

- [ ] When groups land in a future sprint, add 4.8 Group cascade cancel
  (207 Multi-Status pattern).
- [ ] When SSE (ADR-017) lands, add a top-level note on push delivery vs
  polling for cache invalidation.

## Out of v1 scope

- Reservation groups (skeleton in §11; no endpoints in swagger).
- Cancellation policy schema + fee compute (caller-provided
  `fee_pence`; quote endpoint stub only).
- Cross-property guest dedup (per-property only for v1).
- OTA modify event handling (dead-letter to staff).
- OTA cancel routing — needs `channel_reservation_id` column, deferred to
  OTA PR (`docs/requirements/ota-channels.md`).
- Multi-currency cancellation maths (single property currency in quote).
- Item-level reactivation (use add-item instead).
- Denormalised `checked_in_at` / `checked_out_at` columns.
- Late-checkout fee auto-posting (manual folio_transaction for v1).
- B2B / travel-agent payment-auth carve-out (currently folded under
  "internal" — revisit when groups + corporate billing land).
