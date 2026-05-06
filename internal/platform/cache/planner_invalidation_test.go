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

// --- plannerKeyOverlaps (pure function) ---

func TestPlannerKeyOverlaps_FullContainment(t *testing.T) {
	// Reservation fully contained within key range
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-01:2026-03-30",
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

func TestPlannerKeyOverlaps_ReservationContainsKey(t *testing.T) {
	// Reservation fully contains key range
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-10:2026-03-20",
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

func TestPlannerKeyOverlaps_PartialOverlapAtStart(t *testing.T) {
	// Reservation starts before key, ends inside
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-10:2026-03-20",
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

func TestPlannerKeyOverlaps_PartialOverlapAtEnd(t *testing.T) {
	// Reservation starts inside key, ends after
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-10:2026-03-20",
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

func TestPlannerKeyOverlaps_AdjacentBefore_NoOverlap(t *testing.T) {
	// Reservation checkout == key start — touching but not overlapping (exclusive)
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-15:2026-03-20",
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

func TestPlannerKeyOverlaps_AdjacentAfter_NoOverlap(t *testing.T) {
	// Reservation checkin == key end — touching but not overlapping (exclusive)
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-10:2026-03-15",
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

func TestPlannerKeyOverlaps_EntirelyBefore_NoOverlap(t *testing.T) {
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-15:2026-03-20",
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

func TestPlannerKeyOverlaps_EntirelyAfter_NoOverlap(t *testing.T) {
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-01:2026-03-10",
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

func TestPlannerKeyOverlaps_WrongPropertyID_NoOverlap(t *testing.T) {
	// Same dates, different property — must not match
	overlaps, err := plannerKeyOverlaps(
		"yop:planner:prop-OTHER:2026-03-10:2026-03-20",
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

func TestPlannerKeyOverlaps_InvalidStartDate_ReturnsError(t *testing.T) {
	_, err := plannerKeyOverlaps(
		"yop:planner:prop-1:not-a-date:2026-03-20",
		"prop-1",
		mustParse(t, "2026-03-10"),
		mustParse(t, "2026-03-20"),
	)
	if err == nil {
		t.Error("expected error for unparseable start date in key")
	}
}

func TestPlannerKeyOverlaps_MissingEndDate_ReturnsError(t *testing.T) {
	// Key has only a start date, no end date after the colon
	_, err := plannerKeyOverlaps(
		"yop:planner:prop-1:2026-03-10",
		"prop-1",
		mustParse(t, "2026-03-10"),
		mustParse(t, "2026-03-20"),
	)
	if err == nil {
		t.Error("expected error for key missing end date")
	}
}
