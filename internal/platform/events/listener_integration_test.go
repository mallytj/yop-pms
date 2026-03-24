package events

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// startTestPostgres starts an isolated PostgreSQL container for the duration of
// the test and returns a connection string. Container is terminated via t.Cleanup.
func startTestPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("yop_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		// Wait until PostgreSQL is actually accepting connections, not just
		// when the port becomes open. The log message appears twice on a clean
		// start: once during initdb and once when postmaster is ready.
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

func integrationLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// notify sends a pg_notify from a dedicated connection. Callers are responsible
// for closing the returned connection.
func notify(t *testing.T, connStr, channel, payload string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("notify connect: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "SELECT pg_notify($1, $2)", channel, payload); err != nil {
		t.Fatalf("pg_notify(%q): %v", channel, err)
	}
}

// TestListener_Integration_EndToEnd verifies the full path from database NOTIFY
// through connect → processNotifications → dispatch → handler.
func TestListener_Integration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)

	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	l := New(connStr, integrationLogger(), nil)
	l.On("test_channel", func(_ context.Context, e Event) error {
		defer wg.Done()
		received = e
		return nil
	})
	l.Start()
	defer l.Stop()

	// Allow time for connect() to run and LISTEN to execute.
	time.Sleep(500 * time.Millisecond)

	notify(t, connStr, "test_channel", `{"operation":"INSERT","property_id":"prop-1"}`)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler not called within 5s of NOTIFY")
	}

	if received.Channel != "test_channel" {
		t.Errorf("Channel: got %q, want %q", received.Channel, "test_channel")
	}
	if received.Data["operation"] != "INSERT" {
		t.Errorf("Data[operation]: got %v, want INSERT", received.Data["operation"])
	}
	if received.Data["property_id"] != "prop-1" {
		t.Errorf("Data[property_id]: got %v, want prop-1", received.Data["property_id"])
	}
	if received.Timestamp.IsZero() {
		t.Error("Timestamp should be set at dispatch time")
	}
}

// TestListener_Integration_MultipleHandlers verifies that all registered
// handlers for a channel are called concurrently on a single notification.
func TestListener_Integration_MultipleHandlers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)

	var wg sync.WaitGroup
	wg.Add(3)

	l := New(connStr, integrationLogger(), nil)
	for range 3 {
		l.On("ch", func(_ context.Context, _ Event) error {
			defer wg.Done()
			return nil
		})
	}
	l.Start()
	defer l.Stop()

	time.Sleep(500 * time.Millisecond)

	notify(t, connStr, "ch", `{}`)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("not all 3 handlers called within 5s")
	}
}

// TestListener_Integration_InvalidJSONDropped verifies that a malformed
// notification payload is silently dropped and the listener continues
// processing subsequent valid notifications without crashing.
func TestListener_Integration_InvalidJSONDropped(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)

	var calls atomic.Int32
	called := make(chan struct{}, 1)

	l := New(connStr, integrationLogger(), nil)
	l.On("ch", func(_ context.Context, _ Event) error {
		calls.Add(1)
		select {
		case called <- struct{}{}:
		default:
		}
		return nil
	})
	l.Start()
	defer l.Stop()

	time.Sleep(500 * time.Millisecond)

	notify(t, connStr, "ch", `not valid json`) // should be silently dropped
	notify(t, connStr, "ch", `{"ok": true}`)   // should reach the handler

	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("handler not called for valid notification after invalid JSON")
	}

	// Brief settle to catch any spurious second call from the bad payload.
	time.Sleep(100 * time.Millisecond)
	if n := calls.Load(); n != 1 {
		t.Errorf("handler called %d times, want 1 (invalid JSON should be dropped)", n)
	}
}

// TestListener_Integration_StopWhileConnected verifies Stop() returns cleanly
// when called on a listener with an active PostgreSQL connection.
func TestListener_Integration_StopWhileConnected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	connStr := startTestPostgres(t)

	l := New(connStr, integrationLogger(), nil)
	l.On("ch", func(_ context.Context, _ Event) error { return nil })
	l.Start()

	time.Sleep(500 * time.Millisecond)

	stopped := make(chan struct{})
	go func() { l.Stop(); close(stopped) }()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5s on a live connection")
	}
}

// TestListener_Integration_Reconnect verifies that after a forced disconnect
// the listener reconnects and calls the onReconnect callback.
func TestListener_Integration_Reconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	ctx := context.Background()
	connStr := startTestPostgres(t)

	reconnected := make(chan struct{}, 1)
	l := New(connStr, integrationLogger(), func() {
		select {
		case reconnected <- struct{}{}:
		default:
		}
	})
	l.On("ch", func(_ context.Context, _ Event) error { return nil })
	l.Start()
	defer l.Stop()

	time.Sleep(1 * time.Second)

	// Terminate the listener's connection from a separate admin connection.
	admin, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	defer admin.Close(ctx)

	if _, err := admin.Exec(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE pid <> pg_backend_pid()
	`); err != nil {
		t.Fatalf("pg_terminate_backend: %v", err)
	}

	// Listener backs off 1s before reconnecting. Allow up to 30s for retries
	// with exponential backoff (1s + 2s + 4s + 8s = 15s worst case).
	select {
	case <-reconnected:
	case <-time.After(30 * time.Second):
		t.Fatal("onReconnect not called within 30s after forced disconnect")
	}
}

// TestListener_Integration_ReconnectContinuesDelivery verifies that after a
// forced reconnect, the listener still delivers subsequent notifications.
func TestListener_Integration_ReconnectContinuesDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test — requires PostgreSQL")
	}

	ctx := context.Background()
	connStr := startTestPostgres(t)

	reconnected := make(chan struct{}, 1)
	l := New(connStr, integrationLogger(), func() {
		select {
		case reconnected <- struct{}{}:
		default:
		}
	})

	var wg sync.WaitGroup
	wg.Add(1)
	l.On("ch", func(_ context.Context, _ Event) error {
		defer wg.Done()
		return nil
	})
	l.Start()
	defer l.Stop()

	time.Sleep(1 * time.Second)

	// Force disconnect via a separate admin connection.
	admin, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	if _, err := admin.Exec(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE pid <> pg_backend_pid()
	`); err != nil {
		admin.Close(ctx)
		t.Fatalf("pg_terminate_backend: %v", err)
	}
	admin.Close(ctx)

	// Wait for reconnect before sending a notification. Allow up to 30s for
	// retries with exponential backoff (1s + 2s + 4s + 8s = 15s worst case).
	select {
	case <-reconnected:
	case <-time.After(30 * time.Second):
		t.Fatal("did not reconnect within 30s")
	}

	// Allow the re-subscribed LISTEN to settle.
	time.Sleep(200 * time.Millisecond)

	notify(t, connStr, "ch", `{"after":"reconnect"}`)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler not called after reconnect")
	}
}
