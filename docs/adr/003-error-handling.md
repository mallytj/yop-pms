# ADR 003: Error Handling

## Status

**Accepted**

All handlers return `apierror.APIError{Code, Message, Status, Suggestions}`. PostgreSQL SQLSTATEs auto-map via `MapPostgresError`: `23505`→409, `23503`→400, `23514`→422, `23P01`→409, `P0001`→422. Sentinels are immutable; customise via `WithMessage()`. 4xx/5xx responses not cached.

Alternatives: Go stdlib `error` (no HTTP status), per-domain error hierarchies (over-engineered), auto-mapping middleware (hides data flow).

---

See: `internal/platform/apierror/apierror.go`, `internal/platform/helpers/db_errors.go`
