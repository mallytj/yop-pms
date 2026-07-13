// Package types provides shared domain-agnostic types used across packages.
package types

import (
	"fmt"
	"time"
)

// ISO8601Date is a date in YYYY-MM-DD format for JSON serialization.
// Handlers compose TSTZRANGE from property check-in/out times (ADR-012).
type ISO8601Date struct {
	time.Time
}

// String returns the date formatted as YYYY-MM-DD.
func (d ISO8601Date) String() string {
	return d.Format("2006-01-02")
}

// MarshalJSON implements json.Marshaler.
func (d ISO8601Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
// Rejects non-ISO8601 date strings at deserialization time.
func (d *ISO8601Date) UnmarshalJSON(data []byte) error {
	s := string(data)
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return fmt.Errorf("ISO8601Date: expected quoted string, got %s", s)
	}
	s = s[1 : len(s)-1]
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("ISO8601Date: invalid date %q, expected YYYY-MM-DD", s)
	}
	d.Time = t
	return nil
}
