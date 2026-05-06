# Platform Layer Guide

This guide explains how to use the platform packages in `internal/platform/`
when building domain handlers.

## Table of Contents

- [Error Handling](#error-handling)
- [Logging](#logging)
- [JSON Response Encoding](#json-response-encoding)
- [Caching](#caching)
- [Event Listening](#event-listening)
- [Outbox Worker](#outbox-worker)
- [Full Example Handler](#full-example-handler)

---

## Error Handling

All errors returned to clients must go through `apierror.MapPostgresError()` to
ensure consistent error responses.

### Using Sentinel Errors (Domain Logic)

For errors that occur in business logic (not from the database):

```go
import "github.com/lexxcode1/yop-pms/internal/platform/apierror"

// In a domain handler
func (h *Handler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
    propertyID := chi.URLParam(r, "id")

    // Validate input
    if propertyID == "" {
        json.WriteError(w, r, apierror
            .ErrBadRequest
            .WithMessage(
              "property ID is required"
            ))
        return
    }

    // If resource doesn't exist
    if !propertyExists {
        json.WriteError(w, r, apierror.ErrNotFound)
        return
    }
}
```

### Adding Suggestions

```go
// Add suggestions to the error response
err := apierror.ErrConflict.WithMessage(
    "property with name 'Sunset Resort' already exists",
).WithSuggestions([]string{"Try a different name", "Check for duplicates"})
json.WriteError(w, r, err)
```

### Handling Database Errors

When database operations fail, map the error automatically:

```go
err := h.store.UpdateProperty(ctx, propertyID, data)
if err != nil {
    json.WriteError(w, r, err)  // Automatically maps to apierror
    return
}
```

The database error is mapped based on PostgreSQL SQLSTATE:

- `23505` (Unique Violation) → 409 Conflict
- `23503` (Foreign Key) → 400 Bad Request
- `23514` (Check Violation) → 422 Unprocessable Entity
- `P0001` (PL/pgSQL RAISE) → 422 (extracts error detail)

### Custom Error Messages

Derive from sentinels without mutation:

```go
// ✗ WRONG: Mutates the sentinel
apierror.ErrConflict.Message = "custom message"

// ✓ CORRECT: Creates new error with custom message
err := apierror.ErrConflict.WithMessage("property with name 'Sunset Resort' already exists")
json.WriteError(w, r, err)
```

---

## Logging

Always retrieve the logger from the request context. The `RequestLogger` middleware injects a per-request logger with metadata.

```go
import "github.com/lexxcode1/yop-pms/internal/platform/logging"

func (h *Handler) GetProperty(w http.ResponseWriter, r *http.Request) {
    logger := logging.FromContext(r.Context())

    propertyID := chi.URLParam(r, "id")
    logger.Info("fetching property", "property_id", propertyID)

    property, err := h.store.GetProperty(r.Context(), propertyID)
    if err != nil {
        logger.Error("failed to fetch property", "error", err)
        json.WriteError(w, r, err)
        return
    }

    logger.Debug("property fetched successfully", "name", property.Name)
    json.WriteJSON(w, http.StatusOK, property)
}
```

### Logger Context

The per-request logger automatically includes:

- `request_id` — Unique ID for the request
- `method` — HTTP method (GET, POST, etc.)
- `path` — Request path
- `remote_ip` — Client IP address
- `trace_id` — OpenTelemetry trace ID (if available)
- `span_id` — OpenTelemetry span ID (if available)

### Never Use `slog.Default()`

```go
// ✗ WRONG: Loses request context
slog.Info("processing request")  // No trace_id, no request_id

// ✓ CORRECT: Uses per-request logger with full context
logger := logging.FromContext(r.Context())
logger.Info("processing request")  // Includes trace_id, request_id, etc.
```

---

## JSON Response Encoding

Use `json.WriteJSON()` for success responses and `json.WriteError()` for errors.

### Success Responses

```go
import yopjson "github.com/lexxcode1/yop-pms/internal/platform/json"

func (h *Handler) ListProperties(w http.ResponseWriter, r *http.Request) {
    properties, err := h.store.ListProperties(r.Context())
    if err != nil {
        yopjson.WriteError(w, r, err)
        return
    }

    // ✓ Automatically sets Content-Type: application/json
    json.WriteJSON(w, http.StatusOK, properties)
}
```

### Error Responses

```go
// Database error (automatically mapped to apierror)
err := h.store.GetProperty(ctx, id)
if err != nil {
    json.WriteError(w, r, err)  // Handles mapping automatically
    return
}

// Direct error response
json.WriteError(w, r, apierror.ErrBadRequest.WithMessage("invalid date range"))
```

### Parsing Request Bodies

```go
var req struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

err := json.ReadJSON(r, &req)
if err != nil {
    // err is already an APIError with status 400
    json.WriteError(w, r, err)
    return
}
```

---

## Caching

Cache belongs in the **service layer**, not handlers. Handlers know nothing about how data is fetched — that's the service's job. The cache client is injected into services via `NewService(store, cache)`.

### Basic Operations

```go
import "github.com/lexxcode1/yop-pms/internal/platform/cache"

func (h *Handler) GetAvailability(w http.ResponseWriter, r *http.Request) {
    propertyID := helpers.GetPropertyIDFromContext(r.Context()) // Helper not implemented yet...
    date := r.URL.Query().Get("date")

    cacheKey := fmt.Sprintf("availability:%s:%s", propertyID, date)

    var availability Availability
    err := h.app.cache.Get(r.Context(), cacheKey, &availability)

    // Cache hit
    if err == nil {
        return json.WriteJSON(w, http.StatusOK, availability)
    }

    // Cache miss or error — fetch from database
    availability, err = h.store.GetAvailability(r.Context(), propertyID, date)
    if err != nil {
        json.WriteError(w, r, err)
        return
    }

    // Cache for 1 hour
    h.app.cache.Set(r.Context(), cacheKey, availability, 1*time.Hour)

    json.WriteJSON(w, http.StatusOK, availability)
}
```

### Read-Through Cache

Use `GetOrSet()` for the common case of "get from cache, or load and cache":

```go
var availability Availability
cacheKey := fmt.Sprintf("availability:%s:%s", propertyID, date)

err := h.app.cache.GetOrSet(
    r.Context(),
    cacheKey,
    &availability,
    1*time.Hour,  // TTL
    func(ctx context.Context) (any, error) {
        return h.store.GetAvailability(ctx, propertyID, date)
    },
)

if err != nil {
    json.WriteError(w, r, err)
    return
}

json.WriteJSON(w, http.StatusOK, availability)
```

### Cache Invalidation

Invalidate cache when data changes:

```go
func (h *Handler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
    propertyID := chi.URLParam(r, "id")

    // ... update logic ...

    // Invalidate all caches for this property
    h.app.cache.Invalidate(r.Context(), fmt.Sprintf("availability:%s:*", propertyID))

    json.WriteJSON(w, http.StatusOK, updatedProperty)
}
```

### Key Naming Convention

Use hierarchical keys with colons as separators:

```text
yop:availability:property-uuid:2026-03-15
yop:pricing:property-uuid:room-type-id
yop:guest:guest-uuid:details
```

Pattern invalidation uses wildcards:

```go
cache.Invalidate(ctx, "yop:availability:property-uuid:*")  // All dates for property
cache.Invalidate(ctx, "yop:availability:*:2026-03-15")     // All properties for date
cache.Invalidate(ctx, "yop:*")                              // Nuke everything
```

### Cache Miss Error

The cache library uses a custom error type:

```go
import "github.com/lexxcode1/yop-pms/internal/platform/cache"

var data MyData
err := h.app.cache.Get(ctx, key, &data)

if err == cache.ErrCacheMiss {
    // Key doesn't exist — load from database
} else if err != nil {
    // Redis error — log and continue
}
```

### Invalidation Handlers

Cache invalidation is driven by PostgreSQL `LISTEN/NOTIFY` (see ADR-010). When a reservation changes, the database fires a notification that the events listener picks up and dispatches to registered handlers.

Handlers live in `internal/platform/cache/` alongside the cache client. Each handler receives a parsed event and decides which keys to evict — the events package has no knowledge of cache internals.

---

## Event Listening

The events listener (`internal/platform/events/`) subscribes to PostgreSQL `LISTEN/NOTIFY` channels and dispatches notifications to registered handlers. It holds a **dedicated connection** outside the pool (a connection running `LISTEN` cannot be used for queries) and reconnects automatically with exponential backoff on failure.

**Handlers are registered at startup in `main.go` — not inside domain packages.** The listener is infrastructure; wiring it to specific handlers is a startup concern.

### Registering a handler

```go
// main.go
el := events.New(cfg.DatabaseURL, logger, onReconnect)
el.On("my_channel", myHandler)
el.Start()
defer el.Stop()
```

`el.On` can be called multiple times for the same channel — all registered handlers run concurrently for each notification.

### Writing a handler

A handler is any function matching `events.Handler`:

```go
type Handler func(ctx context.Context, event Event) error
```

The `Event` carries the channel name, a timestamp, and the parsed JSON payload as `map[string]any`. Handlers should be fast and non-blocking; expensive work can be done inline since each handler runs in its own goroutine tracked by the listener's `WaitGroup`.

```go
func MyHandler(logger *slog.Logger) events.Handler {
    return func(ctx context.Context, event events.Event) error {
        // type-assert the fields you need from event.Data
        id, ok := event.Data["record_id"].(string)
        if !ok {
            return fmt.Errorf("missing record_id in payload")
        }
        // ... do work
        return nil
    }
}
```

Returning an error logs it but does not stop other handlers or crash the listener. Handlers should only return errors for payload problems (bad format) — cache or network failures should be logged and swallowed so one bad notification doesn't break the channel.

### Reconnect flush

Pass an `onReconnect` callback to `events.New` to handle the disconnect gap. During a reconnect window any number of mutations could have been missed, so the safest recovery is a full cache flush:

```go
events.New(cfg.DatabaseURL, logger, func() {
    if err := appCache.Invalidate(ctx, "yop:*"); err != nil {
        logger.Error("failed to flush cache on reconnect", "error", err)
    }
})
```

### Invalidation Events

Cache invalidation handlers live in `internal/platform/cache/` and are registered at startup in `main.go`. Each handler receives a parsed `events.Event` and evicts the keys it owns — the events package has no knowledge of cache internals.

```go
// internal/platform/cache/planner_invalidation.go
func NewReservationChangeHandler(c *Client, logger *slog.Logger) events.Handler {
    return func(ctx context.Context, event events.Event) error {
        propertyID, _ := event.Data["property_id"].(string)
        c.Invalidate(ctx, fmt.Sprintf("yop:planner:%s:*", propertyID))
        return nil
    }
}
```

---

## Outbox Worker

Background tasks (emails, webhooks) must not block the request path. The outbox worker (`internal/platform/worker`) decouples them safely: a task row is inserted into `internal.outbox_events` as part of the same database transaction as the domain mutation. Even if the process crashes immediately after, the row survives and the worker delivers it on restart.

**Rule:** anything that calls an external service (SMTP, webhook, push notification) goes through the outbox, never inline in a handler.

### Enqueueing an event

Call `worker.Enqueue` from a service method, passing the SQLC `*store.Queries` for the current request and a typed payload struct:

```go
import "github.com/lexxcode1/yop-pms/internal/platform/worker"

func (s *Service) CreateReservation(ctx context.Context, req CreateReservationRequest) (Reservation, error) {
    reservation, err := s.store.CreateReservation(ctx, ...)
    if err != nil {
        return Reservation{}, apierror.MapPostgresError(err)
    }

    // Enqueue for async delivery — never blocks the request
    _ = worker.Enqueue(ctx, s.store, worker.EventConfirmationEmail, worker.ConfirmationEmailPayload{
        ReservationID: reservation.ID.String(),
        GuestEmail:    req.GuestEmail,
        GuestName:     req.GuestName,
        PropertyName:  req.PropertyName,
    })

    return reservation, nil
}
```

Use `worker.EnqueueAt` to schedule delivery at a future time (e.g. pre-arrival emails 24 hours before check-in):

```go
worker.EnqueueAt(ctx, s.store, worker.EventPreArrivalEmail, worker.PreArrivalEmailPayload{
    ReservationID: reservation.ID.String(),
    GuestEmail:    req.GuestEmail,
    GuestName:     req.GuestName,
    PropertyName:  req.PropertyName,
    CheckIn:       req.CheckIn,
}, req.CheckIn.Add(-24*time.Hour))
```

### Event types and payload structs

Event types use dot-namespaced constants. Each has a corresponding typed payload struct in `internal/platform/worker/payloads.go`:

| Constant | Payload struct |
|---|---|
| `worker.EventConfirmationEmail` | `worker.ConfirmationEmailPayload` |
| `worker.EventPreArrivalEmail` | `worker.PreArrivalEmailPayload` |
| `worker.EventCancellationEmail` | `worker.CancellationEmailPayload` |

To add a new event type: define a constant and a payload struct in `payloads.go`, then register a handler in `main.go`.

### Handler registration

Handlers are registered at startup in `cmd/server/main.go` — not inside domain packages. The worker is infrastructure; wiring it to specific domain logic is a startup concern.

```go
// cmd/server/main.go
outboxWorker.Register(worker.EventConfirmationEmail, smtp.HandleConfirmation(smtpClient))
outboxWorker.Register(worker.EventPreArrivalEmail,   smtp.HandlePreArrival(smtpClient))
```

A handler receives the raw JSONB payload and unmarshals it into its own typed struct:

```go
func HandleConfirmation(client *SMTPClient) worker.Handler {
    return func(ctx context.Context, payload json.RawMessage) error {
        var p worker.ConfirmationEmailPayload
        if err := json.Unmarshal(payload, &p); err != nil {
            return fmt.Errorf("parse payload: %w", err)
        }
        return client.SendConfirmation(ctx, p.GuestEmail, p.GuestName, p.ReservationID)
    }
}
```

Returning an error triggers a retry with exponential backoff (`min(2^n, 1800)` seconds, default 3 retries). After exhausting retries the event is dead-lettered (`status = 'failed'`).

### Dead-letter channel

When an event exhausts its retries the worker emits a `pg_notify` on `outbox_dead_lettered`:

```json
{ "id": "...", "event_type": "smtp.confirmation", "last_error": "dial tcp: connection refused" }
```

Subscribe to this channel via the event listener to surface recurring failures in monitoring or alert the property owner:

```go
eventListener.On("outbox_dead_lettered", func(ctx context.Context, e events.Event) error {
    logger.Error("outbox event dead-lettered",
        "id",         e.Data["id"],
        "event_type", e.Data["event_type"],
        "error",      e.Data["last_error"],
    )
    return nil
})
```

### Querying stuck or failed events

```sql
-- All failed events in the last 24 hours
SELECT id, event_type, retry_count, last_error, created_at
FROM internal.outbox_events
WHERE status = 'failed'
  AND created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;

-- Events still pending (including backoff queue)
SELECT id, event_type, retry_count, process_at
FROM internal.outbox_events
WHERE status = 'pending'
ORDER BY process_at;
```

---

## Full Example Handler

The architecture has three layers. Each layer has one responsibility:

| Layer       | Responsibility                                            |
| ----------- | --------------------------------------------------------- |
| **Handler** | HTTP only — parse request, validate input, write response |
| **Service** | Business logic + caching                                  |
| **Store**   | Raw database queries (SQLC-generated)                     |

```go
package booking

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/lexxcode1/yop-pms/internal/platform/apierror"
    "github.com/lexxcode1/yop-pms/internal/platform/cache"
    "github.com/lexxcode1/yop-pms/internal/platform/logging"
    yopjson "github.com/lexxcode1/yop-pms/internal/platform/json"
    "github.com/lexxcode1/yop-pms/internal/platform/worker"
    "github.com/lexxcode1/yop-pms/internal/store"
)
// --- Service (business logic + caching) ---

type Service struct {
    store *store.Queries
    cache *cache.Client
}

func NewService(store *store.Queries, cache *cache.Client) *Service {
    return &Service{store: store, cache: cache}
}

func (s *Service) CreateReservation(ctx context.Context, req CreateReservationRequest) (Reservation, error) {
    reservation, err := s.store.CreateReservation(ctx, req.PropertyID, req.GuestID, req.CheckIn, req.CheckOut)
    if err != nil {
        // Map postgres errors (unique violation, FK, etc.) to typed APIErrors
        return Reservation{}, apierror.MapPostgresError(err)
    }

    // Invalidate availability cache — co-located with the data mutation
    s.cache.Invalidate(ctx, fmt.Sprintf("yop:availability:%s:*", req.PropertyID))

    // Enqueue confirmation email — async, never blocks the request path
    _ = worker.Enqueue(ctx, s.store, worker.EventConfirmationEmail, worker.ConfirmationEmailPayload{
        ReservationID: reservation.ID.String(),
        GuestEmail:    req.GuestEmail,
        GuestName:     req.GuestName,
        PropertyName:  req.PropertyName,
    })

    return reservation, nil
}

func (s *Service) GetReservation(ctx context.Context, id string) (Reservation, error) {
    var reservation Reservation
    err := s.cache.GetOrSet(
        ctx,
        fmt.Sprintf("yop:reservation:%s", id),
        &reservation,
        24*time.Hour,
        func(ctx context.Context) (any, error) {
            r, err := s.store.GetReservation(ctx, id)
            if err != nil {
                return Reservation{}, apierror.MapStoreError(err)
            }
            return r, nil
        },
    )
    return reservation, err
}

// --- Handler (HTTP only) ---

type Handler struct {
    service *Service
}

func NewHandler(service *Service) *Handler {
    return &Handler{service: service}
}

// CreateReservation handles POST /v1/reservations
// @Summary      Create a new reservation
// @Tags         Reservations
// @Accept       json
// @Produce      json
// @Param        body  body     CreateReservationRequest true "Reservation details"
// @Success      201   {object} Reservation
// @Failure      400   {object} apierror.APIError
// @Failure      409   {object} apierror.APIError
// @Router       /v1/reservations [post]
func (h *Handler) CreateReservation(w http.ResponseWriter, r *http.Request) {
    logger := logging.FromContext(r.Context())

    var req CreateReservationRequest
    if err := yopjson.ReadJSON(r, &req); err != nil {
        yopjson.WriteError(w, r, err)
        return
    }

    if req.PropertyID == "" || req.GuestID == "" {
        yopjson.WriteError(w, r, apierror.ErrBadRequest.WithMessage("property_id and guest_id are required"))
        return
    }

    reservation, err := h.service.CreateReservation(r.Context(), req)
    if err != nil {
        yopjson.WriteError(w, r, err)
        return
    }

    logger.Info("reservation created", "reservation_id", reservation.ID)
    yopjson.WriteJSON(w, http.StatusCreated, reservation)
}

// GetReservation handles GET /v1/reservations/:id
// @Summary      Get a reservation by ID
// @Tags         Reservations
// @Produce      json
// @Param        id   path     string true "Reservation ID"
// @Success      200  {object} Reservation
// @Failure      404  {object} apierror.APIError
// @Router       /v1/reservations/{id} [get]
func (h *Handler) GetReservation(w http.ResponseWriter, r *http.Request) {
    reservation, err := h.service.GetReservation(r.Context(), chi.URLParam(r, "id"))
    if err != nil {
        yopjson.WriteError(w, r, err)
        return
    }

    yopjson.WriteJSON(w, http.StatusOK, reservation)
}

// Routes returns the reservation routes
func (h *Handler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Post("/", h.CreateReservation)
    r.Get("/{id}", h.GetReservation)
    return r
}
```

### Wiring it up

```go
// In cmd/server/api.go
store := bookingstore.New(app.db)
service := booking.NewService(store, app.cache)
handler := booking.NewHandler(service)

r.Mount("/v1/reservations", handler.Routes())
```

---

## Middleware Order and OTel

The middleware stack is ordered as:

1. **otelchi.Middleware** — Creates root span for the request
2. **RequestLogger** — Injects per-request logger (with trace/span IDs from otelchi)
3. **RequestID** — Adds request ID to context
4. **RealIP** — Extracts client IP
5. **Recoverer** — Catches panics
6. **StripSlashes** — Normalizes paths
7. **CORS** — Handles cross-origin requests
8. **Idempotency** — Enforces atomically reserved, request-scoped idempotency keys (only on /v1)

**Important:** otelchi must run FIRST. The RequestLogger middleware calls `logging.WithTraceID()` which relies on the span created by otelchi.

---

## Testing

Test each layer in isolation — no need to mock cache in handler tests since handlers don't touch cache.

### Handler tests (HTTP concerns only)

```go
func TestCreateReservation_MissingFields(t *testing.T) {
    // Handler tests use a mock service — no cache or store needed
    handler := NewHandler(&mockService{})

    req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{}`)))
    ctx := logging.WithContext(req.Context(), logging.NewLogger("dev"))
    req = req.WithContext(ctx)

    w := httptest.NewRecorder()
    handler.CreateReservation(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }
}
```

### Service tests (caching + business logic)

```go
func TestGetReservation_CacheHit(t *testing.T) {
    // Service tests exercise cache + store interaction
    service := NewService(&mockStore{}, mockCacheClient)

    reservation, err := service.GetReservation(context.Background(), "res-1")

    // Assert store was not called (cache hit)
    // Assert reservation matches cached value
}
```

---

## FAQ

**Q: Should I use `slog.Default()` if the context doesn't have a logger?**
A: No. `logging.FromContext()` never returns nil — it falls back to `slog.Default()`. Always use `logging.FromContext()`.

**Q: What if I need a logger outside an HTTP request context?**
A: Create one with `logging.NewLogger(os.Getenv("APP_ENV"))` in that scope. Only inject via context within request handlers.

**Q: Can I cache database errors?**
A: No, don't cache errors. Cache only successful data. Errors are typically transient.

**Q: Why is the idempotency key header missing from CORS in development?**
A: It's in the CORS allowed headers. Make sure you're running the current version with the updated middleware stack.

**Q: How do I add a custom span in my handler?**
A: Use `go.opentelemetry.io/otel` directly (not yet documented; see ADR 0005 future considerations).
