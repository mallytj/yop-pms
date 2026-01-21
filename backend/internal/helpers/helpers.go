package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/types"

	"regexp"
	"testing"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

var (
	ErrStartingTx      = errors.New("error starting transaction")
	ErrCommitingTx     = errors.New("error committing transaction")
	ErrDuplicatedField = errors.New("duplicated field error")
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

// BuildAndServeHttpRequest is a helper function to build and serve an HTTP request.
// method: HTTP method (GET, POST, etc.)
// url: Request URL
// body: Request body (can be nil)
// r: chi.Mux router to serve the request
// Returns the ResponseRecorder
// Example: BuildAndServeHttpRequest("POST", "/users", params, r) => *httptest.ResponseRecorder
func BuildAndServeHttpRequest(method string, url string, body interface{}, r *chi.Mux) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewReader(jsonBody)
	} else {
		reqBody = bytes.NewReader([]byte{})
	}
	req := httptest.NewRequest(method, url, reqBody)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	return rr
}

// CreateTestLicence is a helper function to create a test licence with the given licence key.
// Returns the created licence.
// licenceKey: Must be in the format "XXX-YYYY" where X is uppercase letter and Y is digit.
// testQueries: Database queries interface for executing SQL commands.
// Example: CreateTestLicence(t, "TEST-1234") = repo.Licence{...}
func CreateTestLicence(t *testing.T, licenceKey string, testQueries *repo.Queries) repo.Licence {
	ctx := context.Background()
	lic, err := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
		LicenceKey:       licenceKey,
		OrganisationName: "Test Organisation",
		ContactEmail:     "test@example.com",
	})
	require.NoError(t, err)
	return lic
}

// CreateTestUser is a helper function to create a test user with the given parameters.
// Returns the created user.
// params: Parameters required to create the user.
// testQueries: Database queries interface for executing SQL commands.
// Example: CreateTestUser(t, params, testQueries) = repo.User{...}
func CreateTestUser(t *testing.T, params types.CreateUserParams, testQueries *repo.Queries) repo.User {
	ctx := context.Background()

	// Go service route to create first user directly
	user, err := testQueries.CreateUser(ctx, repo.CreateUserParams{
		LicenceID:    ToPgUUID(&params.LicenceID),
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: params.Password,
		FirstName:    params.FirstName,
		LastName:     params.LastName,
		Role:         string(params.Role),
		IsActive:     ToPgBool(&params.IsActive),
	})

	// Ensure no error occurred during first user creation
	require.NoError(t, err, fmt.Sprintf("failed to create test user: %v", err))

	return user
}

// CreateTestProperty is a helper function to create a test property with the given parameters.
// Returns the created property.
// params: Parameters required to create the property.
// testQueries: Database queries interface for executing SQL commands.
// Example: CreateTestProperty(t, params, testQueries) = repo.Property{...}
func CreateTestProperty(t *testing.T, params repo.CreatePropertyParams, testQueries *repo.Queries) repo.Property {
	// Create context
	ctx := context.Background()

	// Create property in the database
	property, err := testQueries.CreateProperty(ctx, params)

	// Ensure no error occurred during property creation
	require.NoError(t, err, fmt.Sprintf("failed to create test property: %v", err))

	// Return the created property
	return property
}

// ParamIsProvided checks if a string pointer parameter is provided (not nil and not empty).
// Example usage: ParamIsProvided(helpers.Ptr("value")) => true
// Example usage: ParamIsProvided(helpers.Ptr("")) => false
// Example usage: ParamIsProvided(nil) => false
// Returns true if the parameter is provided, false otherwise.
func ParamIsProvided(param *string) bool {
	return param != nil && *param != ""
}
