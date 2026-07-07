package cache

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/lexxcode1/yop-pms/internal/platform/events"
)

func invalidationLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func makeEvent(data map[string]any) events.Event {
	return events.Event{Channel: "reservation_changes", Data: data}
}

func assertContains(t *testing.T, patterns []string, want string) {
	t.Helper()
	for _, p := range patterns {
		if p == want {
			return
		}
	}
	t.Errorf("expected patterns to contain %q\ngot: %v", want, patterns)
}

// mockCache implements reservationCacheInvalidator for testing.
// It records all patterns passed to Invalidate, and for InvalidateIf it runs
// the predicate against keysToTest so tests can assert which keys are evicted.
type mockCache struct {
	mu         sync.Mutex
	patterns   []string // patterns passed to Invalidate
	ifPattern  string   // pattern passed to InvalidateIf
	keysToTest []string // keys presented to the shouldDelete predicate
	deleted    []string // keys where shouldDelete returned true
}

func (m *mockCache) Invalidate(_ context.Context, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.patterns = append(m.patterns, pattern)
	return nil
}

func (m *mockCache) InvalidateIf(_ context.Context, pattern string, shouldDelete func(key string) bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ifPattern = pattern
	for _, key := range m.keysToTest {
		if shouldDelete(key) {
			m.deleted = append(m.deleted, key)
		}
	}
	return nil
}

func (m *mockCache) snapshot() (patterns []string, ifPattern string, deleted []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.patterns...),
		m.ifPattern,
		append([]string(nil), m.deleted...)
}

// --- parseReservationChange ---

func TestParseReservationChange_ValidPayload(t *testing.T) {
	change, err := parseReservationChange(makeEvent(map[string]any{
		"operation":      "INSERT",
		"property_id":    "prop-1",
		"record_id":      "res-1",
		"check_in_date":  "2026-03-15",
		"check_out_date": "2026-03-17",
		"table":          "reservations",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if change.PropertyID != "prop-1" {
		t.Errorf("PropertyID: got %q, want %q", change.PropertyID, "prop-1")
	}
	if change.RecordID != "res-1" {
		t.Errorf("RecordID: got %q, want %q", change.RecordID, "res-1")
	}
	if change.CheckIn.Format(dateLayout) != "2026-03-15" {
		t.Errorf("CheckIn: got %q, want 2026-03-15", change.CheckIn.Format(dateLayout))
	}
	if change.CheckOut.Format(dateLayout) != "2026-03-17" {
		t.Errorf("CheckOut: got %q, want 2026-03-17", change.CheckOut.Format(dateLayout))
	}
}

func TestParseReservationChange_MissingCheckIn_ReturnsError(t *testing.T) {
	_, err := parseReservationChange(makeEvent(map[string]any{
		"property_id":    "prop-1",
		"check_out_date": "2026-03-17",
	}))
	if err == nil {
		t.Error("expected error for missing check_in_date")
	}
}

func TestParseReservationChange_MissingCheckOut_ReturnsError(t *testing.T) {
	_, err := parseReservationChange(makeEvent(map[string]any{
		"property_id":   "prop-1",
		"check_in_date": "2026-03-15",
	}))
	if err == nil {
		t.Error("expected error for missing check_out_date")
	}
}

func TestParseReservationChange_InvalidCheckInFormat_ReturnsError(t *testing.T) {
	_, err := parseReservationChange(makeEvent(map[string]any{
		"check_in_date":  "not-a-date",
		"check_out_date": "2026-03-17",
	}))
	if err == nil {
		t.Error("expected error for invalid check_in_date format")
	}
}

func TestParseReservationChange_InvalidCheckOutFormat_ReturnsError(t *testing.T) {
	_, err := parseReservationChange(makeEvent(map[string]any{
		"check_in_date":  "2026-03-15",
		"check_out_date": "not-a-date",
	}))
	if err == nil {
		t.Error("expected error for invalid check_out_date format")
	}
}

// --- NewReservationChangeHandler ---

func TestReservationChangeHandler_TwoNightStay(t *testing.T) {
	mock := &mockCache{}
	handler := NewReservationChangeHandler(mock, invalidationLogger())

	err := handler(context.Background(), makeEvent(map[string]any{
		"operation":      "INSERT",
		"property_id":    "prop-1",
		"record_id":      "res-1",
		"check_in_date":  "2026-03-15",
		"check_out_date": "2026-03-17", // 2 nights: 15th and 16th
		"table":          "reservations",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	patterns, ifPattern, _ := mock.snapshot()

	// 2 availability keys + 1 reservation key
	if len(patterns) != 3 {
		t.Errorf("expected 3 Invalidate calls (2 dates + 1 reservation), got %d: %v", len(patterns), patterns)
	}
	assertContains(t, patterns, "yop:availability:prop-1:*:2026-03-15")
	assertContains(t, patterns, "yop:availability:prop-1:*:2026-03-16")
	assertContains(t, patterns, "yop:reservation:res-1")

	// Planner scan was issued for the right property
	if ifPattern != "yop:planner:prop-1:*" {
		t.Errorf("planner pattern: got %q, want %q", ifPattern, "yop:planner:prop-1:*")
	}
}

func TestReservationChangeHandler_SingleNightStay(t *testing.T) {
	mock := &mockCache{}
	handler := NewReservationChangeHandler(mock, invalidationLogger())

	err := handler(context.Background(), makeEvent(map[string]any{
		"operation":      "INSERT",
		"property_id":    "prop-1",
		"record_id":      "res-1",
		"check_in_date":  "2026-03-15",
		"check_out_date": "2026-03-16", // 1 night
		"table":          "reservations",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	patterns, _, _ := mock.snapshot()
	if len(patterns) != 2 {
		t.Errorf("expected 2 Invalidate calls (1 date + 1 reservation), got %d: %v", len(patterns), patterns)
	}
	assertContains(t, patterns, "yop:availability:prop-1:*:2026-03-15")
	assertContains(t, patterns, "yop:reservation:res-1")
}

func TestReservationChangeHandler_PlannerOverlap_IsDeleted(t *testing.T) {
	mock := &mockCache{
		keysToTest: []string{
			"yop:planner:prop-1:2026-03-10:2026-03-20", // overlaps [15, 17)
			"yop:planner:prop-1:2026-03-01:2026-03-10", // ends at check-in — no overlap
		},
	}
	handler := NewReservationChangeHandler(mock, invalidationLogger())

	err := handler(context.Background(), makeEvent(map[string]any{
		"operation":      "UPDATE",
		"property_id":    "prop-1",
		"record_id":      "res-1",
		"check_in_date":  "2026-03-15",
		"check_out_date": "2026-03-17",
		"table":          "reservations",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, deleted := mock.snapshot()
	if len(deleted) != 1 || deleted[0] != "yop:planner:prop-1:2026-03-10:2026-03-20" {
		t.Errorf("deleted: got %v, want [yop:planner:prop-1:2026-03-10:2026-03-20]", deleted)
	}
}

func TestReservationChangeHandler_PlannerNoOverlap_NothingDeleted(t *testing.T) {
	mock := &mockCache{
		keysToTest: []string{
			"yop:planner:prop-1:2026-01-01:2026-02-01", // way before
			"yop:planner:prop-1:2026-05-01:2026-06-01", // way after
		},
	}
	handler := NewReservationChangeHandler(mock, invalidationLogger())

	err := handler(context.Background(), makeEvent(map[string]any{
		"operation":      "INSERT",
		"property_id":    "prop-1",
		"record_id":      "res-1",
		"check_in_date":  "2026-03-15",
		"check_out_date": "2026-03-17",
		"table":          "reservations",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, deleted := mock.snapshot()
	if len(deleted) != 0 {
		t.Errorf("expected no planner keys deleted, got: %v", deleted)
	}
}

func TestReservationChangeHandler_MissingDates_ReturnsError(t *testing.T) {
	mock := &mockCache{}
	handler := NewReservationChangeHandler(mock, invalidationLogger())

	err := handler(context.Background(), makeEvent(map[string]any{
		"operation":   "INSERT",
		"property_id": "prop-1",
		"record_id":   "res-1",
		"table":       "reservations",
		// check_in_date and check_out_date absent
	}))
	if err == nil {
		t.Error("expected error for missing dates, got nil")
	}
}

func TestReservationChangeHandler_InvalidDateFormat_ReturnsError(t *testing.T) {
	mock := &mockCache{}
	handler := NewReservationChangeHandler(mock, invalidationLogger())

	err := handler(context.Background(), makeEvent(map[string]any{
		"operation":      "INSERT",
		"property_id":    "prop-1",
		"record_id":      "res-1",
		"check_in_date":  "not-a-date",
		"check_out_date": "2026-03-17",
		"table":          "reservations",
	}))
	if err == nil {
		t.Error("expected error for invalid date format, got nil")
	}
}

func TestReservationChangeHandler_NoInvalidationsOnParseError(t *testing.T) {
	mock := &mockCache{}
	handler := NewReservationChangeHandler(mock, invalidationLogger())

	_ = handler(context.Background(), makeEvent(map[string]any{
		"check_in_date":  "bad",
		"check_out_date": "2026-03-17",
	}))

	patterns, _, _ := mock.snapshot()
	if len(patterns) != 0 {
		t.Errorf("expected no Invalidate calls on parse error, got %d", len(patterns))
	}
}
