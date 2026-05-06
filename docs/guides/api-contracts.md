# API Contracts & Response Formats

This document defines the standard formats for all API responses and errors in Yop PMS.

## Table of Contents

- [Success Response Format](#success-response-format)
- [Error Response Format](#error-response-format)
- [HTTP Status Codes](#http-status-codes)
- [Idempotency](#idempotency)
- [Pagination](#pagination)
- [Timestamps](#timestamps)

---

## Success Response Format

All successful responses are **direct JSON objects** (no envelope). The HTTP status code indicates success.

### 2xx Success

```http
GET /v1/properties/prop-123
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Sunset Resort",
  "city": "Miami",
  "created_at": "2026-01-15T10:30:00Z"
}
```

### 201 Created

```http
POST /v1/reservations
HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "property_id": "prop-123",
  "guest_id": "guest-456",
  "check_in": "2026-04-01T15:00:00Z",
  "check_out": "2026-04-05T10:00:00Z",
  "status": "confirmed",
  "created_at": "2026-03-15T10:30:00Z"
}
```

### 204 No Content

```http
DELETE /v1/reservations/res-789
HTTP/1.1 204 No Content
```

---

## Error Response Format

All errors follow a **consistent envelope** backed by the `internal/platform/apierror` package. This format ensures that both human users and automated systems can understand and resolve issues.

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "code": "BAD_REQUEST",
  "message": "property_id is required",
  "status": 400,
  "suggestions": [
    "Ensure the property_id is provided in the request body",
    "Check if the property_id is a valid UUID"
  ]
}
```

### Fields

| Field         | Type     | Description                                                          |
| ------------- | -------- | -------------------------------------------------------------------- |
| `code`        | string   | Machine-readable error code (e.g., `BAD_REQUEST`, `CONFLICT`)        |
| `message`     | string   | Human-readable error message suitable for displaying to users        |
| `status`      | int      | HTTP status code (for convenience; also in response headers)         |
| `suggestions` | string[] | (Optional) Actionable steps to resolve the error                     |

### Error Codes

The `apierror` package provides standard sentinel errors:

| Code                   | HTTP Status | Meaning                                          |
| ---------------------- | ----------- | ------------------------------------------------ |
| `BAD_REQUEST`          | 400         | Malformed request (invalid JSON, missing fields) |
| `NOT_FOUND`            | 404         | Resource doesn't exist                           |
| `CONFLICT`             | 409         | Resource already exists or dates overlap         |
| `UNPROCESSABLE_ENTITY` | 422         | Request data violates business rules             |
| `INTERNAL_ERROR`       | 500         | Server error (unexpected failure)                |

---

## Go Usage (`internal/platform/apierror`)

When building handlers or services, use the `apierror` package to maintain consistency.

### 1. Basic Sentinel
```go
return apierror.ErrNotFound
```

### 2. Custom Message
```go
return apierror.ErrBadRequest.WithMessage("check_in date cannot be in the past")
```

### 3. Adding Suggestions
```go
return apierror.ErrConflict.
    WithMessage("room is already occupied").
    WithSuggestions([]string{
        "Select a different room",
        "Change the check-in period"
    })
```

### 4. Automatic DB Mapping
The package automatically maps PostgreSQL errors (SQLSTATE) to appropriate `APIError` objects:
- `UniqueViolation` → `CONFLICT`
- `CheckViolation` → `UNPROCESSABLE_ENTITY`
- `ExclusionViolation` → `CONFLICT` (e.g. overlapping dates)

```go
if err := store.Create(ctx, data); err != nil {
    return apierror.MapStoreError(err)
}
```

---

## HTTP Status Codes

### 2xx Success

| Status | Meaning                                 |
| ------ | --------------------------------------- |
| 200    | OK — Request succeeded, body included   |
| 201    | Created — Resource created successfully |
| 204    | No Content — Success, no body           |

### 4xx Client Error

| Status | Error Code             | Meaning                                |
| ------ | ---------------------- | -------------------------------------- |
| 400    | `BAD_REQUEST`          | Malformed request or invalid JSON      |
| 401    | `UNAUTHORIZED`         | Missing or invalid authentication      |
| 403    | `FORBIDDEN`            | Authenticated but not authorized       |
| 404    | `NOT_FOUND`            | Resource doesn't exist                 |
| 409    | `CONFLICT`             | Duplicate unique constraint or overlap |
| 422    | `UNPROCESSABLE_ENTITY` | Data fails business rule validation    |
| 429    | `RATE_LIMITED`         | Too many requests                      |

### 5xx Server Error

| Status | Error Code            | Meaning                     |
| ------ | --------------------- | --------------------------- |
| 500    | `INTERNAL_ERROR`      | Server error (log details)  |
| 503    | `SERVICE_UNAVAILABLE` | Dependency down (DB, Redis) |

---

## Idempotency

All POST and PATCH requests must include the `Idempotency-Key` header:

```http
POST /v1/reservations
Content-Type: application/json
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000

{
  "property_id": "prop-123",
  "guest_id": "guest-456",
  ...
}
```

### Idempotency Behavior

- **Missing key** → 400 Bad Request
- **First request** → Reserve key, process normally, cache response (status + body)
- **Duplicate key for the same request** → Return cached response (same status + body)
- **Duplicate key while the original request is still processing** → Wait briefly for the cached response, then return it; if it is still processing after the wait, return 409 Conflict
- **Duplicate key for a different request** → 409 Conflict
- **TTL** → Cached responses stored for 24 hours

### Idempotency Key Format

- Must be a unique string (UUID recommended)
- Case-sensitive
- Client-generated (server doesn't generate keys)
- Can be any format; opaque to API
- Must not be reused across different methods, paths, request bodies, or authenticated principals

---

## Pagination

List endpoints support cursor-based pagination (offset/limit):

```http
GET /v1/properties?limit=10&offset=0
HTTP/1.1 200 OK
Content-Type: application/json

{
  "items": [
    { "id": "prop-1", "name": "Resort A" },
    { "id": "prop-2", "name": "Resort B" }
  ],
  "pagination": {
    "limit": 10,
    "offset": 0,
    "total": 42,
    "has_more": true
  }
}
```

### Pagination Fields

| Field      | Type    | Description                       |
| ---------- | ------- | --------------------------------- |
| `limit`    | int     | Requested page size (default: 10) |
| `offset`   | int     | Number of items skipped           |
| `total`    | int     | Total count of items              |
| `has_more` | boolean | Whether more items exist          |

### Query Parameters

- `limit` — Items per page (max 100, default 10)
- `offset` — Number of items to skip (default 0)

```http
GET /v1/reservations?limit=20&offset=40
```

---

## Timestamps

All timestamps are ISO 8601 format with timezone:

```
2026-03-15T10:30:00Z        (UTC)
2026-03-15T10:30:00+05:30   (IST)
2026-03-15T10:30:00-08:00   (PST)
```

### Timestamp Fields

Standard fields on all resources:

| Field        | Type      | Description                    |
| ------------ | --------- | ------------------------------ |
| `created_at` | timestamp | When resource was created      |
| `updated_at` | timestamp | When resource was last updated |
| `deleted_at` | timestamp | When resource was soft-deleted |

---

## Request/Response Examples

### Example 1: Create Reservation (Success)

```http
POST /v1/reservations
Content-Type: application/json
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000

{
  "property_id": "prop-123",
  "guest_id": "guest-456",
  "check_in": "2026-04-01",
  "check_out": "2026-04-05"
}

---

HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "res-789",
  "property_id": "prop-123",
  "guest_id": "guest-456",
  "check_in": "2026-04-01T15:00:00Z",
  "check_out": "2026-04-05T10:00:00Z",
  "status": "confirmed",
  "created_at": "2026-03-15T10:30:00Z"
}
```

### Example 2: Dates Overlap Error

```http
POST /v1/reservations
Content-Type: application/json
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440001

{
  "property_id": "prop-123",
  "guest_id": "guest-789",
  "check_in": "2026-04-02",
  "check_out": "2026-04-06"
}

---

HTTP/1.1 409 Conflict
Content-Type: application/json

{
  "code": "CONFLICT",
  "message": "the dates overlap with an existing reservation",
  "status": 409
}
```

### Example 3: Invalid JSON

```http
POST /v1/reservations
Content-Type: application/json

{
  invalid json
}

---

HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "code": "BAD_REQUEST",
  "message": "request body is not valid JSON or contains unknown fields",
  "status": 400
}
```

### Example 4: Missing Required Field

```http
POST /v1/reservations
Content-Type: application/json
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440002

{
  "property_id": "prop-123"
}

---

HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "code": "BAD_REQUEST",
  "message": "guest_id is required",
  "status": 400
}
```

### Example 5: Missing Idempotency Key

```http
POST /v1/reservations
Content-Type: application/json

{
  "property_id": "prop-123",
  "guest_id": "guest-456"
}

---

HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "code": "BAD_REQUEST",
  "message": "Idempotency-Key header is required",
  "status": 400
}
```

### Example 6: List with Pagination

```http
GET /v1/properties?limit=10&offset=0

---

HTTP/1.1 200 OK
Content-Type: application/json

{
  "items": [
    {
      "id": "prop-1",
      "name": "Sunset Resort",
      "city": "Miami",
      "created_at": "2026-01-15T10:30:00Z"
    },
    {
      "id": "prop-2",
      "name": "Mountain Lodge",
      "city": "Denver",
      "created_at": "2026-02-20T14:15:00Z"
    }
  ],
  "pagination": {
    "limit": 10,
    "offset": 0,
    "total": 42,
    "has_more": true
  }
}
```

---

## Best Practices for API Consumers

1. **Always include `Idempotency-Key` for POST/PATCH** — UUID is recommended
2. **Generate one key per logical request** — Reuse the same key only for retries of that exact method, path, body, and authenticated principal
3. **Handle 409 for idempotency conflicts** — A reused key for a different request or a long-running in-flight request returns Conflict
4. **Handle all error codes** — Don't assume 200/201 for all requests
5. **Use timestamps as-is** — Already timezone-aware
6. **Implement exponential backoff** — For 5xx errors and rate limiting (429)
7. **Log error responses** — Include `code` + `message` in logs for debugging

---

## CORS & Headers

Allowed headers in requests:

```
Accept
Authorization
Content-Type
Idempotency-Key
```

Allowed methods:

```
GET
POST
PUT
DELETE
OPTIONS
```

Allowed origins: Configured via `ALLOWED_ORIGINS` environment variable (space-separated list).
