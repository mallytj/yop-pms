# Yop PMS — LLM Context Map

Compressed pointer map for AI agents. **No content here** — just routes
into the docs tree. Update when adding/removing files. Humans read
`docs/README.md`; agents read this.

## Stack

Go 1.23 (Chi) + SvelteKit 5 (Runes) + PostgreSQL 18 + Redis 7. SQLC for DB.
Schema-first API: Swagger comments → OpenAPI 3 → TS types. Pure CSS
frontend. Goose migrations. See `CLAUDE.md` (root) for commands.

## Doc tree

```
docs/
├── README.md              human entry point
├── CONTEXT.md             this file
├── TODO.md                outstanding structure changes (reservation API sprint)
├── DEPLOYMENT.md          prod deploy / scaling
├── adr/                   architecture decision records
├── guides/                how-tos (constraints, testing, platform)
├── requirements/          domain RTMs
├── flows/                 sequence diagrams per flow
├── database/              schema notes
├── operations/            ops runbooks
└── ideas/                 spike notes / unblessed proposals
```

## ADRs (`docs/adr/`)

Each is a hard, recorded decision. Read before contradicting.

- 001 Monorepo · 002 Techstack · 003 Schema-first API · 004 Core DB principles
- 005 Error handling · 006 Structured logging · 007 Idempotency-Key
- 008 Redis caching · 009 OpenTelemetry · 010 Reactive cache invalidation
- 011 Check-constraint consistency · 012 Transactional outbox
- 013 Locking & availability (hold-as-lock, auto-pin, ledger-as-truth)
- 014 Cursor pagination · 015 State-machine rollup
- 016 Guest-aware hold TTLs · 017 SSE for real-time frontend
- 018 `stay_period` time semantics (TSTZ bounds = property check-in/out)
- 019 Payment authorization model (deferred impl)
- 020 Reservation `stay_period_envelope` materialised column

## Requirements (`docs/requirements/`)

- `reservations.md` — reservation API RTM (canonical for current sprint)
- `reservation-groups.md` — groups (deferred)
- `ota-channels.md` — OTA inbound (deferred; v1 pins enum + dead-letter)
- `folios.md` — folio model (placeholder; finance PR)
- `authorization.md` — role + permission resolution

## Flows (`docs/flows/`)

- `reservations.md` — sequence diagrams for every reservation lifecycle path

## Guides (`docs/guides/`)

- `platform-layer.md` — logging, errors, caching, JSON helpers
- `backend-constraints.md` · `frontend-constraints.md` — generated validation
- `api-contracts.md` — response shapes, error codes, idempotency
- `openapi-sveltekit.md` — generated TS types in frontend
- `testing.md` · `configuration.md`

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

## What lives where (LLM cheatsheet)

| Need to | Look at |
| ------- | ------- |
| understand a domain rule | `requirements/<domain>.md` §relevant |
| see how a flow runs end-to-end | `flows/<domain>.md` |
| know why something is the way it is | `adr/NNN-*.md` |
| add a DB constraint | migration + `make gen-constraints` |
| add an endpoint | handler + Swagger comment + `make gen` |
| add a SQL query | `internal/store/queries/*.sql` + `make sqlc` |
| know commands | root `CLAUDE.md` or `Makefile` |
| see open work | `docs/TODO.md` |
