package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"
)

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- backoffSeconds ---

func TestBackoffSeconds(t *testing.T) {
	tests := []struct {
		retryCount int
		want       int
	}{
		{0, 1},
		{1, 2},
		{2, 4},
		{3, 8},
		{4, 16},
		{5, 32},
		{6, 64},
		{7, 128},
		{8, 256},
		{9, 512},
		{10, 1024},
		{11, 1800}, // capped: 2^11 = 2048 > 1800
		{100, 1800},
	}
	for _, tt := range tests {
		got := backoffSeconds(tt.retryCount)
		if got != tt.want {
			t.Errorf("backoffSeconds(%d) = %d; want %d", tt.retryCount, got, tt.want)
		}
	}
}

// --- New / defaults ---

func TestNew_DefaultConfig(t *testing.T) {
	e := New(nil, testLogger(t), Config{})
	defer e.Stop()

	if e.pollInterval != 5*time.Second {
		t.Errorf("pollInterval = %v; want 5s", e.pollInterval)
	}
	if e.batchSize != 10 {
		t.Errorf("batchSize = %d; want 10", e.batchSize)
	}
	if e.maxRetries != 3 {
		t.Errorf("maxRetries = %d; want 3", e.maxRetries)
	}
}

func TestNew_CustomConfig(t *testing.T) {
	e := New(nil, testLogger(t), Config{
		PollInterval: 30 * time.Second,
		BatchSize:    5,
		MaxRetries:   7,
	})
	defer e.Stop()

	if e.pollInterval != 30*time.Second {
		t.Errorf("pollInterval = %v; want 30s", e.pollInterval)
	}
	if e.batchSize != 5 {
		t.Errorf("batchSize = %d; want 5", e.batchSize)
	}
	if e.maxRetries != 7 {
		t.Errorf("maxRetries = %d; want 7", e.maxRetries)
	}
}

// --- Register ---

func TestRegister_HandlerCallable(t *testing.T) {
	e := New(nil, testLogger(t), Config{})
	defer e.Stop()

	var called bool
	e.Register("smtp.test", func(_ context.Context, _ json.RawMessage) error {
		called = true
		return nil
	})

	h, ok := e.handlers["smtp.test"]
	if !ok {
		t.Fatal("handler not registered for smtp.test")
	}
	if err := h(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestRegister_OverwritesPrevious(t *testing.T) {
	e := New(nil, testLogger(t), Config{})
	defer e.Stop()

	var calls int
	e.Register("smtp.test", func(_ context.Context, _ json.RawMessage) error { calls++; return nil })
	e.Register("smtp.test", func(_ context.Context, _ json.RawMessage) error { calls += 10; return nil })

	_ = e.handlers["smtp.test"](context.Background(), nil)
	if calls != 10 {
		t.Errorf("calls = %d; want 10 (second registration should win)", calls)
	}
}

func TestRegister_MultipleEventTypes(t *testing.T) {
	e := New(nil, testLogger(t), Config{})
	defer e.Stop()

	e.Register("smtp.confirmation", func(_ context.Context, _ json.RawMessage) error { return nil })
	e.Register("smtp.pre_arrival", func(_ context.Context, _ json.RawMessage) error { return nil })

	if _, ok := e.handlers["smtp.confirmation"]; !ok {
		t.Error("smtp.confirmation not registered")
	}
	if _, ok := e.handlers["smtp.pre_arrival"]; !ok {
		t.Error("smtp.pre_arrival not registered")
	}
}

// --- Stop without Start ---

func TestStop_WithoutStart_DoesNotHang(t *testing.T) {
	e := New(nil, testLogger(t), Config{})
	done := make(chan struct{})
	go func() { e.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() hung without a prior Start()")
	}
}
