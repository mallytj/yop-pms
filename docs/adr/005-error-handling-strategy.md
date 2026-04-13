# ADR 005: Error Handling Strategy

## Status

**Accepted**

## Context

Every API endpoint needs consistent, predictable error responses. Without a clear error handling strategy, we risk:

- Inconsistent HTTP status codes across endpoints
- Leaking implementation details in error messages
- Difficulty debugging production issues
- Poor client UX when handling errors

Database errors (constraints, validation failures, business rule violations) need to map to appropriate HTTP responses automatically.

## Decision

We implement a centralized error handling system:

1. **APIError struct** — All errors conform to a standard shape:

   ```go
   type Suggestions []string

   type APIError struct {
       Code        string        `json:"code"` // Machine-readable code: "NOT_FOUND", "CONFLICT", etc.
       Message     string        `json:"message"` // User-friendly message
       Status      int           `json:"-"` // HTTP status code
       Suggestions Suggestions `json:"suggestions,omitempty"` // Optional suggestions for the client
   }
   ```

2. **Sentinel errors** — Pre-defined errors for common cases (NotFound, BadRequest, Conflict, etc.) that can be customized via `WithMessage()` without mutation

3. **PostgreSQL error mapping** — `MapPostgresError()` maps database-specific SQLSTATEs to API errors:
   - `23505` (Unique Violation) → 409 Conflict
   - `23503` (Foreign Key Violation) → 400 Bad Request
   - `23514` (Check Violation) → 422 Unprocessable Entity
   - `23P01` (Exclusion Violation) → 409 Conflict
   - `P0001` (PL/pgSQL RAISE) → 422 with custom detail message

4. **Immutable customization** — Use `WithMessage()` to derive custom errors from sentinels without mutation, preserving Code and Status
5. **Suggestions** — Add suggestions to the error response to help the client understand what went wrong and how to fix it

## Consequences

### ✅ Positive

- **Consistency** — All endpoints return the same error shape, making client error handling predictable
- **Type safety** — Go type system ensures all errors are handled
- **Database intelligence** — SQLSTATE codes automatically mapped to correct HTTP status
- **Custom messages** — Extract meaningful details from database errors (e.g., violation details from CHECK constraints)
- **Immutability** — Sentinels remain unchanged; customization creates new instances

### ⚠️ Negative

- **Required discipline** — Developers must remember to use `apierror.MapPostgresError()` in handlers; easy to forget
- **No automatic wrapping** — Errors must flow through the apierror package explicitly
- **Limited error chains** — Go's error wrapping (`%w`) is one-way; we don't unwrap database errors automatically (though we could via `errors.As()`)

## Alternatives Considered

- **Standard Go errors** — Rejected because they don't carry HTTP status codes; handlers would need switch statements to map errors to status codes, creating inconsistency

- **Strongly-typed error hierarchy (custom types per domain)** — Rejected as over-engineered; a single APIError with codes is simpler and sufficient

- **Error middleware auto-mapping** — Rejected because it shifts responsibility away from handlers; explicit `MapPostgresError()` calls make data flow clearer

## References

- `internal/platform/apierror/apierror.go` — Implementation
- `internal/platform/helpers/db_errors.go` — Error code constants
- RFC 7231 (HTTP Semantics) — Status code definitions
