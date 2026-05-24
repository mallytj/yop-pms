// Package util is for global helpers to be reused across the backend
package util

import "github.com/jackc/pgx/v5/pgtype"

// NullText returns the string value of a nullable text, or empty string.
func NullText(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}
