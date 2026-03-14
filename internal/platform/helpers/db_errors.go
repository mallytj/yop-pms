package helpers

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL error codes
const (
	UniqueViolationCode           = "23505"
	ForeignKeyViolationCode       = "23503"
	CheckViolationCode            = "23514"
	NotNullViolationCode          = "23502"
	RaiseExceptionCode            = "P0001"
	InvalidTextRepresentationCode = "22P02"
	DataExceptionCode             = "22000"
	ExclusionViolationCode        = "23P01"
	UndefinedObjectCode           = "42704"
)

// CheckErrorCode checks if the given error is a pgx.PgError with the specified code.
func CheckErrorCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}
