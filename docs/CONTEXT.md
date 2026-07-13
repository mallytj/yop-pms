# Yop PMS — LLM Context Map

Compressed pointer map for AI agents. **No content here** — just routes
into the docs tree. Update when adding/removing files. Humans read
the root `README.md`; agents read this.

## Stack

Go 1.23 (Chi) + SvelteKit 5 (Runes) + PostgreSQL 18 + Redis 7. SQLC for DB.
Schema-first API: Swagger comments → OpenAPI 3 → TS types. Pure CSS
frontend. Goose migrations. See `AGENTS.md` (root) for commands.

## Doc tree

```
docs/
├── CONTEXT.md                   this file
├── agentic-engineering-workflow.md
├── adr/                         architecture decision records
├── agents/                      agent skills hub (domain, issue-tracker, triage-labels)
├── guides/                      how-tos (api-contracts, openapi-ts-usage)
├── requirements/                legacy RTMs (superseded by agentic-engineering-workflow.md)
├── flows/                       sequence diagrams per flow
├── conventions/                 DB conventions, ERD
├── pruned/                      deferred decisions (e.g. payment authorization)
├── role-specs/                  role job spec templates
└── loom-placeholder.*           screenshot placeholders for loom
```

## ADRs (`docs/adr/`)

Each is a hard, recorded decision. Read before contradicting.

- 001 Schema-first API · 002 Core DB principles
- 003 Error handling · 004 Idempotency-Key · 005 Check-constraint consistency
- 006 Transactional outbox · 007 Locking & availability (hold-as-lock, auto-pin, ledger-as-truth)
- 008 Cursor pagination · 009 State-machine rollup
- 010 Guest-aware hold TTLs · 011 SSE for real-time frontend
- 012 `stay_period` time semantics (TSTZ bounds = property check-in/out)
- 013 Reservation `stay_period_envelope` materialised column
- 014 Audit logs via database trigger (reservations + items → `auth.audit_logs`)
- 015 Response depth via `?include=`

Pruned: 019 Payment authorization model (deferred to finance PR; see `docs/pruned/`).

## Requirements — Legacy RTMs (`docs/requirements/`)

> **⚠️ SUPERSEDED** by [Agentic Engineering Workflow](agentic-engineering-workflow.md).
> New requirements go in Linear Job Specs (Stage 4: to-spec).
> These files exist as historical reference for implemented code.

- `reservations.md` — reservation API RTM (legacy)
- `reservation-groups.md` — groups (legacy, deferred)
- `ota-channels.md` — OTA inbound (legacy, deferred)
- `folios.md` — folio model (legacy, placeholder)
- `authorization.md` — role + permission resolution (legacy, placeholder)
- `db.md` — DB-level requirements (legacy)

## Flows (`docs/flows/`)

- `reservations.md` — sequence diagrams for every reservation lifecycle path

## Guides (`docs/guides/`)

- `api-contracts.md` — response shapes, error codes, idempotency
- `openapi-ts-usage.md` — generated TS types in frontend

Platform packages self-document via Go package comments in `internal/platform/*`.

## Code layout

```
cmd/server/             entry: main.go, api.go (route reg)
internal/booking/       reservation domain (current sprint)
internal/planner/       planner domain
internal/pricing/       rate plans, price grid
internal/platform/      cross-cutting (config, db, mw, cache, otel, etc.)
internal/store/         SQLC-generated DB layer (DO NOT EDIT)
internal/store/queries/ raw SQL (edited; SQLC source)
migrations/             Goose migrations (zero-padded sequential)
web/                    SvelteKit 5 frontend
config/                 generated constraints + runtime config
```

## Conventions (must follow)

- Financials: `INTEGER` pence/cents only (no floats)
- Timestamps: `TIMESTAMPTZ`
- PKs: UUIDv7
- FKs: `RESTRICT` on delete (preserve history)
- Soft delete: `deleted_at TIMESTAMPTZ`; uniqueness `WHERE deleted_at IS NULL`
- Multi-tenancy: `property_id` + RLS via `app.current_property_id`
- Optimistic locking: `version` column on high-concurrency tables
- Constraint names: `{table}_{column}_{suffix}`
- `make gen` after Swagger changes; `make gen-constraints` after CHECK changes

## Workflow

1. Edit Swagger comments in handler → `make swag` → OpenAPI → TS types
2. Edit SQL in `internal/store/queries/` → `make sqlc`
3. Run `make audit` before commit
4. Pre-push hook enforces `make gen`

## Domain Terms

### Tape Chart (Live Booking Grid)

**Block** — A coloured bar spanning date columns in the grid. Each block represents one `reservation_item` (solid) or one `maintenance_block` (striped/diagonal). One row (room) × N date columns (stay period). Items from the same reservation share a colour accent for visual grouping.

**Hold** — A reservation with `status=hold`. Unconfirmed; TTL-bound (`website_hold_ttl_seconds`, `internal_hold_ttl_seconds` in `property_settings`). Drawn as a dashed/outline block to distinguish from `confirmed`.

**Room status (housekeeping)** — `clean → dirty → in_progress → inspected`. Also `out_of_service`, `linen_change`. Only visible on the housekeeping tape chart view — NOT on the reception planner. Determines room-row colour/icon in housekeeping mode.

**Room condition** — `occupied | vacant | reserved | out_of_service | checked_out`. Cross-role view of whether a slot is usable. Drives basic cell affordance (empty cell clickable for create drag only if condition permits).

**Drag modes** — Three interaction modes:
  - **Create drag**: Drag down/right from empty cell → creates a new `reservation_item` with `status=hold`. Optimistic local update, server-validated on save.
  - **Move drag**: Drag block body → changes room assignment or dates. For `status=checked_in` items this triggers the room-move flow (§2.3 of reservations flow); DNM check applies. Shortens/stretches stay as a single operation on the item (no split).
  - **Resize drag**: Drag block edge → extends or shortens stay. Shorten = one operation on existing item, no split.

**Conflict** — Validated on save. Three types:
  - **Double-book**: Two reservation blocks same room/date
  - **Maintenance overlap**: Reservation block × maintenance block same room/date. Maintenance is exclusive — someone should not be in a room during a maintenance period.
  - **Status-locked**: Room condition blocks write (checked_in guest, out_of_service, past-night).
  Conflict resolution: hold-expiry race is handled bidirectionally — local optimistic UI update, server validation on save, and SSE push from other sessions that overrides stale local state.

**Today column** — Grid date column representing current date at property local time. Advances at midnight local time. Virtual scroll works both directions (past and future) — today is not locked to left edge.

### Common (pre-existing)

**Item (ReservationItem)** — A single room's stay within a reservation. Carries its own `stay_period` (TSTZRANGE), room assignment, occupancy (`adults_count`, `children_count`), rate plan, and status. Multiple items = multiple rooms. One item = one capacity consumption unit per night. The reservation's `stay_period_envelope` is the union of its items' periods (ADR-013).
_Avoid_: Line item, room booking, sub-reservation

**Property Settings** — Per-property operational configuration stored as columns on `operations.property_settings`. Includes hold TTLs (`website_hold_ttl_seconds`, `internal_hold_ttl_seconds`), checkout grace periods (`late_checkout_grace_minutes`), archive thresholds (`reservation_archive_after_days`), and no-show grace (`no_show_grace_minutes`). Read on every hold-create and every worker tick.

**Audit Log** — Immutable record in `auth.audit_logs` of every INSERT/UPDATE/DELETE on `reservations` and `reservation_items`. Written automatically by database trigger (ADR-014), never by application code. Records `user_id`, `action`, `entity`, `entity_id`, and a `changes JSONB` diff.
_Avoid_: Event log, activity feed, change history

**Admin/Tab Room** — A house reservation kept permanently in `checked_in` used as a holding account for outstanding balances. When a guest's folio cannot be settled at checkout (e.g. corporate billing, disputed charge), staff transfers the balance to the Admin/Tab Room folio before checking the guest out. Folio transfer is a finance PR concern. Checkout hard-blocks on `balance > 0` — this is the standard resolution path.

**Do Not Move (DNM)** — Flag on `reservation_item` set by staff. Indicates guest must not be relocated. Checked before room assignment (§2.2), reassignment, post-checkin room move (§2.3), and mid-stay room type change (§2.5). Override requires `reservations:override_dnm` permission + recorded reason. Frontend shows warning; hard block without the permission.

## Example dialogue

> **Dev:** "When a staff member checks in a guest, what happens to the audit log?"
> **Domain expert:** "Nothing the handler needs to do. The database trigger writes an **Audit Log** row automatically. All the handler does is update the **Item** status to `checked_in`. The trigger captures the old status, new status, who did it, and when."
> **Dev:** "And if the **Property Settings** have a 2-hour hold TTL, and the hold expires?"
> **Domain expert:** "The **Hold Expiry Sweep** worker cancels the **Reservation** and all its **Items**. The trigger writes audit rows for each change with `user_id` set to the system user. Same guarantee — no code path can skip the audit."

## What lives where (LLM cheatsheet)

| Need to | Look at |
| ------- | ------- |
| understand a domain rule | `agentic-engineering-workflow.md` (new) or legacy `requirements/<domain>.md` |
| see how a flow runs end-to-end | `flows/<domain>.md` |
| know why something is the way it is | `adr/NNN-*.md` |
| add a DB constraint | migration + `make gen-constraints` |
| add an endpoint | handler + Swagger comment + `make gen` |
| add a SQL query | `internal/store/queries/*.sql` + `make sqlc` |
| know commands | root `AGENTS.md` or `Makefile` |
| see ADR index | `docs/adr/README.md` |
