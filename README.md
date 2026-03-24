# Yop PMS

Yop PMS is a modern, high-performance Property Management System built for reliability and low operational cost. The stack is type-safe end-to-end — from the database schema to the frontend — to minimise runtime errors and maximise developer velocity.

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Frontend | SvelteKit 5 (Runes) | Reactive UI, minimal boilerplate |
| Backend | Go + Chi | High-performance HTTP server |
| Database | PostgreSQL 18 | Primary data store, ACID-compliant |
| Cache | Redis 7 | Read-through cache, idempotency, sessions |
| DB Access | SQLC | Type-safe query generation (no ORM) |
| Migrations | Goose | SQL migration management |
| API Contract | OpenAPI / Swagger | Schema-first, generated TypeScript types |
| Observability | OpenTelemetry | Distributed tracing + structured logging |
| Containerisation | Docker | `scratch`-based image (~30MB) |
| CI/CD | GitHub Actions | Build, test, deploy pipeline |

## Architecture

### Monorepo
All code, documentation, and infrastructure lives in one repository. A single `docker-compose.yml` and `Makefile` orchestrate the entire system. See [ADR-001](./docs/adr/001-monorepo.md).

### Schema-First API
Swagger annotations in Go handlers are the single source of truth. `make gen` produces `/api/openapi.json` and generates `/web/src/lib/types/api.d.ts` — the frontend never defines its own API types. See [ADR-003](./docs/adr/003-schema_first_api.md).

### Three-Layer Backend

```
Handler  →  parse request, validate input, write response
Service  →  business logic, caching, error mapping
Store    →  SQLC-generated database queries (never edit manually)
```

Handlers have no knowledge of the cache or database. Services own data retrieval and map all database errors to typed `APIError` responses before they reach the handler.

### Platform Layer

Cross-cutting concerns live in `internal/platform/` and are shared across all domains:

| Package | Purpose |
|---------|---------|
| `apierror` | Typed error responses with PostgreSQL SQLSTATE mapping |
| `cache` | Redis read-through cache client with prefix namespacing |
| `logging` | Structured JSON logging via `slog`, OTel-enriched per request |
| `middleware` | Request logger, idempotency enforcement |
| `otel` | OpenTelemetry tracer provider setup |
| `events` | PostgreSQL `LISTEN/NOTIFY` listener for reactive cache invalidation |

See [Platform Layer Guide](./docs/PLATFORM_LAYER_GUIDE.md) for usage patterns.

### Reactive Cache Invalidation
PostgreSQL triggers fire `NOTIFY` on reservation, guest, and pricing changes. The `events` listener receives these immediately and invalidates only the affected cache keys. TTLs (24h) are a safety net for listener downtime, not the primary freshness mechanism. See [ADR-010](./docs/adr/010-reactive-cache-invalidation.md).

### Database
- **Financials** — `INTEGER` only (smallest currency unit, no floats)
- **Timestamps** — `TIMESTAMPTZ` exclusively
- **Primary keys** — UUIDv7 (time-sortable)
- **Multi-tenancy** — `property_id` on every tenant-isolated table; Row-Level Security enabled
- **Soft deletes** — `deleted_at TIMESTAMPTZ`; uniqueness indexes use `WHERE (deleted_at IS NULL)`

See [ADR-004](./docs/adr/004-core_db_principles.md) and [Database Conventions](./docs/database/conventions.md).

## Getting Started

```bash
make setup     # Install tools, create .env, install npm deps
make docker-up # Start PostgreSQL, Redis, and the Go server
make dev       # Start Go (Air hot-reload) + SvelteKit (Vite) concurrently
```

Visit `/swagger/index.html` for the API docs.

See [Configuration](./docs/CONFIGURATION.md) for more in depth building

## Common Commands

```bash
make gen          # Regenerate OpenAPI spec + TypeScript types (run after changing Swagger comments)
make sqlc         # Regenerate SQLC store (run after changing SQL queries)
make test         # Run all tests
make audit        # go vet, govulncheck, svelte-check
make lint         # golangci-lint + Prettier
make format       # go fmt + Prettier
make reset-db     # Full DB teardown and restart
```

## Local Services

| Service | Port | Purpose |
|---------|------|---------|
| Go server | 8080 | Backend API |
| SvelteKit | 5173 | Frontend dev server |
| PostgreSQL | 5433 | Primary database |
| Redis | 6379 | Cache / sessions |
| Adminer | 8081 | Database admin UI |
| Redis Commander | 8082 | Redis admin UI |

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture Decision Records](./docs/adr/) | Why every major decision was made |
| [Platform Layer Guide](./docs/PLATFORM_LAYER_GUIDE.md) | How to use platform packages when building domain handlers |
| [API Contracts](./docs/API_CONTRACTS.md) | API design conventions and contract generation |
| [Configuration](./docs/CONFIGURATION.md) | Environment variables and configuration reference |
| [Testing Guide](./docs/TESTING_GUIDE.md) | Testing strategy and patterns |
| [Deployment](./docs/DEPLOYMENT.md) | Deployment procedures and infrastructure |
| [Database ERD](./docs/database/yop-pms-erd.md) | Entity-relationship diagram |
| [Database Conventions](./docs/database/conventions.md) | Schema design rules |
| [Roadmap](./docs/operations/roadmap.md) | Planned features and milestones |
