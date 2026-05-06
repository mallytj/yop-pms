package events

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// newTestListener creates a Listener without a DB connection, for unit testing dispatch logic.
func newTestListener(onReconnect func()) *Listener {
	ctx, cancel := context.WithCancel(context.Background())
	return &Listener{
		handlers:    make(map[string][]Handler),
		logger:      testLogger(),
		onReconnect: onReconnect,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func notification(channel, payload string) *pgconn.Notification {
	return &pgconn.Notification{Channel: channel, Payload: payload}
}

// --- On() ---

func TestOn_RegistersHandler(t *testing.T) {
	l := newTestListener(nil)
	l.On("ch", func(ctx context.Context, e Event) error { return nil })

	if len(l.handlers["ch"]) != 1 {
		t.Errorf("expected 1 handler, got %d", len(l.handlers["ch"]))
	}
}

func TestOn_MultipleHandlers_SameChannel(t *testing.T) {
	l := newTestListener(nil)
	l.On("ch", func(ctx context.Context, e Event) error { return nil })
	l.On("ch", func(ctx context.Context, e Event) error { return nil })

	if len(l.handlers["ch"]) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(l.handlers["ch"]))
	}
}

func TestOn_MultipleChannels_Independent(t *testing.T) {
	l := newTestListener(nil)
	l.On("ch_a", func(ctx context.Context, e Event) error { return nil })
	l.On("ch_b", func(ctx context.Context, e Event) error { return nil })
	l.On("ch_b", func(ctx context.Context, e Event) error { return nil })

	if len(l.handlers["ch_a"]) != 1 {
		t.Errorf("ch_a: expected 1 handler, got %d", len(l.handlers["ch_a"]))
	}
	if len(l.handlers["ch_b"]) != 2 {
		t.Errorf("ch_b: expected 2 handlers, got %d", len(l.handlers["ch_b"]))
	}
}

// --- dispatch() ---

func TestDispatch_ValidPayload_CallsHandler(t *testing.T) {
	l := newTestListener(nil)

	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	l.On("reservation_changes", func(ctx context.Context, e Event) error {
		defer wg.Done()
		received = e
		return nil
	})

	l.dispatch(notification("reservation_changes", `{"operation":"INSERT","property_id":"prop-1"}`))
	wg.Wait()

	if received.Channel != "reservation_changes" {
		t.Errorf("Channel: got %q, want %q", received.Channel, "reservation_changes")
	}
	if received.Data["operation"] != "INSERT" {
		t.Errorf("Data[operation]: got %v, want INSERT", received.Data["operation"])
	}
	if received.Data["property_id"] != "prop-1" {
		t.Errorf("Data[property_id]: got %v, want prop-1", received.Data["property_id"])
	}
}

func TestDispatch_SetsTimestamp(t *testing.T) {
	l := newTestListener(nil)

	before := time.Now()
	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	l.On("ch", func(ctx context.Context, e Event) error {
		defer wg.Done()
		received = e
		return nil
	})

	l.dispatch(notification("ch", `{}`))
	wg.Wait()

	if received.Timestamp.Before(before) {
		t.Error("expected Timestamp to be set at dispatch time")
	}
}

func TestDispatch_InvalidJSON_DoesNotCallHandler(t *testing.T) {
	l := newTestListener(nil)

	called := false
	l.On("ch", func(ctx context.Context, e Event) error {
		called = true
		return nil
	})

	l.dispatch(notification("ch", `not valid json`))

	// Allow goroutines to settle
	time.Sleep(20 * time.Millisecond)

	if called {
		t.Error("handler should not be called when payload is invalid JSON")
	}
}

func TestDispatch_UnknownChannel_IsNoop(t *testing.T) {
	l := newTestListener(nil)
	// Should not panic with no handlers registered for this channel
	l.dispatch(notification("unknown_channel", `{"key":"value"}`))
}

func TestDispatch_HandlerError_DoesNotPanic(t *testing.T) {
	l := newTestListener(nil)

	var wg sync.WaitGroup
	wg.Add(1)

	l.On("ch", func(ctx context.Context, e Event) error {
		defer wg.Done()
		return errors.New("something went wrong")
	})

	l.dispatch(notification("ch", `{}`))
	wg.Wait()
}

func TestDispatch_AllHandlersCalled(t *testing.T) {
	l := newTestListener(nil)

	var count atomic.Int32
	var wg sync.WaitGroup
	wg.Add(3)

	handler := func(ctx context.Context, e Event) error {
		defer wg.Done()
		count.Add(1)
		return nil
	}

	l.On("ch", handler)
	l.On("ch", handler)
	l.On("ch", handler)

	l.dispatch(notification("ch", `{}`))
	wg.Wait()

	if count.Load() != 3 {
		t.Errorf("expected 3 handler calls, got %d", count.Load())
	}
}

func TestDispatch_OnlyCallsHandlersForChannel(t *testing.T) {
	l := newTestListener(nil)

	var calledA, calledB atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)

	l.On("ch_a", func(ctx context.Context, e Event) error {
		defer wg.Done()
		calledA.Store(true)
		return nil
	})
	l.On("ch_b", func(ctx context.Context, e Event) error {
		calledB.Store(true)
		return nil
	})

	l.dispatch(notification("ch_a", `{}`))
	wg.Wait()

	if !calledA.Load() {
		t.Error("expected ch_a handler to be called")
	}
	if calledB.Load() {
		t.Error("expected ch_b handler NOT to be called")
	}
}

// --- Stop() ---

func TestStop_WaitsForInFlightHandlers(t *testing.T) {
	l := newTestListener(nil)

	started := make(chan struct{})
	finished := make(chan struct{})

	l.On("ch", func(ctx context.Context, e Event) error {
		close(started)
		time.Sleep(30 * time.Millisecond)
		close(finished)
		return nil
	})

	l.dispatch(notification("ch", `{}`))
	<-started

	l.Stop()

	select {
	case <-finished:
		// handler completed before Stop returned — correct
	default:
		t.Error("Stop() returned before in-flight handler completed")
	}
}

// --- onReconnect ---

func TestOnReconnect_CalledAfterConnect(t *testing.T) {
	called := false
	onReconnect := func() { called = true }

	l := newTestListener(onReconnect)

	// Simulate what connect() does on a reconnect
	l.reconnect()

	if !called {
		t.Error("expected onReconnect to be called")
	}
}

func TestOnReconnect_NilSafe(t *testing.T) {
	l := newTestListener(nil)
	// Should not panic when onReconnect is nil
	l.reconnect()
}
