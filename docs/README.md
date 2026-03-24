# Yop PMS Documentation

Complete documentation for the Yop PMS (Property Management System) project.

## Quick Navigation

### For New Developers

1. **Start here:** [Project Overview](../README.md) — Architecture, stack, running the project
2. **Platform Layer:** [PLATFORM_LAYER_GUIDE.md](PLATFORM_LAYER_GUIDE.md) — How to use logging, errors, caching, JSON handling
3. **Writing Tests:** [TESTING_GUIDE.md](TESTING_GUIDE.md) — Unit tests, integration tests, mocks
4. **Configuration:** [CONFIGURATION.md](CONFIGURATION.md) — Environment variables, dev setup

### For API Consumers

1. **API Contracts:** [API_CONTRACTS.md](API_CONTRACTS.md) — Response formats, error codes, idempotency, pagination
2. **Deployment:** [DEPLOYMENT.md](DEPLOYMENT.md) — Running in production, Docker, Kubernetes

### For Architects

1. **Architecture Decisions:** [adr/](adr/) — Why we made each major decision
2. **Scaling Guide:** [DEPLOYMENT.md#scaling-considerations](DEPLOYMENT.md#scaling-considerations) — Growing beyond single instance

---

## Documentation Index

### Architecture Decision Records (ADRs)

All major decisions documented with context, tradeoffs, and consequences:

| ADR | Title | Status |
|-----|-------|--------|
| [0001](adr/0001-error-handling-strategy.md) | Error Handling Strategy | Accepted |
| [0002](adr/0002-structured-logging-approach.md) | Structured Logging Approach | Accepted |
| [0003](adr/0003-idempotency-key-enforcement.md) | Idempotency Key Enforcement | Accepted |
| [0004](adr/0004-redis-caching-layer.md) | Redis Caching Layer Design | Accepted |
| [0005](adr/0005-opentelemetry-observability.md) | OpenTelemetry Observability | Accepted |

### Developer Guides

| Document | Purpose |
|----------|---------|
| [PLATFORM_LAYER_GUIDE.md](PLATFORM_LAYER_GUIDE.md) | Using error handling, logging, JSON, caching in handlers |
| [API_CONTRACTS.md](API_CONTRACTS.md) | Standard response formats, error codes, headers |
| [TESTING_GUIDE.md](TESTING_GUIDE.md) | Writing unit and integration tests |
| [CONFIGURATION.md](CONFIGURATION.md) | Environment variables and setup |

### Operations Guides

| Document | Purpose |
|----------|---------|
| [DEPLOYMENT.md](DEPLOYMENT.md) | Building, deploying, and scaling to production |

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
- Simple Get/Set/Delete operations with JSON serialization
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

### `internal/platform/middleware` — HTTP Middleware
- **RequestLogger** — Logs all requests with latency and status
- **Idempotency** — Redis-backed idempotency enforcement for POST/PATCH

**When to use:** Request processing (auto-applied by router)

---

## Common Tasks

### Add a New HTTP Endpoint

1. Create handler in domain package (e.g., `internal/booking/handlers.go`)
2. Use `json.ReadJSON()` to parse request
3. Call domain logic via store
4. Return result via `json.WriteJSON()` or error via `json.WriteError()`
5. Don't forget: `logging.FromContext()` for observability
6. Add Swagger comments for documentation

See [PLATFORM_LAYER_GUIDE.md#full-example-handler](PLATFORM_LAYER_GUIDE.md#full-example-handler)

### Write a Test

1. Mock external dependencies (store, cache)
2. Inject logger via `logging.WithContext()`
3. Use `httptest` to create request/response
4. Assert status code and response body

See [TESTING_GUIDE.md#testing-handlers](TESTING_GUIDE.md#testing-handlers)

### Deploy to Production

1. Build Docker image
2. Push to registry
3. Run database migrations
4. Update Kubernetes deployment
5. Monitor health check and logs

See [DEPLOYMENT.md](DEPLOYMENT.md)

### Handle a Database Error

1. Call store function
2. If error, pass directly to `json.WriteError()`
3. SQLSTATE is automatically mapped to appropriate HTTP status
4. Custom message extracted from CHECK constraint or RAISE exception

See [PLATFORM_LAYER_GUIDE.md#handling-database-errors](PLATFORM_LAYER_GUIDE.md#handling-database-errors)

### Cache a Query Result

1. Define cache key (e.g., `yop:availability:property-uuid:date`)
2. Use `cache.GetOrSet()` with loader function
3. On data mutation, call `cache.Invalidate()` to clear related keys

See [PLATFORM_LAYER_GUIDE.md#caching](PLATFORM_LAYER_GUIDE.md#caching)

---

## Key Concepts

### Idempotency

All POST/PATCH requests must include `Idempotency-Key` header. Responses are cached for 24 hours, so repeating the same request returns the same result (no duplicates).

**Why?** Protects against network retries accidentally creating duplicate bookings.

See [ADR-0003](adr/0003-idempotency-key-enforcement.md)

### Error Responses

All errors follow a consistent format:

```json
{
  "code": "CONFLICT",
  "message": "the dates overlap with an existing reservation",
  "status": 409
}
```

**Why?** API consumers can parse errors programmatically; `code` is stable while `message` is human-friendly.

See [API_CONTRACTS.md#error-response-format](API_CONTRACTS.md#error-response-format)

### Structured Logging

All logs are JSON with context:

```json
{
  "time": "2026-03-15T10:30:00Z",
  "level": "INFO",
  "msg": "request completed",
  "request_id": "550e8400-e29b",
  "method": "POST",
  "path": "/v1/reservations",
  "status": 201,
  "latency_ms": 45.3,
  "trace_id": "abc123def456"
}
```

**Why?** Enables log aggregation (ELK, Datadog) and correlation with traces.

See [ADR-0002](adr/0002-structured-logging-approach.md)

### OpenTelemetry Tracing

HTTP requests automatically create spans. Database queries are traced. All logs include trace_id + span_id for correlation.

**Why?** In production, traces answer "why was this request slow?" across the entire stack.

See [ADR-0005](adr/0005-opentelemetry-observability.md)

---

## Frequently Asked Questions

**Q: Where do I add business logic?**
A: Domain packages like `internal/booking/`, `internal/pricing/`. Keep handlers thin (just HTTP I/O).

**Q: How do I handle API versioning?**
A: Routes are under `/v1`. When breaking changes come, create `/v2` routes alongside `/v1`.

**Q: Can I use a different cache besides Redis?**
A: Yes, but you'd swap the implementation at `app.cache = cache.New(...)` in main.go. The interface is the same.

**Q: What if Redis goes down?**
A: Idempotency middleware fails open (allows request through with warning). Cache reads return ErrCacheMiss (query runs again). API stays available.

**Q: How do I add metrics/alerts?**
A: See [DEPLOYMENT.md#monitoring--alerts](DEPLOYMENT.md#monitoring--alerts). OpenTelemetry metrics are exportable to Prometheus.

---

## Contributing to Docs

When adding features:

1. **Write an ADR** if it's a major decision (error handling, caching, authentication, etc.)
2. **Update PLATFORM_LAYER_GUIDE.md** if it affects handler development
3. **Update CONFIGURATION.md** if it adds environment variables
4. **Update DEPLOYMENT.md** if it requires infrastructure changes
5. **Update TESTING_GUIDE.md** if it introduces new testing patterns

Keep docs in sync with code. Stale docs are worse than no docs.

---

## Documentation Standards

- Use **Markdown** for all docs
- Include **code examples** for everything
- Link to **source code** where possible (e.g., `internal/platform/apierror/apierror.go`)
- Add **context** (why we do something, not just what)
- Keep **ADRs** as records (don't update them; create new ones for reversals)
- Use **tables** for reference material

---

## Related Resources

### Inside This Repo

- `CLAUDE.md` — Instructions for Claude Code when working in this repo
- `Makefile` — Common commands (make dev, make test, make deploy)
- `internal/platform/` — Platform package implementation
- `cmd/server/` — Server bootstrap

### External

- [Go Best Practices](https://golang.org/doc/effective_go)
- [PostgreSQL SQLSTATE codes](https://www.postgresql.org/docs/current/errcodes-appendix.html)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [API Best Practices (Stripe)](https://stripe.com/docs/api)

---

**Last updated:** PR 2 (Platform Layer Infrastructure)
**Status:** Complete for phase 2

