package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lexxcode1/yop-pms/internal/store"
)

var _ outboxInserter = (*mockQuerier)(nil)

// mockQuerier records calls to CreateOutboxEvent for assertion.
type mockQuerier struct {
	fn  func(ctx context.Context, arg *store.CreateOutboxEventParams) (uuid.UUID, error)
	got *store.CreateOutboxEventParams
}

func (m *mockQuerier) CreateOutboxEvent(ctx context.Context, arg *store.CreateOutboxEventParams) (uuid.UUID, error) {
	m.got = arg
	return m.fn(ctx, arg)
}

func okQuerier() *mockQuerier {
	return &mockQuerier{fn: func(_ context.Context, _ *store.CreateOutboxEventParams) (uuid.UUID, error) {
		return uuid.New(), nil
	}}
}

func errQuerier(err error) *mockQuerier {
	return &mockQuerier{fn: func(_ context.Context, _ *store.CreateOutboxEventParams) (uuid.UUID, error) {
		return uuid.UUID{}, err
	}}
}

// --- EnqueueAt ---

func TestEnqueueAt_HappyPath(t *testing.T) {
	q := okQuerier()
	processAt := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	payload := map[string]string{"key": "value"}

	if err := EnqueueAt(context.Background(), q, "test.event", payload, processAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.got == nil {
		t.Fatal("CreateOutboxEvent not called")
	}
	if q.got.EventType != "test.event" {
		t.Errorf("EventType = %q; want test.event", q.got.EventType)
	}
	if string(q.got.Payload) != `{"key":"value"}` {
		t.Errorf("Payload = %q; want {\"key\":\"value\"}", q.got.Payload)
	}
	want := pgtype.Timestamptz{Time: processAt, Valid: true}
	if q.got.ProcessAt != want {
		t.Errorf("ProcessAt = %v; want %v", q.got.ProcessAt, want)
	}
}

func TestEnqueueAt_MarshalError(t *testing.T) {
	q := okQuerier()
	// Channels cannot be JSON-marshalled.
	err := EnqueueAt(context.Background(), q, "test.event", make(chan int), time.Now())
	if err == nil {
		t.Fatal("expected error for unmarshalable payload, got nil")
	}
	if q.got != nil {
		t.Error("CreateOutboxEvent should not be called when marshal fails")
	}
}

func TestEnqueueAt_DBError(t *testing.T) {
	dbErr := errors.New("connection refused")
	q := errQuerier(dbErr)

	err := EnqueueAt(context.Background(), q, "test.event", "payload", time.Now())
	if err == nil {
		t.Fatal("expected error from DB, got nil")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("error chain should contain dbErr; got %v", err)
	}
}

func TestEnqueueAt_ErrorWrapping(t *testing.T) {
	tests := []struct {
		name        string
		payload     any
		dbErr       error
		wantContain string
	}{
		{
			name:        "marshal failure prefix",
			payload:     make(chan int),
			wantContain: "worker: marshal payload",
		},
		{
			name:        "db failure prefix",
			payload:     "ok",
			dbErr:       errors.New("timeout"),
			wantContain: "worker: create outbox event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q outboxInserter
			if tt.dbErr != nil {
				q = errQuerier(tt.dbErr)
			} else {
				q = okQuerier()
			}
			err := EnqueueAt(context.Background(), q, "test.event", tt.payload, time.Now())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); len(got) < len(tt.wantContain) || got[:len(tt.wantContain)] != tt.wantContain {
				t.Errorf("error = %q; want prefix %q", got, tt.wantContain)
			}
		})
	}
}

// --- Enqueue ---

func TestEnqueue_DelegatesToEnqueueAt(t *testing.T) {
	q := okQuerier()
	before := time.Now()

	if err := Enqueue(context.Background(), q, "test.event", "payload"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := time.Now()
	if q.got == nil {
		t.Fatal("CreateOutboxEvent not called")
	}
	processAt := q.got.ProcessAt.Time
	if processAt.Before(before) || processAt.After(after) {
		t.Errorf("ProcessAt %v not within [%v, %v]", processAt, before, after)
	}
}

func TestEnqueue_PropagatesError(t *testing.T) {
	dbErr := errors.New("db down")
	err := Enqueue(context.Background(), errQuerier(dbErr), "test.event", "payload")
	if !errors.Is(err, dbErr) {
		t.Errorf("expected dbErr in chain; got %v", err)
	}
}
