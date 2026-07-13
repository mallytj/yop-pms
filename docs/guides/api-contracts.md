# API Contracts

Endpoint schemas (request/response shapes, status codes per route) live in the
generated OpenAPI spec rendered at `/scalar` (or `/swagger/index.html` until
YOP-32 lands). This guide covers **cross-cutting contracts** that OpenAPI
cannot express.

## Idempotency-Key

All `POST` and `PATCH` requests must include an `Idempotency-Key` header.

**Format:** Client-generated unique string. UUID recommended. Opaque to server.
Case-sensitive.

**Behaviour:**

- **Missing** → `400 BAD_REQUEST`
- **First request** → Reserve key, process, cache response (status + body) for
  24 hours
- **Duplicate key, same request** → Return cached response
- **Duplicate key, original still processing** → Brief wait, then return cached
  response; if still in-flight after wait → `409 CONFLICT`
- **Duplicate key, different request** (different method, path, body, or
  principal) → `409 CONFLICT`
- **TTL** → 24 hours

**Key scope:** Never reuse a key across different methods, paths, request
bodies, or authenticated principals.

See ADR-007.

## Timestamps

All timestamps are ISO 8601 with timezone. Server emits UTC (`Z` suffix).

```
2026-03-15T10:30:00Z
2026-03-15T10:30:00+05:30
2026-03-15T10:30:00-08:00
```

**Standard fields on every resource:**

| Field        | Description                      |
| ------------ | -------------------------------- |
| `created_at` | When resource was created        |
| `updated_at` | When resource was last modified  |
| `deleted_at` | When resource was soft-deleted   |

## CORS

**Allowed request headers:**

```
Accept
Authorization
Content-Type
Idempotency-Key
```

**Allowed methods:** `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`

**Allowed origins:** Configured via `ALLOWED_ORIGINS` env var
(space-separated list).

## Best Practices for API Consumers

1. **Always include `Idempotency-Key` for POST/PATCH** — UUID recommended
2. **One key per logical request** — Reuse only for retries of the same
   method, path, body, and principal
3. **Handle 409 for idempotency conflicts** — Reused key for different request
   or long-running in-flight returns Conflict
4. **Handle all error codes** — Don't assume 200/201
5. **Use timestamps as-is** — Already timezone-aware
6. **Exponential backoff on 5xx and 429** — Retry with jitter
7. **Log error responses** — Include `code` + `message` in logs
