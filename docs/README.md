# Yop PMS Documentation

Complete documentation for the Yop PMS (Property Management System) project.

## Quick Navigation

### For New Developers

1. **Start here:** [Project Overview](../README.md) — Architecture, stack, running the project
2. **Platform Layer:** [guides/platform-layer.md](guides/platform-layer.md) — How to use logging, errors, caching, JSON handling
3. **Backend Constraints:** [guides/backend-constraints.md](guides/backend-constraints.md) — Type-safe backend validation from DB schema
4. **Frontend Constraints:** [guides/frontend-constraints.md](guides/frontend-constraints.md) — Shared validation rules for the UI
5. **Writing Tests:** [guides/testing.md](guides/testing.md) — Unit tests, integration tests, mocks
6. **Configuration:** [guides/configuration.md](guides/configuration.md) — Environment variables, dev setup

### For API Consumers

1. **API Contracts:** [guides/api-contracts.md](guides/api-contracts.md) — Response formats, error codes, idempotency, pagination
2. **OpenAPI in SvelteKit:** [guides/openapi-sveltekit.md](guides/openapi-sveltekit.md) — Using generated types in the frontend
3. **Deployment:** [DEPLOYMENT.md](DEPLOYMENT.md) — Running in production, Docker, Kubernetes

### For Architects

1. **Architecture Decisions:** [adr/](adr/) — Why we made each major decision
2. **Scaling Guide:** [DEPLOYMENT.md#scaling-considerations](DEPLOYMENT.md#scaling-considerations) — Growing beyond single instance

---

## Documentation Index

### Architecture Decision Records (ADRs)

All major decisions documented with context, tradeoffs, and consequences:

| ADR                                            | Title                       | Status   |
| ---------------------------------------------- | --------------------------- | -------- |
| [001](adr/001-monorepo.md)                     | Monorepo Structure          | Accepted |
| [002](adr/002-techstack.md)                    | Core Tech Stack             | Accepted |
| [003](adr/003-schema_first_api.md)             | Schema-First API            | Accepted |
| [004](adr/004-core_db_principles.md)           | Core DB Principles          | Accepted |
| [005](adr/005-error-handling-strategy.md)      | Error Handling Strategy     | Accepted |
| [006](adr/006-structured-logging-approach.md)  | Structured Logging          | Accepted |
| [007](adr/007-idempotency-key-enforcement.md)  | Idempotency Keys            | Accepted |
| [008](adr/008-redis-caching-layer.md)          | Redis Caching Layer         | Accepted |
| [009](adr/009-opentelemetry-observability.md)  | OpenTelemetry Observability | Accepted |
| [010](adr/010-reactive-cache-invalidation.md)  | Reactive Cache Invalidation | Accepted |
| [011](adr/011-check-constraint-consistency.md) | Constraint Consistency      | Accepted |

### Developer Guides

| Document                                                         | Purpose                                                  |
| ---------------------------------------------------------------- | -------------------------------------------------------- |
| [guides/platform-layer.md](guides/platform-layer.md)             | Using error handling, logging, JSON, caching in handlers |
| [guides/backend-constraints.md](guides/backend-constraints.md)   | Backend validation using DB constraints                  |
| [guides/frontend-constraints.md](guides/frontend-constraints.md) | Frontend validation using DB constraints                 |
| [guides/api-contracts.md](guides/api-contracts.md)               | Standard response formats, error codes, headers          |
| [guides/testing.md](guides/testing.md)                           | Writing unit and integration tests                       |
| [guides/configuration.md](guides/configuration.md)               | Environment variables and setup                          |
| [guides/openapi-ts-usage.md](guides/openapi-ts-usage.md)         | Using OpenAPI contracts for the frontend                 |

---

## Platform Packages Overview

The platform layer provides cross-cutting concerns for all domain endpoints:

### `internal/platform/apierror` — Consistent Error Handling

- Sentinel errors: `ErrNotFound`, `ErrBadRequest`, `ErrConflict`, etc.
- Automatic PostgreSQL error mapping (SQLSTATE → HTTP status)
- Immutable error messages via `WithMessage()`

**When to use:** Every handler that returns errors

```go
json.WriteError(w, r, apierror.ErrNotFound)
json.WriteError(w, r, err)  // Automatically maps DB errors
```

### `internal/platform/logging` — Structured Request Logging

- Per-request logger with context (request_id, method, path, remote_ip)
- OpenTelemetry trace/span ID enrichment
- Environment-aware log levels (Debug in dev, Info in prod)

**When to use:** Every handler function

```go
logger := logging.FromContext(r.Context())
logger.Info("processing request", "property_id", propertyID)
```

### `internal/platform/json` — HTTP Response Encoding

- `WriteJSON(w, status, data)` — Encode successful responses
- `WriteError(w, r, err)` — Encode errors with automatic mapping
- `ReadJSON(r, &dst)` — Parse requests with validation

**When to use:** All HTTP I/O in handlers

```go
json.WriteJSON(w, http.StatusOK, property)
json.WriteError(w, r, apierror.ErrBadRequest)
```

### `internal/platform/cache` — Redis Caching

- Get/Set/Delete operations with JSON serialization
- `GetOrSet()` for read-through caching
- Pattern-based invalidation (SCAN+DEL, not KEYS)
- Prefix namespacing to avoid collisions

**When to use:** Frequently-accessed read-heavy data

```go
var availability Availability
err := app.cache.GetOrSet(ctx, key, &availability, 1*time.Hour,
    func(ctx context.Context) (any, error) {
        return app.store.GetAvailability(ctx, propID, date)
    })
```

### `internal/platform/events` - Event Listener

- `LISTEN/NOTFIY` implementation for Go
- Used for reactive cache invalidation
- May be used for WebSockets in future

```go
 eventListener := events.New(cfg.DatabaseURL, logger, func() {
  if err := appCache.Invalidate(context.Background(), "yop:*"); err != nil {
   logger.Error("failed to flush cache on event listener reconnect", "error", err)
  }
 })

 eventListener.On("reservation_changes", cache.NewReservationChangeHandler(appCache, logger))
 eventListener.Start()
 defer eventListener.Stop()
```

### `internal/platform/constraints`

- Consistent constraints between database, backend and frontend
- Gets information from `config/constraints.g.yml`
- [Usage Example](./guides/backend-constraints.md)

---

## Common Tasks

### Add a New HTTP Endpoint

1. Create handler in domain package (e.g., `internal/booking/handlers.go`)
2. Use `json.ReadJSON()` to parse request
3. Call domain logic via store
4. Return result via `json.WriteJSON()` or error via `json.WriteError()`
5. Don't forget: `logging.FromContext()` for observability
6. Add Swagger comments for documentation

See [guides/platform-layer.md#full-example-handler](guides/platform-layer.md#full-example-handler)

### Write a Test

1. Mock external dependencies (store, cache)
2. Inject logger via `logging.WithContext()`
3. Use `httptest` to create request/response
4. Assert status code and response body

See [guides/testing.md#testing-handlers](guides/testing.md#testing-handlers)

### Handle a Database Error

1. Call store function
2. If error, pass directly to `json.WriteError()`
3. SQLSTATE is automatically mapped to appropriate HTTP status
4. Custom message extracted from CHECK constraint or RAISE exception

See [guides/platform-layer.md#handling-database-errors](guides/platform-layer.md#handling-database-errors)

### Cache a Query Result

1. Define cache key (e.g., `yop:availability:property-uuid:date`)
2. Use `cache.GetOrSet()` with loader function
3. On data mutation, call `cache.Invalidate()` to clear related keys

See [guides/platform-layer.md#caching](guides/platform-layer.md#caching)

### Listen to an event

1. Set `NOTIFY` in database
2. Create `EventListener`
3. Create event handlers in `internal/platform/event`

See [guides/platform-layer.md#events](guides/platform-layer.md#events)

### Accessing constraints

See [guides/backend-constraints](./guides/backend-constraints.md) for backend
See [guides/frontend-constraints](./guides/frontend-constraints.md) for frontend

## Key Concepts

### Idempotency

All POST/PATCH requests must include `Idempotency-Key` header. The first request reserves the key atomically, successful responses are cached for 24 hours, and repeating the same request returns the same response without duplicate processing. Reusing a key for a different request returns 409 Conflict.

See [ADR-007](adr/007-idempotency-key-enforcement.md)

### Error Responses

All errors follow a consistent format:

```json
{
  "code": "CONFLICT",
  "message": "the dates overlap with an existing reservation",
  "status": 409
}
```

See [guides/api-contracts.md#error-response-format](guides/api-contracts.md#error-response-format)

---

## Contributing to Docs

When adding features:

1. **Write an ADR** if it's a major decision
2. **Update guides/platform-layer.md** if it affects handler development
3. **Update guides/configuration.md** if it adds environment variables
4. **Update guides/testing.md** if it introduces new testing patterns

Keep docs in sync with code. Stale docs are worse than no docs.
