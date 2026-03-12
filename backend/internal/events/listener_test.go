package events_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"ollerod-pms/internal/events"
)

// ---------------------------------------------------------------------------
// Package-level test setup
// ---------------------------------------------------------------------------

var (
	testConnStr string
	testLogger  *slog.Logger
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("events_test"),
		postgres.WithUsername("admin"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432").WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}
	defer pgContainer.Terminate(ctx)

	testConnStr, _ = pgContainer.ConnectionString(ctx, "sslmode=disable")

	// Run migrations so the schema is present (pq LISTEN doesn't need tables,
	// but we run them so the listener works against a valid DB).
	db, err := sql.Open("pgx", testConnStr)
	if err != nil {
		log.Fatal(err)
	}
	if err := goose.Up(db, "../adapters/postgresql/migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}
	db.Close()

	os.Exit(m.Run())
}

// newListener returns a fresh EventListener connected to the test Postgres.
func newListener(t *testing.T) *events.EventListener {
	t.Helper()
	l := events.NewEventListener(testConnStr, testLogger)
	require.NoError(t, l.Start())
	t.Cleanup(func() { l.Stop() })
	return l
}

// notify sends a NOTIFY on the given channel via a standard database/sql connection.
func notify(t *testing.T, channel string, payload map[string]interface{}) {
	t.Helper()
	db, err := sql.Open("pgx", testConnStr)
	require.NoError(t, err)
	defer db.Close()

	raw, _ := json.Marshal(payload)
	_, err = db.Exec(fmt.Sprintf("NOTIFY %s, '%s'", channel, string(raw)))
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestEventListener_ReceivesNotification(t *testing.T) {
	l := newListener(t)

	var received atomic.Bool
	ch := make(chan events.Event, 1)

	err := l.On("test_channel", func(ctx context.Context, ev events.Event) error {
		ch <- ev
		received.Store(true)
		return nil
	})
	require.NoError(t, err)

	// Give the listener time to subscribe before sending NOTIFY
	time.Sleep(200 * time.Millisecond)

	payload := map[string]interface{}{"key": "value", "num": 42}
	notify(t, "test_channel", payload)

	select {
	case ev := <-ch:
		assert.Equal(t, "test_channel", ev.Channel)
		assert.Equal(t, "value", ev.Data["key"])
		assert.NotZero(t, ev.Timestamp)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

func TestEventListener_MultipleHandlers_AllInvoked(t *testing.T) {
	l := newListener(t)

	const numHandlers = 3
	var wg sync.WaitGroup
	wg.Add(numHandlers)

	var count atomic.Int32
	for i := 0; i < numHandlers; i++ {
		err := l.On("multi_channel", func(ctx context.Context, ev events.Event) error {
			count.Add(1)
			wg.Done()
			return nil
		})
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)
	notify(t, "multi_channel", map[string]interface{}{"x": 1})

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		assert.Equal(t, int32(numHandlers), count.Load())
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out; only %d/%d handlers invoked", count.Load(), numHandlers)
	}
}

func TestEventListener_HandlerOnSecondChannel_NotCalledForFirstChannel(t *testing.T) {
	l := newListener(t)

	var calledA, calledB atomic.Bool

	require.NoError(t, l.On("channel_a", func(_ context.Context, _ events.Event) error {
		calledA.Store(true)
		return nil
	}))
	require.NoError(t, l.On("channel_b", func(_ context.Context, _ events.Event) error {
		calledB.Store(true)
		return nil
	}))

	time.Sleep(200 * time.Millisecond)
	notify(t, "channel_a", map[string]interface{}{"src": "a"})

	// Give handlers time to fire
	time.Sleep(500 * time.Millisecond)
	assert.True(t, calledA.Load(), "channel_a handler should have been called")
	assert.False(t, calledB.Load(), "channel_b handler should NOT have been called for channel_a event")
}

func TestEventListener_MalformedPayload_HandlerNotCalled(t *testing.T) {
	l := newListener(t)

	var called atomic.Bool
	require.NoError(t, l.On("bad_payload_channel", func(_ context.Context, _ events.Event) error {
		called.Store(true)
		return nil
	}))

	time.Sleep(200 * time.Millisecond)

	// Send syntactically invalid JSON — listener should log and NOT call handler
	db, err := sql.Open("pgx", testConnStr)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("NOTIFY bad_payload_channel, 'not { valid json'")
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	assert.False(t, called.Load(), "handler must not be called for malformed payload")
}

func TestEventListener_HandlerError_DoesNotCrashListener(t *testing.T) {
	l := newListener(t)

	// Handler that always errors — listener should log the error and keep running
	var invocations atomic.Int32
	require.NoError(t, l.On("err_channel", func(_ context.Context, _ events.Event) error {
		invocations.Add(1)
		return fmt.Errorf("intentional handler error")
	}))

	time.Sleep(200 * time.Millisecond)
	notify(t, "err_channel", map[string]interface{}{"ping": true})

	// Send a second notification to prove the listener survived the first error
	time.Sleep(300 * time.Millisecond)
	notify(t, "err_channel", map[string]interface{}{"ping": true})

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, int32(2), invocations.Load(),
		"listener must survive a handler error and continue processing")
}

func TestEventListener_ConcurrentOn_NoDataRace(t *testing.T) {
	// This test is most useful with `go test -race`
	l := newListener(t)
	time.Sleep(100 * time.Millisecond)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			// Alternate between registering handlers and sending notifications
			ch := fmt.Sprintf("concurrent_channel_%d", i%4)
			_ = l.On(ch, func(_ context.Context, _ events.Event) error { return nil })
		}(i)
	}

	// Also concurrently fire NOTIFYs
	for i := 0; i < 4; i++ {
		go func(i int) {
			time.Sleep(50 * time.Millisecond)
			notify(t, fmt.Sprintf("concurrent_channel_%d", i), map[string]interface{}{"i": i})
		}(i)
	}

	wg.Wait()
	// No assertions — if there's a data race the -race detector will catch it.
}

func TestEventListener_Stop_GracefullyDrainsHandlers(t *testing.T) {
	l := newListener(t)

	var done atomic.Bool
	require.NoError(t, l.On("drain_channel", func(_ context.Context, _ events.Event) error {
		time.Sleep(100 * time.Millisecond) // simulate slow handler
		done.Store(true)
		return nil
	}))

	time.Sleep(200 * time.Millisecond)
	notify(t, "drain_channel", map[string]interface{}{"drain": true})

	time.Sleep(50 * time.Millisecond) // let handler start
	l.Stop()                          // should wait for handler to finish

	assert.True(t, done.Load(), "Stop() must wait for in-flight handlers to complete")
}
