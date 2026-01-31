package helpers

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"regexp"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
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

// Lpad pads the input string s with the padStr character on the left until it reaches the overallLength.
// If s is already longer than or equal to overallLength, it returns s unchanged.
func Lpad(s, padStr string, overallLength int) string {
	if len(s) >= overallLength {
		return s
	}
	padLength := overallLength - len(s)
	buffer := make([]rune, padLength)
	for i := 0; i < padLength; i++ {
		buffer[i] = rune(padStr[0]) // Assuming padStr is a single character
	}
	return string(buffer) + s
}

// hashPassword hashes the provided password using bcrypt.
// Returns the hashed password as a string or an error if hashing fails.
// Example usage: hashPassword("mysecretpassword") => "$2a$10$EixZaYVK1fsbw1ZfbX3OXePaWxn96p36Z1Z6Fh5j6K5j6K5j6K5j6"
func HashPassword(password string) (string, error) {
	// Generate the bcrypt hash of the password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// Handle potential error during hashing
	if err != nil {
		return "", err
	}

	// Convert hashed bytes to string
	hashedPassword := string(hashedBytes)

	// Return the hashed password
	return hashedPassword, nil
}

// StructToSlice converts a struct into a slice of its field values. Exclude ID
func StructToSlice(s interface{}) []interface{} {
	v := reflect.ValueOf(s)

	// If a pointer is passed, get the underlying element
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	values := make([]interface{}, v.NumField()) // Exclude ID field

	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == "ID" {
			continue
		}
		values[i] = v.Field(i).Interface()
	}

	return values
}
