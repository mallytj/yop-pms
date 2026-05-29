package realtime

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexxcode1/yop-pms/internal/platform/events"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func TestNewHub_CreatesEmptyClientMap(t *testing.T) {
	h := NewHub(testLogger())
	if h == nil {
		t.Fatal("NewHub returned nil")
	}
	if len(h.clients) != 0 {
		t.Errorf("expected 0 clients, got %d", len(h.clients))
	}
}

func TestOnEvent_MissingPropertyID_DropsSilently(t *testing.T) {
	h := NewHub(testLogger())
	event := events.Event{
		Channel:   "reservation_changes",
		Data:      map[string]any{"table": "reservations", "op": "INSERT"},
		Timestamp: time.Now(),
	}
	// Should not panic or error.
	err := h.OnEvent(context.Background(), event)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestOnEvent_InvalidPropertyID_DropsSilently(t *testing.T) {
	h := NewHub(testLogger())
	event := events.Event{
		Channel:   "reservation_changes",
		Data:      map[string]any{"property_id": "not-a-uuid"},
		Timestamp: time.Now(),
	}
	err := h.OnEvent(context.Background(), event)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestOnEvent_WithPropertyID_BroadcastsToMatchingClients(t *testing.T) {
	h := NewHub(testLogger())

	propertyID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	ch := make(chan SSEMessage, 64)
	h.mu.Lock()
	h.clients[ch] = propertyID
	h.mu.Unlock()

	event := events.Event{
		Channel: "reservation_changes",
		Data: map[string]any{
			"table":       "reservations",
			"op":          "INSERT",
			"id":          uuid.New().String(),
			"property_id": propertyID.String(),
		},
		Timestamp: time.Now(),
	}

	err := h.OnEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case msg := <-ch:
		if msg.Event != "reservation.changed" {
			t.Errorf("expected event 'reservation.changed', got '%s'", msg.Event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected message on client channel, got none")
	}
}

func TestOnEvent_SlowConsumer_DropsSilently(t *testing.T) {
	h := NewHub(testLogger())

	propertyID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	// Zero-buffer channel with no reader — completely backed up.
	ch := make(chan SSEMessage, 0)
	h.mu.Lock()
	h.clients[ch] = propertyID
	h.mu.Unlock()

	event := events.Event{
		Channel: "reservation_changes",
		Data: map[string]any{
			"table":       "reservations",
			"op":          "INSERT",
			"id":          uuid.New().String(),
			"property_id": propertyID.String(),
		},
		Timestamp: time.Now(),
	}

	err := h.OnEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both original message and resync are dropped — channel is full
	// and no reader is waiting. This is acceptable: the client will
	// reconnect via EventSource and receive a fresh state.
	select {
	case <-ch:
		t.Fatal("expected no message on backed-up channel")
	case <-time.After(50 * time.Millisecond):
		// correct — silently dropped
	}
}

func TestResync_BroadcastsToAllClients(t *testing.T) {
	h := NewHub(testLogger())

	ch1 := make(chan SSEMessage, 64)
	ch2 := make(chan SSEMessage, 64)
	h.mu.Lock()
	h.clients[ch1] = uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	h.clients[ch2] = uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	h.mu.Unlock()

	h.Resync(context.Background())

	for i, ch := range []chan SSEMessage{ch1, ch2} {
		select {
		case msg := <-ch:
			if msg.Event != "resync" {
				t.Errorf("client %d: expected 'resync', got '%s'", i, msg.Event)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("client %d: expected resync message, got none", i)
		}
	}
}

func TestBuildMessage_MapsTableToEventName(t *testing.T) {
	h := NewHub(testLogger())
	tests := []struct {
		table string
		want  string
	}{
		{"reservations", "reservation.changed"},
		{"reservation_items", "reservation.changed"},
		{"room_inventory_ledger", "availability.changed"},
		{"booked_daily_rates", "rate.changed"},
		{"unknown_table", "change"},
	}

	for _, tt := range tests {
		event := events.Event{
			Channel: "reservation_changes",
			Data: map[string]any{
				"table":       tt.table,
				"op":          "INSERT",
				"id":          uuid.New().String(),
				"property_id": uuid.New().String(),
			},
			Timestamp: time.Now(),
		}
		msg := h.ConvertEventToSSEMessage(event)
		if msg.Event != tt.want {
			t.Errorf("table=%q: expected event %q, got %q", tt.table, tt.want, msg.Event)
		}
	}
}
