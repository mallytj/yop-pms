# ADR 004: Idempotency Key

## Status

**Accepted**

POST and PATCH require an `Idempotency-Key` header (UUID). Middleware reserves the key in Redis via `SET NX` before the handler runs, captures the response on success, and serves cached responses to retries within 24h. Concurrent duplicates with matching fingerprint wait briefly for the in-flight request; mismatched fingerprints return 409. Failures (4xx/5xx) are not cached. Redis outage fails open (warn + allow).

Alternatives: DB-stored request fingerprints (extra queries, cleanup burden), client-side retry logic (no enforcement), outbox-pattern dedupe (over-engineered for MVP).

---

See: `internal/platform/middleware/idempotency.go`
