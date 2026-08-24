package cache

import (
	"testing"
	"time"
)

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		t.Fatalf("mustParse(%q): %v", s, err)
	}
	return d
}

// --- tapeChartKeyOverlaps (pure function) ---

func TestTapeChartKeyOverlaps_FullContainment(t *testing.T) {
	// Reservation fully contained within key range
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-01:2026-03-30",
		"prop-1",
		mustParse(t, "2026-03-10"),
		mustParse(t, "2026-03-20"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overlaps {
		t.Error("expected overlap: reservation inside key range")
	}
}

func TestTapeChartKeyOverlaps_ReservationContainsKey(t *testing.T) {
	// Reservation fully contains key range
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-10:2026-03-20",
		"prop-1",
		mustParse(t, "2026-03-01"),
		mustParse(t, "2026-03-30"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overlaps {
		t.Error("expected overlap: key range inside reservation")
	}
}

func TestTapeChartKeyOverlaps_PartialOverlapAtStart(t *testing.T) {
	// Reservation starts before key, ends inside
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-10:2026-03-20",
		"prop-1",
		mustParse(t, "2026-03-05"),
		mustParse(t, "2026-03-15"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overlaps {
		t.Error("expected overlap: reservation overlaps start of key range")
	}
}

func TestTapeChartKeyOverlaps_PartialOverlapAtEnd(t *testing.T) {
	// Reservation starts inside key, ends after
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-10:2026-03-20",
		"prop-1",
		mustParse(t, "2026-03-15"),
		mustParse(t, "2026-03-25"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overlaps {
		t.Error("expected overlap: reservation overlaps end of key range")
	}
}

func TestTapeChartKeyOverlaps_AdjacentBefore_NoOverlap(t *testing.T) {
	// Reservation checkout == key start — touching but not overlapping (exclusive)
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-15:2026-03-20",
		"prop-1",
		mustParse(t, "2026-03-10"),
		mustParse(t, "2026-03-15"), // checkout == key_start
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overlaps {
		t.Error("expected no overlap: reservation checkout equals key start (exclusive boundary)")
	}
}

func TestTapeChartKeyOverlaps_AdjacentAfter_NoOverlap(t *testing.T) {
	// Reservation checkin == key end — touching but not overlapping (exclusive)
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-10:2026-03-15",
		"prop-1",
		mustParse(t, "2026-03-15"), // checkin == key_end
		mustParse(t, "2026-03-20"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overlaps {
		t.Error("expected no overlap: reservation checkin equals key end (exclusive boundary)")
	}
}

func TestTapeChartKeyOverlaps_EntirelyBefore_NoOverlap(t *testing.T) {
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-15:2026-03-20",
		"prop-1",
		mustParse(t, "2026-03-01"),
		mustParse(t, "2026-03-10"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overlaps {
		t.Error("expected no overlap: reservation entirely before key range")
	}
}

func TestTapeChartKeyOverlaps_EntirelyAfter_NoOverlap(t *testing.T) {
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-01:2026-03-10",
		"prop-1",
		mustParse(t, "2026-03-15"),
		mustParse(t, "2026-03-20"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overlaps {
		t.Error("expected no overlap: reservation entirely after key range")
	}
}

func TestTapeChartKeyOverlaps_WrongPropertyID_NoOverlap(t *testing.T) {
	// Same dates, different property — must not match
	overlaps, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-OTHER:2026-03-10:2026-03-20",
		"prop-1",
		mustParse(t, "2026-03-10"),
		mustParse(t, "2026-03-20"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overlaps {
		t.Error("expected no overlap: different property ID")
	}
}

func TestTapeChartKeyOverlaps_InvalidStartDate_ReturnsError(t *testing.T) {
	_, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:not-a-date:2026-03-20",
		"prop-1",
		mustParse(t, "2026-03-10"),
		mustParse(t, "2026-03-20"),
	)
	if err == nil {
		t.Error("expected error for unparseable start date in key")
	}
}

func TestTapeChartKeyOverlaps_MissingEndDate_ReturnsError(t *testing.T) {
	// Key has only a start date, no end date after the colon
	_, err := tapeChartKeyOverlaps(
		"yop:tapechart:prop-1:2026-03-10",
		"prop-1",
		mustParse(t, "2026-03-10"),
		mustParse(t, "2026-03-20"),
	)
	if err == nil {
		t.Error("expected error for key missing end date")
	}
}
