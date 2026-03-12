package helpers

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"regexp"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrStartingTx      = errors.New("error starting transaction")
	ErrCommittingTx    = errors.New("error committing transaction")
	ErrDuplicatedField = errors.New("duplicated field error")
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

// GetErrorCode extracts the error code from a pgx.PgError if possible, otherwise returns an empty string.
func GetErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
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

// ToPgTstzRange converts a start date and end date to a pgtype.Tstzrange pointer.
// Example usage: ToPgTstzRange(startTime, endTime)
// returns &pgtype.Tstzrange{Lower: startTime, Upper: endTime, Valid: true}
func ToPgTstzRange(start, end time.Time) *pgtype.Range[pgtype.Timestamptz] {
	pgRange := pgtype.Range[pgtype.Timestamptz]{
		Lower:     pgtype.Timestamptz{Time: start, Valid: true},
		Upper:     pgtype.Timestamptz{Time: end, Valid: true},
		Valid:     true,
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive,
	}

	return &pgRange
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

// StructToSlice converts a struct into a slice of its field values, excluding
// any field named "ID".
func StructToSlice(s interface{}) []interface{} {
	v := reflect.ValueOf(s)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	values := make([]interface{}, 0, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == "ID" {
			continue
		}
		values = append(values, v.Field(i).Interface())
	}

	return values
}

// MatchRegex checks if the input string matches the provided regex pattern.
// Returns true if it matches, false otherwise.
// Example usage: MatchRegex("^GRP-\\d{5}$", "GRP-12345") => true
func MatchRegex(pattern, input string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// StringOrEmpty returns the string value if valid, otherwise returns an empty string.
func StringOrEmpty(pgStr pgtype.Text) string {
	if pgStr.Valid {
		return pgStr.String
	}
	return ""
}

// StrToDate parses a date string in "YYYY-MM-DD" format to a time.Time object.
func StrToDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, ErrNotProvided
	}
	const layout = "2006-01-02"
	return time.Parse(layout, dateStr)
}

// ParseDateRange parses start and end date strings in "YYYY-MM-DD" format to time.Time objects.
func ParseDateRange(startStr, endStr string) (time.Time, time.Time, error) {
	startDate, err := StrToDate(startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start date format: %w", err)
	}

	endDate, err := StrToDate(endStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end date format: %w", err)
	}

	return startDate, endDate, nil
}

// PsqlErrToCustomErr maps specific PostgreSQL error codes to custom error messages for better user feedback and HTTP status code handling in API responses.
// Example usage: If a unique constraint violation occurs in the database, this function will return ErrUniqueViolation instead of the raw pgx.PgError,
// allowing handlers to return a 400 Bad Request status with a clear message.
// This function can be extended to handle more PostgreSQL error codes as needed.
func PsqlErrToCustomErr(err error) error {
	// Map specific database errors to more user-friendly error messages
	// Also so we can send different http status codes from the handlers based on the error type
	switch GetErrorCode(err) {
	case UniqueViolationCode:
		return ErrUniqueViolation
	case ForeignKeyViolationCode:
		return ErrForeignKeyViolation
	case CheckViolationCode:
		return ErrCheckViolation
	default:
		return err
	}
}

// CustomErrToHTTPStatus maps custom application errors to appropriate HTTP status codes for API responses.
// Example usage: If a handler returns ErrNotPermitted, this function will return http.StatusForbidden (403),
// allowing the API to respond with the correct status code and message.
// This function can be extended to handle more custom errors as needed, ensuring consistent error handling across the application.
func CustomErrToHTTPStatus(err error) int {
	switch err {
	case ErrNotPermitted:
		return http.StatusForbidden
	case ErrNotProvided:
		return http.StatusBadRequest
	case ErrUniqueViolation, ErrForeignKeyViolation, ErrCheckViolation, ErrNotNullViolation, ErrInvalidTextRepresentation, ErrDataException, ErrExclusionViolation:
		return http.StatusBadRequest
	case ErrRelatedEntityNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// Deref is a generic helper function that safely dereferences a pointer to a value of any type.
// If the pointer is nil, it returns the zero value of the type.
// Example usage: Deref(helpers.Ptr("example")) => "example"
// Example usage: Deref((*string)(nil)) => "" (empty string, the zero value for string)
// Example usage: Deref(helpers.Ptr(42)) => 42
// Example usage: Deref((*int)(nil)) => 0 (the zero value for int)
func Deref[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}

func ToNullUUID(u *uuid.UUID) uuid.NullUUID {
	if u == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *u, Valid: true}
}

func ToNullText[T ~string](s *T) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: string(*s), Valid: true}
}

// NumDaysBetween returns the number of days between two time.Time values.
// Example usage: NumDaysBetween(time.Now(), time.Now().AddDate(0, 0, 1)) => 1
func NumDaysBetween(start, end time.Time) int {
	return int(end.Sub(start).Hours() / 24)
}
