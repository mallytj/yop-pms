package util

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestNightsBetween_OneNight(t *testing.T) {
	arrival := time.Date(2026, 7, 1, 15, 0, 0, 0, time.UTC)
	departure := time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC)
	dates := NightsBetween(arrival, departure)
	if len(dates) != 1 {
		t.Fatalf("expected 1 night, got %d", len(dates))
	}
	if !dates[0].Equal(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected July 1, got %v", dates[0])
	}
}

func TestNightsBetween_MultipleNights(t *testing.T) {
	arrival := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	departure := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	dates := NightsBetween(arrival, departure)
	if len(dates) != 3 {
		t.Fatalf("expected 3 nights, got %d", len(dates))
	}
}

func TestNightsBetween_ZeroNights(t *testing.T) {
	arrival := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	departure := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	dates := NightsBetween(arrival, departure)
	if len(dates) != 0 {
		t.Fatalf("expected 0 nights, got %d", len(dates))
	}
}

func TestNightsBetween_Negative(t *testing.T) {
	arrival := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	departure := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	dates := NightsBetween(arrival, departure)
	if len(dates) != 0 {
		t.Fatalf("expected 0 nights for inverted range, got %d", len(dates))
	}
}

func TestDatesToPGDates(t *testing.T) {
	dates := []time.Time{
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	}
	pg := DatesToPGDates(dates)
	if len(pg) != 2 {
		t.Fatalf("expected 2 pgtype.Dates, got %d", len(pg))
	}
	if !pg[0].Valid || !pg[0].Time.Equal(dates[0]) {
		t.Errorf("pg[0] mismatch: valid=%v time=%v", pg[0].Valid, pg[0].Time)
	}
}

func TestDatesToPGDates_Empty(t *testing.T) {
	pg := DatesToPGDates(nil)
	if len(pg) != 0 {
		t.Errorf("expected empty slice, got %d", len(pg))
	}
}

func TestRemovedDates(t *testing.T) {
	old := []time.Time{
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
	}
	new_ := []time.Time{
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	removed := RemovedDates(old, new_)
	if len(removed) != 2 {
		t.Fatalf("expected 2 removed dates, got %d", len(removed))
	}
}

func TestRemovedDates_NoneRemoved(t *testing.T) {
	old := []time.Time{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
	new_ := []time.Time{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
	removed := RemovedDates(old, new_)
	if len(removed) != 0 {
		t.Fatalf("expected 0 removed, got %d", len(removed))
	}
}

func TestAddedDates(t *testing.T) {
	old := []time.Time{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
	new_ := []time.Time{
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	}
	added := AddedDates(old, new_)
	if len(added) != 1 {
		t.Fatalf("expected 1 added date, got %d", len(added))
	}
}

func TestAddedDates_NoneAdded(t *testing.T) {
	old := []time.Time{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
	new_ := []time.Time{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
	added := AddedDates(old, new_)
	if len(added) != 0 {
		t.Fatalf("expected 0 added, got %d", len(added))
	}
}

func TestTSToTime_Valid(t *testing.T) {
	ts := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	pg := pgtype.Timestamptz{Time: ts, Valid: true}
	got := TSToTime(pg)
	if !got.Equal(ts) {
		t.Errorf("TSToTime = %v, want %v", got, ts)
	}
}

func TestTSToTime_Invalid(t *testing.T) {
	pg := pgtype.Timestamptz{}
	got := TSToTime(pg)
	if !got.IsZero() {
		t.Errorf("expected zero time for invalid pg, got %v", got)
	}
}

func TestTSToTimePtr_Valid(t *testing.T) {
	ts := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	pg := pgtype.Timestamptz{Time: ts, Valid: true}
	got := TSToTimePtr(pg)
	if got == nil || !got.Equal(ts) {
		t.Errorf("TSToTimePtr = %v, want *%v", got, ts)
	}
}

func TestTSToTimePtr_Invalid(t *testing.T) {
	pg := pgtype.Timestamptz{}
	got := TSToTimePtr(pg)
	if got != nil {
		t.Errorf("expected nil for invalid pg, got %v", got)
	}
}

func TestFormatRange_Valid(t *testing.T) {
	lower := time.Date(2026, 7, 1, 15, 0, 0, 0, time.UTC)
	upper := time.Date(2026, 7, 4, 11, 0, 0, 0, time.UTC)
	r := ToRange(lower, upper)
	got := FormatRange(r)
	expected := fmt.Sprintf("[%s,%s)", lower.Format(time.RFC3339), upper.Format(time.RFC3339))
	if got != expected {
		t.Errorf("FormatRange = %q, want %q", got, expected)
	}
}

func TestFormatRange_Empty(t *testing.T) {
	r := pgtype.Range[pgtype.Timestamptz]{}
	got := FormatRange(r)
	// Empty range formats zero time values with default bracket style.
	if len(got) == 0 {
		t.Error("FormatRange(empty) returned empty string")
	}
}

func TestToRange_ProducesValidRange(t *testing.T) {
	lower := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	upper := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	r := ToRange(lower, upper)
	if !r.Valid {
		t.Error("expected valid range")
	}
	if !r.Lower.Valid || !r.Lower.Time.Equal(lower) {
		t.Error("lower bound mismatch")
	}
	if !r.Upper.Valid || !r.Upper.Time.Equal(upper) {
		t.Error("upper bound mismatch")
	}
}
