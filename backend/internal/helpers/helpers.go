package helpers

import (
	"errors"
	"fmt"
	"net/http"

	"regexp"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrStartingTx            = errors.New("error starting transaction")
	ErrCommittingTx          = errors.New("error committing transaction")
	ErrDuplicatedField       = errors.New("duplicated field error")
	ErrRelatedEntityNotFound = errors.New("related entity not found")
	// ErrCommitingTx is deprecated: use ErrCommittingTx instead
	ErrCommitingTx = ErrCommittingTx
)

// CheckErrorCode checks if the given error is a pgx.PgError with the specified code.
func CheckErrorCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}

// StringCharCount returns the number of characters in a string, accounting for multi-byte characters.
func StringCharCount(s string) int {
	return utf8.RuneCountInString(s)
}

// IsValidEmail performs a regex check to validate email format.
func IsValidEmail(email string) bool {
	// Simple regex for demonstration purposes; consider using a more robust regex for production use.
	// Checks for general email format: local-part@domain
	const emailRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}

// Ptr is a generic helper function that returns a pointer to the given value.
// Example: Ptr("example") returns *string
// Usage: s := helpers.Ptr("example")
func Ptr[T any](v T) *T {
	return &v
}

// ToPgText converts a string pointer to pgtype.Text, handling nil values appropriately.
// If the input pointer is nil, it returns a pgtype.Text with Valid set to false.
// Example usage: ToPgText(helpers.Ptr("example"))
// returns pgtype.Text{String: "example", Valid: true}
// ToPgText(nil) returns pgtype.Text{Valid: false}
func ToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// ToPgTextPtr converts a string to a pgtype.Text pointer.
// Example usage: ToPgTextPtr("example")
// returns &pgtype.Text{String: "example", Valid: true}
func ToPgTextPtr(s string) *pgtype.Text {
	pgText := pgtype.Text{String: s, Valid: true}
	return &pgText
}

// ToPgUUID converts a UUID pointer to pgtype.UUID, handling nil values appropriately.
// If the input pointer is nil, it returns a pgtype.UUID with Valid set to false.
// Example usage: ToPgUUID(helpers.Ptr(uuid.New()))
// returns pgtype.UUID{Bytes: <UUID>, Valid: true}
// ToPgUUID(nil) returns pgtype.UUID{Valid: false}
func ToPgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// ToPgBool converts a bool pointer to pgtype.Bool, handling nil values appropriately.
// If the input pointer is nil, it returns a pgtype.Bool with Valid set to false.
// Example usage: ToPgBool(helpers.Ptr(true))
// returns pgtype.Bool{Bool: true, Valid: true}
// ToPgBool(nil) returns pgtype.Bool{Valid: false}
func ToPgBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{Valid: false}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

// ToPgBoolPtr converts a bool to a pgtype.Bool pointer.
// Example usage: ToPgBoolPtr(true)
// returns &pgtype.Bool{Bool: true, Valid: true}
func ToPgBoolPtr(b bool) *pgtype.Bool {
	pgBool := pgtype.Bool{Bool: b, Valid: true}
	return &pgBool
}

// ExtractAndParseUUIDParam extracts a parameter from the URL and parses it as a UUID.
// Returns the parsed UUID or an error if extraction or parsing fails.
// Example usage: ExtractAndParseUUIDParam(r, "userID") => uuid.UUID, nil
// Example usage: ExtractAndParseUUIDParam(r, "invalidParam") => uuid.Nil, error
func ExtractAndParseUUIDParam(r *http.Request, param string) (uuid.UUID, error) {
	// Extract the parameter from the URL
	rawParam := chi.URLParam(r, param)

	// Parse the parameter as a UUID
	userID, err := uuid.Parse(rawParam)
	if err != nil {
		// Return an error if parsing fails
		return uuid.Nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Return the parsed UUID
	return userID, nil
}

// ParamIsProvided checks if a string pointer parameter is provided (not nil and not empty).
// Example usage: ParamIsProvided(helpers.Ptr("value")) => true
// Example usage: ParamIsProvided(helpers.Ptr("")) => false
// Example usage: ParamIsProvided(nil) => false
// Returns true if the parameter is provided, false otherwise.
func ParamIsProvided(param *string) bool {
	return param != nil && *param != ""
}
