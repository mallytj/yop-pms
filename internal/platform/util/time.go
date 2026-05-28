package util

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// TSToTime converts a nullable timestamptz to time.Time.
// Returns zero time if not valid.
func TSToTime(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}

// TSToTimePtr converts a nullable timestamptz to a time pointer.
// Returns nil if not valid.
func TSToTimePtr(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

// NightsBetween returns all calendar dates from arrival to departure (exclusive).
func NightsBetween(arrival, departure time.Time) []time.Time {
	var dates []time.Time
	current := arrival.Truncate(24 * time.Hour)
	end := departure.Truncate(24 * time.Hour)
	for current.Before(end) {
		dates = append(dates, current)
		current = current.Add(24 * time.Hour)
	}
	return dates
}

// DatesToPGDates converts a slice of time.Time to a slice of pgtype.Date.
// All dates are marked Valid. The output slice is a copy.
func DatesToPGDates(dates []time.Time) []pgtype.Date {
	pgDates := make([]pgtype.Date, len(dates))
	for i, d := range dates {
		pgDates[i] = pgtype.Date{Time: d, Valid: true}
	}
	return pgDates
}
