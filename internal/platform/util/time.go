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
