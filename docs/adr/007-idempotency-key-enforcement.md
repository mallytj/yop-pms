# ADR 007: Idempotency Key Enforcement

## Status

**Accepted**

## Context

In distributed systems, network retries can cause duplicate requests:

- Client doesn't know if the first request succeeded (network timeout)
- Retries the request, causing unintended side effects
- Example: POST /bookings retried → two bookings created instead of one

For state-changing operations (POST, PATCH), we need idempotency guarantees so clients can retry safely.

The most important idempotency security concern is to prevent duplicate payments in the future.
However, covering all of the POST and PATCH requests with idempotency middleware is a good start.

## Decision

We enforce idempotency via middleware for POST and PATCH requests:

1. **Idempotency-Key header required** — Clients must include `Idempotency-Key` header (UUID recommended) for POST/PATCH
   - Missing header → 400 Bad Request

2. **Redis-backed response caching** — On first request:
   - Capture the response (status, headers, body)
   - Store in Redis with key: `idempotency:{idempotency-key}`
   - TTL: 24 hours (sufficient for retries, prevents unbounded growth)

3. **Cache hit returns cached response** — If the same Idempotency-Key appears:
   - Retrieve cached response from Redis
   - Return immediately without executing the handler
   - Client sees identical response every time

4. **Only cache 2xx responses** — Failures (4xx, 5xx) are not cached to avoid hiding transient errors

5. **Fail open on Redis unavailability** — If Redis is down:
   - Log warning but allow request through (availability over perfect idempotency)
   - Prevents cascading failures

## Consequences

### ✅ Positive

- **Retry safety** — Clients can safely retry POST/PATCH without worrying about duplicates
- **Simple contract** — Single header requirement; easy for clients to understand
- **Transparent** — Handlers don't need to change; middleware handles everything
- **24h TTL** — Prevents unbounded cache growth while allowing typical retry windows
- **Graceful degradation** — Redis failure doesn't block requests

### ⚠️ Negative

- **Client discipline required** — Clients must generate unique Idempotency-Keys per request; a predictable key defeats the purpose
- **Header serialization overhead** — Response capture (headers, body, status) serialized to JSON and stored in Redis
- **24h TTL assumption** — Some clients may retry after 24h; those requests won't be idempotent (acceptable trade-off)
- **Body size limit** — Very large response bodies consume significant Redis memory (unlikely in practice for our API)
- **Requires Redis** — Idempotency depends on Redis availability; local-only deployments can't use this

## Alternatives Considered

- **Database deduplication** — Store request fingerprints in the database, check before inserting. Rejected because:
  - More complex to implement (extra queries per write)
  - Harder to clean up stale entries
  - Database consistency becomes critical

- **Client-side retry logic only** — Rejected because:
  - Shifts responsibility to client; some clients won't implement correctly
  - Doesn't solve the duplicate request problem

- **Outbox pattern** — Rejected for MVP because:
  - More complex (requires background worker)
  - Unnecessary overhead for current scale
  - Can adopt later if idempotency requirements grow

## References

- `internal/platform/middleware/idempotency.go` — Middleware implementation
- [Stripe Idempotency Guide](https://stripe.com/docs/api/idempotent_requests) — Industry standard reference
- [HTTP RFC 9110: Idempotent Methods](https://www.rfc-editor.org/rfc/rfc9110#name-overview)
