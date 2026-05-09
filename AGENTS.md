# AGENTS.md

## Project Structure
- **Backend**: Go with Chi router in `cmd/` and `internal/`
- **Frontend**: SvelteKit 5 with Runes in `web/`
- **Database**: PostgreSQL 18 with SQLC for type-safe queries
- **Cache**: Redis 7
- **API Contracts**: OpenAPI/Swagger schema-first approach

## Key Commands
- `make setup` - First-time setup (install tools, create .env, install npm deps)
- `make docker-up` - Start PostgreSQL, Redis, Adminer, Redis Commander, and server
- `make dev` - Start Go (Air hot-reload) + SvelteKit (Vite) concurrently
- `make gen` - Full API contract generation (Swagger → OpenAPI → TypeScript types)
- `make test` - Run all tests (backend + frontend)
- `make audit` - Full quality suite: go mod verify, go vet, tests, govulncheck, svelte-check

## Architecture Notes
- **No ORM**: Raw SQL via SQLC for type safety without abstraction overhead
- **Schema-First API**: Swagger annotations in Go handlers → `make swag` → `/api/swagger.json` → OpenAPI → TypeScript types
- **Platform Layer**: Cross-cutting concerns in `internal/platform/` (config, db, middleware, logging, apierror, cache, constraints, events, helpers, json, otel)
- **Database Conventions**: Financials use INTEGER only, timestamps use TIMESTAMPTZ exclusively, UUIDv7 primary keys
- **Multi-tenancy**: Every tenant-isolated table has `property_id`; Row-Level Security enabled

## Doc-Driven Development
- **Mandate**: All feature implementation and logic modifications MUST follow the sequence diagrams in \`docs/flows/\`, Requirement Traceability Matrices in \`docs/requirements/\`, relevant \`docs/adr/\`, and technical \`docs/guides/\`.
- **Workflow**: Use the \`doc-driven-development\` skill to map implementation plans to diagram steps.
- **Traceability**: Core requirements and ADRs MUST be cited in file headers: \`// Core Requirements: [R-RES-XXX], [ADR-XXX]\`.