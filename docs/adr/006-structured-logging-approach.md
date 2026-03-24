# ADR 006: Structured Logging Approach

## Status
**Accepted**

## Context

Unstructured logging makes debugging and monitoring difficult:
- Manual string parsing in logs makes searching hard
- No consistent way to trace requests through the system
- Difficult to aggregate logs across microservices
- Operations team lacks machine-readable context for alerting

We need structured logging that integrates with observability tools and correlates requests end-to-end.

## Decision

We use **Go's standard `log/slog` package** with structured logging:

1. **JSON handler** — All logs output as JSON for easy parsing and ingestion into logging systems
   - Debug level in "dev" environment
   - Info level in production

2. **Per-request logger** — `RequestLogger` middleware creates a request-scoped logger enriched with:
   - `request_id` — Unique identifier from `X-Request-ID` header (set by Chi)
   - `method` — HTTP method
   - `path` — Request URI
   - `remote_ip` — Client IP (with X-Forwarded-For support for proxied requests)
   - `trace_id` + `span_id` — OpenTelemetry correlation IDs

3. **Context propagation** — Logger stored in `context.Context` via `logging.WithContext()` and retrieved via `logging.FromContext()`

4. **OTel enrichment** — `WithTraceID()` enriches the logger with trace/span IDs from `context.Context` if a span exists

## Consequences

### ✅ Positive
* **Structured format** — JSON output integrates with log aggregation systems (ELK, Datadog, etc.)
* **Request correlation** — trace_id links logs across multiple services and layers
* **Built-in stdlib** — No external dependencies; ships with Go
* **Performance** — slog is optimized for production logging with minimal allocation
* **Context-aware** — Automatic enrichment with request metadata and trace IDs throughout the call stack

### ⚠️ Negative
* **No batteries included** — slog is minimal; custom handlers needed for special formatting (e.g., filtering PII)
* **Manual context passing** — Developers must pass context through function calls; easy to miss a layer
* **No automatic sampling** — High-traffic endpoints may produce very large logs; would need sampling strategy
* **Fixed structure** — Adding new fields to every log requires middleware changes

## Alternatives Considered

* **Unstructured text logging (fmt.Printf)** — Rejected because logs aren't machine-parseable; defeats monitoring

* **Third-party library (Zap, Logrus)** — Rejected because slog is now stdlib and eliminates external dependencies. Zap was overkill for our use case.

* **Structured logging without OTel correlation** — Rejected because trace_id is critical for debugging distributed issues

## References

* `internal/platform/logging/logging.go` — Logger setup and context helpers
* `internal/platform/middleware/logger.go` — HTTP middleware that creates per-request loggers
* [Go blog: Structured logging with slog](https://go.dev/blog/slog)
* ADR-009: OpenTelemetry Observability (related)
