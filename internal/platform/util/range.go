package util

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ToRange builds a TSTZRANGE with inclusive lower and exclusive upper bounds.
func ToRange(lower, upper time.Time) pgtype.Range[pgtype.Timestamptz] {
	return pgtype.Range[pgtype.Timestamptz]{
		Lower:     pgtype.Timestamptz{Time: lower, Valid: true},
		Upper:     pgtype.Timestamptz{Time: upper, Valid: true},
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive,
		Valid:     true,
	}
}

// FormatRange renders a TSTZRANGE using its actual bound types.
func FormatRange(r pgtype.Range[pgtype.Timestamptz]) string {
	open := "["
	if r.LowerType == pgtype.Exclusive {
		open = "("
	}
	closeBracket := ")"
	if r.UpperType == pgtype.Inclusive {
		closeBracket = "]"
	}
	return fmt.Sprintf("%s%s,%s%s", open, TSToTime(r.Lower).Format(time.RFC3339), TSToTime(r.Upper).Format(time.RFC3339), closeBracket)
}
