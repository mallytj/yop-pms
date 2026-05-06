package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// startTestPostgres starts an isolated PostgreSQL 18 container and returns a
// connection string. Container is terminated via t.Cleanup.
func startTestPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("yop_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := ctr.Terminate(ctx); err != nil {
			t.Errorf("terminate postgres container: %v", err)
		}
	})

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}
	return connStr
}

// applyMigration reads the Up section of migrations/00006_outbox_worker.sql
// and executes it against the given database pool.
func applyMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, file, _, _ := runtime.Caller(0)
	migPath := filepath.Join(filepath.Dir(file), "../../..", "migrations", "00006_outbox_worker.sql")

	raw, err := os.ReadFile(migPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	up := extractGooseUp(raw)
	if _, err := pool.Exec(context.Background(), up); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
}

// extractGooseUp returns the SQL between "-- +goose Up" and "-- +goose Down"
// with goose directive comments stripped.
func extractGooseUp(src []byte) string {
	var buf bytes.Buffer
	inUp := false
	scanner := bufio.NewScanner(bytes.NewReader(src))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "-- +goose Up":
			inUp = true
		case trimmed == "-- +goose Down":
			return buf.String()
		case inUp && strings.HasPrefix(trimmed, "-- +goose"):
			// skip goose directive lines
		case inUp:
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

func newTestPool(t *testing.T, connStr string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func integrationLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func insertEvent(t *testing.T, pool *pgxpool.Pool, eventType string, payload any) {
	t.Helper()
	b, _ := json.Marshal(payload)
	_, err := pool.Exec(context.Background(),
		`INSERT INTO internal.outbox_events (event_type, payload) VALUES ($1, $2)`,
		eventType, b,
	)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
}

func countByStatus(t *testing.T, pool *pgxpool.Pool, status string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM internal.outbox_events WHERE status = $1`, status,
	).Scan(&n)
	if err != nil {
		t.Fatalf("count by status: %v", err)
	}
	return n
}

// --- Integration Tests ---

func TestWorker_Integration_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)
	pool := newTestPool(t, connStr)
	applyMigration(t, pool)

	insertEvent(t, pool, EventConfirmationEmail, ConfirmationEmailPayload{
		ReservationID: "res-1",
		GuestEmail:    "guest@example.com",
		GuestName:     "Alice",
		PropertyName:  "Yop Hotel",
	})

	var received ConfirmationEmailPayload
	called := make(chan struct{})

	e := New(pool, integrationLogger(), Config{PollInterval: 100 * time.Millisecond})
	e.Register(EventConfirmationEmail, func(_ context.Context, p json.RawMessage) error {
		if err := json.Unmarshal(p, &received); err != nil {
			return err
		}
		close(called)
		return nil
	})
	e.Start()
	defer e.Stop()

	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("handler not called within 5s")
	}

	pollDeadline := time.After(2 * time.Second)
	for {
		if countByStatus(t, pool, "completed") == 1 {
			break
		}
		select {
		case <-pollDeadline:
			t.Errorf("completed count never reached 1 within 2s")
			return
		case <-time.After(20 * time.Millisecond):
		}
	}
	if received.ReservationID != "res-1" {
		t.Errorf("ReservationID = %q; want res-1", received.ReservationID)
	}
}

func TestWorker_Integration_RetryOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)
	pool := newTestPool(t, connStr)
	applyMigration(t, pool)

	insertEvent(t, pool, EventPreArrivalEmail, PreArrivalEmailPayload{GuestEmail: "g@example.com"})

	var calls atomic.Int32
	e := New(pool, integrationLogger(), Config{
		PollInterval: 50 * time.Millisecond,
		MaxRetries:   3,
	})
	e.Register(EventPreArrivalEmail, func(_ context.Context, _ json.RawMessage) error {
		calls.Add(1)
		return errors.New("smtp unavailable")
	})
	e.Start()
	defer e.Stop()

	// backoff after first fail: 1s. Force process_at back to now so next poll picks it up.
	resetProcessAt := func() {
		_, err := pool.Exec(context.Background(),
			`UPDATE internal.outbox_events SET process_at = NOW() WHERE status = 'pending'`)
		if err != nil {
			t.Errorf("reset process_at: %v", err)
		}
	}

	// Wait for first call, then let retries fire by resetting process_at.
	deadline := time.After(10 * time.Second)
	for calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("handler only called %d times (want 3) within 10s", calls.Load())
		case <-time.After(100 * time.Millisecond):
			resetProcessAt()
		}
	}

	failDeadline := time.After(2 * time.Second)
	for {
		if countByStatus(t, pool, "failed") == 1 {
			break
		}
		select {
		case <-failDeadline:
			t.Errorf("failed count never reached 1 within 2s after maxRetries")
			return
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func TestWorker_Integration_DeadLetterNotify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)
	pool := newTestPool(t, connStr)
	applyMigration(t, pool)

	// maxRetries=1 so first failure immediately dead-letters.
	insertEvent(t, pool, EventCancellationEmail, CancellationEmailPayload{GuestEmail: "g@example.com"})

	// Subscribe to dead_lettered channel before starting the worker.
	listenConn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire listen conn: %v", err)
	}
	defer listenConn.Release()
	if _, err := listenConn.Exec(context.Background(), "LISTEN outbox_dead_lettered"); err != nil {
		t.Fatalf("listen: %v", err)
	}

	e := New(pool, integrationLogger(), Config{
		PollInterval: 50 * time.Millisecond,
		MaxRetries:   1,
	})
	e.Register(EventCancellationEmail, func(_ context.Context, _ json.RawMessage) error {
		return fmt.Errorf("always fails")
	})
	e.Start()
	defer e.Stop()

	notifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := listenConn.Conn().WaitForNotification(notifyCtx)
	if err != nil {
		t.Fatalf("no notification on outbox_dead_lettered within 5s: %v", err)
	}
	if n.Channel != "outbox_dead_lettered" {
		t.Errorf("channel = %q; want outbox_dead_lettered", n.Channel)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(n.Payload), &payload); err != nil {
		t.Fatalf("parse notification payload: %v", err)
	}
	if payload["event_type"] != EventCancellationEmail {
		t.Errorf("event_type = %q; want %q", payload["event_type"], EventCancellationEmail)
	}
}

func TestWorker_Integration_StuckProcessingReclaimed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)
	pool := newTestPool(t, connStr)
	applyMigration(t, pool)

	// Simulate a row stuck in 'processing' with an expired process_at (crash scenario).
	_, err := pool.Exec(context.Background(), `
		INSERT INTO internal.outbox_events (event_type, payload, status, process_at)
		VALUES ($1, $2, 'processing', NOW() - INTERVAL '1 second')
	`, EventConfirmationEmail, []byte(`{"reservation_id":"stuck-1"}`))
	if err != nil {
		t.Fatalf("insert stuck row: %v", err)
	}

	called := make(chan struct{})
	e := New(pool, integrationLogger(), Config{PollInterval: 50 * time.Millisecond})
	e.Register(EventConfirmationEmail, func(_ context.Context, _ json.RawMessage) error {
		close(called)
		return nil
	})
	e.Start()
	defer e.Stop()

	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("stuck processing row not reclaimed within 5s")
	}
}

func TestWorker_Integration_UnknownEventTypeDeadLettered(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)
	pool := newTestPool(t, connStr)
	applyMigration(t, pool)

	insertEvent(t, pool, "unknown.type", map[string]string{"x": "y"})

	e := New(pool, integrationLogger(), Config{
		PollInterval: 50 * time.Millisecond,
		MaxRetries:   3,
	})
	// No handler registered for "unknown.type"
	e.Start()
	defer e.Stop()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("event not dead-lettered within 5s")
		case <-time.After(100 * time.Millisecond):
			if countByStatus(t, pool, "failed") == 1 {
				return
			}
		}
	}
}
