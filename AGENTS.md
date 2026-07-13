# AGENTS.md

> See `docs/CONTEXT.md` (domain terms, doc tree) + `.pi/` (skills, agents)

## Stack

- **Backend:** Go + Chi (`cmd/`, `internal/`)
- **Frontend:** SvelteKit 5 with Runes (`web/`, aliases: `$lib`, `$components`,
  `$helpers`, `$stores`, `$types`, `$actions`)
- **Database:** PostgreSQL 18 + SQLC (typed queries)
- **Cache:** Redis 7
- **API:** Schema-first (Swagger → OpenAPI → TS types)

## Key Commands

### Development

```bash
make setup          # Install tools, .env, npm deps
make docker-up      # PG, Redis, Adminer, Redis Commander, server
make db-up          # PG + Redis only (Go via Air locally)
make dev            # Go (Air) + SvelteKit (Vite) concurrently
```

### Code Generation

```bash
make gen                # Full pipeline (Swagger → OpenAPI → TS types)
make swag               # Swagger/OpenAPI from Go comments
make sqlc               # Typed DB code from SQL queries
make gen-constraints    # Sync constraints.g.yml + .ts from live DB
```

Run `make gen` after Swagger comment changes. `make gen-constraints` after DB
constraint changes.

### Testing

```bash
make test           # All tests (backend + frontend)
make test-backend   # go test -race ./...
```

### Quality

```bash
make audit    # go mod verify, go vet, tests, govulncheck, svelte-check
make lint     # golangci-lint (Go) + Prettier (frontend)
make format   # go fmt (Go) + Prettier (frontend)
```

Pre-push hooks enforce lint + test. Run `make lint` after code gen.

### Database

```bash
make reset-db        # docker-compose down -v + up
make goose-circle    # Full migration reset
```

Migrations: `/migrations/`. Zero-padded filenames. `GOOSE_*` env vars in `.env`.

## Generated-Code Boundaries

**Generated — never edit manually:**

| Path                                      | Source                           | Regenerate               |
| ----------------------------------------- | -------------------------------- | ------------------------ |
| `internal/store/`                         | SQL in `internal/store/queries/` | `make sqlc`              |
| `/api/swagger.json` + `/api/openapi.json` | Go Swagger comments              | `make swag` / `make gen` |
| `web/src/lib/types/api.d.ts`              | OpenAPI 3.0 spec                 | `make gen`               |
| `config/constraints.g.yml`                | Live DB check constraints        | `make gen-constraints`   |
| `web/src/lib/types/constraints.g.ts`      | Live DB check constraints        | `make gen-constraints`   |

Edit **source** (SQL, Go Swagger comments, live DB), re-run gen command.

## DB Conventions (enforced in migrations)

- **Financials:** `INTEGER` only (smallest unit — no floats)
- **Timestamps:** `TIMESTAMPTZ` exclusively
- **Primary keys:** UUIDv7 (time-sortable)
- **Foreign keys:** `RESTRICT` on delete (preserve history)
- **Soft deletes:** `deleted_at TIMESTAMPTZ` — unique indexes use
  `WHERE (deleted_at IS NULL)`
- **Multi-tenancy:** Every tenant-isolated table has `property_id`; RLS enabled
- **Optimistic locking:** `version` column on high-concurrency tables
- **Text fields:** Check constraints on length; `CITEXT` for emails, usernames,
  codes
- **Constraint naming:** `{table}_{column}_{suffix}`

## API Contract Flow (Schema-First)

1. Swagger annotations in Go handlers → `make swag` → `/api/swagger.json`
2. Convert to OpenAPI 3.0 → `/api/openapi.json`
3. Generate TypeScript types → `web/src/lib/types/api.d.ts`

**Frontend: never define own API types** — all from generation.

## Doc-Driven Development

- Feature implementation MUST follow sequence diagrams (`docs/flows/`), RTMs
  (`docs/requirements/`), ADRs (`docs/adr/`), guides (`docs/guides/`).
- Use `doc-driven-development` skill to map plans to diagram steps.
- Cite requirements + ADRs in file headers:
  `// Core Requirements: [R-RES-XXX], [ADR-XXX]`
- Use `validation.Struct(input, "schema.table")` for struct validation. Add
  `constraints:"schema.table"` tag on nested slice fields.

## Agent Resources

| Resource                | File                                   | Purpose                                                     |
| ----------------------- | -------------------------------------- | ----------------------------------------------------------- |
| Domain context          | `docs/CONTEXT.md`                      | Domain terms, doc tree, cheat sheet                         |
| ADR index               | `docs/adr/README.md`                   | Architecture decisions (001-015)                            |
| Issue tracker           | `docs/agents/issue-tracker.md`         | Linear CLI v2, branch naming, lifecycle                     |
| Triage labels           | `docs/agents/triage-labels.md`         | 5 canonical label mapping                                   |
| Domain docs consumption | `docs/agents/domain.md`                | How to explore domain docs                                  |
| Workflow                | `docs/agentic-engineering-workflow.md` | 6-stage build process                                       |
| Skills                  | `.pi/skills/`                          | Reusable procedures (ask-yop, run-audit, code-review, etc.) |

Advisory agents (CTO, Boutique Director, Compliancy, UX Expert): invoke via
`run-audit` skill. Fresh context each run — don't auto-read AGENTS.md or docs/.

## Rules (do not rely on inference)

- **Keep frontend Pure CSS** — no framework, no Tailwind, no Bootstrap.
- **Ask before adding deps.** Prefer stdlib + well-known, actively maintained
  packages.
- **Never edit generated files** (`internal/store/`, `/api/`,
  `config/constraints.g.*`, `web/src/lib/types/api.d.ts`,
  `web/src/lib/types/constraints.g.ts`).
- **Never commit secrets, API keys, or absolute local paths.**
- **Run `make lint` after code gen** — linters catch what generators miss.
- **Run `make test` when test files or related source change.**
- **Keep Go files under ~300 lines.** Split when exceeds.
- **Redis: transient only** — sessions, token blacklists, rate limits. Not
  primary store. Data must survive Redis flush.
