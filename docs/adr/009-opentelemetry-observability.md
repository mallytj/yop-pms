# ADR 009: OpenTelemetry Observability

## Status
**Accepted**

## Context

As the system grows, we need visibility into:
- Request latency across the stack (HTTP handlers, database queries, cache operations)
- Where time is being spent (which endpoints are slow)
- Distributed trace correlation across services
- Performance bottlenecks and dependencies

Simple logging isn't enough; we need distributed tracing to understand request flows.

## Decision

We integrate **OpenTelemetry (OTel)** for end-to-end tracing:

1. **Trace provider setup** — `otel.Setup()` initializes the global tracer provider:
   - Exports to OTLP endpoint (gRPC or HTTP, auto-detected)
   - No-op tracer if endpoint is empty (development mode)
   - Flushes traces on shutdown

2. **Adaptive sampling**:
   - Development ("dev"): Always sample all traces for visibility
   - Production: Parent-based sampling with 10% ratio to limit overhead

3. **Service metadata** — Traces tagged with:
   - `service.name` — Application identifier
   - `service.version` — Deployment version for tracking regressions

4. **Request correlation** — Logger automatically enriched with OTel trace/span IDs:
   - Logs output `trace_id` and `span_id` fields
   - Links logs to distributed traces for full context

5. **Pluggable exporter** — Supports both HTTP and gRPC OTLP:
   - HTTP first (common for dev/cloud setups)
   - Fallback to gRPC if HTTP unavailable

6. **Middleware integration** — Chi router and database drivers auto-instrumented:
   - HTTP spans created automatically (method, path, status code)
   - Database spans created by pgx driver

## Consequences

### ✅ Positive
* **End-to-end visibility** — Trace entire request from HTTP handler through database
* **Performance insights** — Identify bottlenecks by trace latency
* **Production ready** — Traces sent to observability backend (Jaeger, Datadog, etc.)
* **Log correlation** — trace_id in logs links to distributed traces
* **Flexible sampling** — Reduces overhead in production while maintaining dev visibility
* **No handler changes** — Instrumentation transparent; Chi and pgx auto-instrument

### ⚠️ Negative
* **Infrastructure dependency** — OTLP endpoint required for traces (no-op if unavailable, but loses visibility)
* **Network overhead** — Trace export adds latency and bandwidth; sampling is critical
* **Learning curve** — Distributed tracing concepts (trace_id, span_id, baggage) unfamiliar to some developers
* **10% sampling may miss rare issues** — Some edge cases won't be traced in production
* **Storage cost** — Observability backend storage scales with trace volume
* **Cold start overhead** — Trace provider initialization on first request adds latency

## Alternatives Considered

* **No distributed tracing** — Rejected because:
  - Logs alone can't reconstruct request flow across services
  - Debugging slow requests becomes guesswork
  - Operations team has no visibility

* **Custom trace correlation via headers** — Rejected because:
  - Reinventing OTel; less standardized
  - Harder to integrate with observability tools
  - OTel is industry standard

* **Always-on 100% sampling** — Rejected because:
  - Produces excessive traces in production
  - Significant network and storage cost
  - 10% sampling sufficient for debugging while managing cost

## References

* `internal/platform/otel/otel.go` — Tracer provider setup
* `internal/platform/middleware/logger.go` — Logger enrichment with trace IDs
* `internal/platform/logging/logging.go` — `WithTraceID()` function
* [OpenTelemetry Go docs](https://opentelemetry.io/docs/instrumentation/go/)
* [OTLP Protocol](https://opentelemetry.io/docs/reference/specification/protocol/)
* ADR-006: Structured Logging (related; logs enriched with trace IDs)
