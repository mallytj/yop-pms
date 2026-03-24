# Testing Guide

This guide explains how to test Yop PMS handlers and business logic.

## Table of Contents

- [Unit Testing](#unit-testing)
- [Integration Testing](#integration-testing)
- [Testing with Mocks](#testing-with-mocks)
- [Testing Handlers](#testing-handlers)
- [Testing Database Code](#testing-database-code)
- [Test Utilities](#test-utilities)

---

## Unit Testing

Unit tests focus on individual functions in isolation.

### Simple Unit Test

```go
// internal/booking/availability_test.go

package booking

import (
    "testing"
    "time"
)

func TestIsAvailable_NoConflict(t *testing.T) {
    service := &AvailabilityService{}

    available, err := service.IsAvailable(
        "prop-1",
        time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
        time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
    )

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if !available {
        t.Error("expected available to be true")
    }
}

func TestIsAvailable_WithConflict(t *testing.T) {
    service := &AvailabilityService{}

    available, err := service.IsAvailable(
        "prop-1",
        time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
        time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
    )

    if !errors.Is(err, ErrDatesUnavailable) {
        t.Errorf("expected ErrDatesUnavailable, got %v", err)
    }

    if available {
        t.Error("expected available to be false")
    }
}
```

### Table-Driven Tests

For testing multiple scenarios:

```go
func TestPriceCalculation(t *testing.T) {
    tests := []struct {
        name          string
        roomType      string
        nights        int
        season        string
        expectedPrice int
        shouldError   bool
    }{
        {
            name:          "standard room in low season",
            roomType:      "standard",
            nights:        3,
            season:        "low",
            expectedPrice: 30000, // 100 USD * 3 nights
            shouldError:   false,
        },
        {
            name:          "deluxe room in peak season",
            roomType:      "deluxe",
            nights:        2,
            season:        "peak",
            expectedPrice: 40000, // 200 USD * 2 nights
            shouldError:   false,
        },
        {
            name:          "invalid room type",
            roomType:      "mansion",
            nights:        1,
            season:        "low",
            expectedPrice: 0,
            shouldError:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := NewPricingService()
            price, err := svc.CalculatePrice(tt.roomType, tt.nights, tt.season)

            if tt.shouldError {
                if err == nil {
                    t.Fatal("expected error, got nil")
                }
            } else {
                if err != nil {
                    t.Fatalf("unexpected error: %v", err)
                }
                if price != tt.expectedPrice {
                    t.Errorf("price: got %d, want %d", price, tt.expectedPrice)
                }
            }
        })
    }
}
```

---

## Integration Testing

Integration tests verify that components work together correctly. They touch real or containerized databases.

### Integration Test with Testcontainers

```go
// internal/booking/store_test.go

package booking

import (
    "context"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestCreateReservation_IntegrationWithDB(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // Start PostgreSQL container
    req := postgres.ContainerRequest{
        Image:       "postgres:18-alpine",
        Env: map[string]string{
            "POSTGRES_DB":       "yop_test",
            "POSTGRES_PASSWORD": "password",
        },
    }

    container, err := postgres.RunContainer(ctx, &req)
    if err != nil {
        t.Fatalf("failed to start postgres container: %v", err)
    }
    defer container.Terminate(ctx)

    // Get connection string
    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatalf("failed to get connection string: %v", err)
    }

    // Connect and create schema
    pool, err := pgxpool.New(ctx, connStr)
    if err != nil {
        t.Fatalf("failed to create pool: %v", err)
    }
    defer pool.Close()

    // Run migrations
    goose.SetDialect("postgres")
    if err := goose.Up(pool, "migrations"); err != nil {
        t.Fatalf("failed to run migrations: %v", err)
    }

    // Create store and test
    store := NewStore(pool)
    res, err := store.CreateReservation(ctx, "prop-1", "guest-1",
        time.Now(), time.Now().AddDate(0, 0, 3))

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if res.ID == "" {
        t.Error("expected reservation ID to be set")
    }
}
```

---

## Testing with Mocks

Mock external dependencies for faster unit tests.

### Mock Store

```go
// internal/booking/mocks.go

package booking

import (
    "context"
    "time"
)

// MockStore is a test double for the booking store
type MockStore struct {
    CreateReservationFn func(ctx context.Context, propertyID, guestID string,
        checkIn, checkOut time.Time) (*Reservation, error)
    GetReservationFn    func(ctx context.Context, id string) (*Reservation, error)
}

func (m *MockStore) CreateReservation(ctx context.Context, propertyID, guestID string,
    checkIn, checkOut time.Time) (*Reservation, error) {
    if m.CreateReservationFn != nil {
        return m.CreateReservationFn(ctx, propertyID, guestID, checkIn, checkOut)
    }
    panic("not mocked")
}

func (m *MockStore) GetReservation(ctx context.Context, id string) (*Reservation, error) {
    if m.GetReservationFn != nil {
        return m.GetReservationFn(ctx, id)
    }
    panic("not mocked")
}
```

### Using Mocks in Tests

```go
func TestCreateReservation_ValidatesInput(t *testing.T) {
    mockStore := &MockStore{
        CreateReservationFn: func(ctx context.Context, propertyID, guestID string,
            checkIn, checkOut time.Time) (*Reservation, error) {
            // Never called — we validate before calling store
            t.Fatal("store should not be called for invalid input")
            return nil, nil
        },
    }

    handler := &Handler{store: mockStore}

    // Test with missing property ID
    req := httptest.NewRequest("POST", "/", nil)
    w := httptest.NewRecorder()

    handler.CreateReservation(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }
}
```

---

## Testing Handlers

Handler tests verify HTTP request/response handling.

### Basic Handler Test

```go
func TestGetReservation_NotFound(t *testing.T) {
    mockStore := &MockStore{
        GetReservationFn: func(ctx context.Context, id string) (*Reservation, error) {
            return nil, sql.ErrNoRows
        },
    }

    handler := &Handler{store: mockStore}

    req := httptest.NewRequest("GET", "/res-999", nil)
    // Inject logger for handler
    ctx := logging.WithContext(req.Context(), logging.NewLogger("dev"))
    req = req.WithContext(ctx)

    w := httptest.NewRecorder()
    handler.GetReservation(w, req)

    if w.Code != http.StatusNotFound {
        t.Errorf("expected 404, got %d", w.Code)
    }

    var errResp apierror.APIError
    json.Unmarshal(w.Body.Bytes(), &errResp)

    if errResp.Code != "NOT_FOUND" {
        t.Errorf("expected NOT_FOUND, got %s", errResp.Code)
    }
}
```

### Handler Test with Request Body

```go
func TestCreateReservation_InvalidJSON(t *testing.T) {
    handler := &Handler{store: &MockStore{}}

    body := bytes.NewReader([]byte(`{invalid json}`))
    req := httptest.NewRequest("POST", "/", body)
    ctx := logging.WithContext(req.Context(), logging.NewLogger("dev"))
    req = req.WithContext(ctx)

    w := httptest.NewRecorder()
    handler.CreateReservation(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }

    var errResp apierror.APIError
    json.Unmarshal(w.Body.Bytes(), &errResp)

    if errResp.Code != "BAD_REQUEST" {
        t.Errorf("expected BAD_REQUEST, got %s", errResp.Code)
    }
}
```

---

## Testing Database Code

Test database-specific logic in isolation.

### Testing SQLC Generated Code

```go
func TestSelectReservationsByProperty(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping database test")
    }

    ctx := context.Background()
    pool := setupTestDB(t, ctx)  // Helper that starts container + runs migrations

    // Insert test data
    propertyID := "prop-1"
    const query = `
        INSERT INTO reservations (id, property_id, guest_id, check_in, check_out)
        VALUES ($1, $2, $3, $4, $5)
    `
    pool.Exec(ctx, query, "res-1", propertyID, "guest-1",
        time.Now(), time.Now().AddDate(0, 0, 3))
    pool.Exec(ctx, query, "res-2", propertyID, "guest-2",
        time.Now().AddDate(0, 0, 5), time.Now().AddDate(0, 0, 10))

    // Query using SQLC-generated code
    queries := New(pool)
    results, err := queries.SelectReservationsByProperty(ctx, propertyID)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if len(results) != 2 {
        t.Errorf("expected 2 results, got %d", len(results))
    }
}
```

---

## Test Utilities

### Helper: Setup Test Database

```go
// internal/db/test_helpers.go

package db

import (
    "context"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/pressly/goose/v3"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// SetupTestDB starts a PostgreSQL container and runs migrations
func SetupTestDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
    req := postgres.ContainerRequest{
        Image: "postgres:18-alpine",
        Env: map[string]string{
            "POSTGRES_DB":       "yop_test",
            "POSTGRES_PASSWORD": "password",
        },
    }

    container, err := postgres.RunContainer(ctx, &req)
    if err != nil {
        t.Fatalf("failed to start postgres: %v", err)
    }

    t.Cleanup(func() {
        if err := container.Terminate(ctx); err != nil {
            t.Errorf("failed to stop postgres: %v", err)
        }
    })

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatalf("failed to get connection string: %v", err)
    }

    pool, err := pgxpool.New(ctx, connStr)
    if err != nil {
        t.Fatalf("failed to create pool: %v", err)
    }

    // Run migrations
    goose.SetDialect("postgres")
    if err := goose.Up(pool, "migrations"); err != nil {
        t.Fatalf("failed to run migrations: %v", err)
    }

    return pool
}
```

### Helper: Mock Redis

```go
// internal/cache/test_helpers.go

package cache

import (
    "testing"

    "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
)

// NewTestCache creates a cache client with miniredis for testing
func NewTestCache(t *testing.T) (*Client, func()) {
    mr, err := miniredis.Run()
    if err != nil {
        t.Fatalf("failed to start miniredis: %v", err)
    }

    client := redis.NewClient(&redis.Options{
        Addr: mr.Addr(),
    })

    cache := New(client, "test:", nil)  // nil logger is OK for tests

    cleanup := func() {
        client.Close()
        mr.Close()
    }

    return cache, cleanup
}
```

### Helper: Inject Logger into Context

```go
// internal/testing/context.go

package testing

import (
    "context"
    "log/slog"
    "os"

    "github.com/lexxcode1/yop-pms/internal/platform/logging"
)

// ContextWithLogger creates a context with a test logger
func ContextWithLogger(ctx context.Context) context.Context {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
    return logging.WithContext(ctx, logger)
}
```

---

## Running Tests

### Run all tests

```bash
make test-backend
```

### Run with verbose output

```bash
go test -v ./internal/...
```

### Run specific package

```bash
go test -race ./internal/booking/...
```

### Run with coverage

```bash
go test -cover ./internal/...
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```

### Skip slow tests

```bash
go test -short ./...
```

### Run single test

```bash
go test -run TestCreateReservation ./internal/booking/
```

---

## Coverage Goals

Aim for **80%+ code coverage** across the codebase.

- **Critical paths** (errors, auth, finance) → 100%
- **Business logic** (availability, pricing) → 90%+
- **Infrastructure** (logging, caching, middleware) → 85%+
- **API handlers** → 80%+

Check coverage with:

```bash
go test -cover ./...
```

---

## Debugging Tests

### Run with extra logging

```bash
go test -v ./internal/booking/
```

### Attach debugger (Delve)

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv test ./internal/booking
(dlv) break TestCreateReservation
(dlv) continue
```

### Print values

```go
t.Logf("reservation: %+v", reservation)
t.Logf("error: %v", err)
```

---

## Common Pitfalls

1. **Not injecting logger into context** — Use `testing.ContextWithLogger()`
2. **Not cleaning up containers** — Use `t.Cleanup()` or `defer`
3. **Global state in tests** — Isolate each test; don't share data
4. **Flaky timing tests** — Avoid `time.Sleep()`; use synchronization primitives
5. **Not testing error cases** — Test both happy path and error scenarios

