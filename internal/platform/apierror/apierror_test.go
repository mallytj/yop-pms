package apierror

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
)

func TestNew(t *testing.T) {
	err := New("TEST_CODE", "test message", http.StatusBadRequest)

	if err.Code != "TEST_CODE" {
		t.Errorf("Code: got %q, want %q", err.Code, "TEST_CODE")
	}
	if err.Message != "test message" {
		t.Errorf("Message: got %q, want %q", err.Message, "test message")
	}
	if err.Status != http.StatusBadRequest {
		t.Errorf("Status: got %d, want %d", err.Status, http.StatusBadRequest)
	}
}

func TestError(t *testing.T) {
	err := New("TEST_CODE", "test message", http.StatusBadRequest)

	if err.Error() != "test message" {
		t.Errorf("Error(): got %q, want %q", err.Error(), "test message")
	}
}

func TestWithMessage(t *testing.T) {
	original := ErrConflict
	customized := original.WithMessage("custom message")

	// Check that the new error has the custom message
	if customized.Message != "custom message" {
		t.Errorf("Message: got %q, want %q", customized.Message, "custom message")
	}

	// Check that code and status are preserved
	if customized.Code != original.Code {
		t.Errorf("Code: got %q, want %q", customized.Code, original.Code)
	}
	if customized.Status != original.Status {
		t.Errorf("Status: got %d, want %d", customized.Status, original.Status)
	}

	// Check that original is not mutated
	if original.Message != "resource already exists" {
		t.Errorf("Original mutated: got %q, want %q", original.Message, "resource already exists")
	}
}

func TestWithSuggestions(t *testing.T) {
	original := ErrConflict
	customized := original.WithSuggestions([]string{"suggestion 1", "suggestion 2"})

	if customized.Suggestions[0] != "suggestion 1" {
		t.Errorf("Suggestions[0]: got %q, want %q", customized.Suggestions[0], "suggestion 1")
	}
	if customized.Suggestions[1] != "suggestion 2" {
		t.Errorf("Suggestions[1]: got %q, want %q", customized.Suggestions[1], "suggestion 2")
	}
	if customized.Code != original.Code {
		t.Errorf("Code: got %q, want %q", customized.Code, original.Code)
	}
	if customized.Status != original.Status {
		t.Errorf("Status: got %d, want %d", customized.Status, original.Status)
	}

	if original.Suggestions != nil {
		t.Errorf("Original should not have suggestions: got %v, want nil", original.Suggestions)
	}
}

func TestWithSuggestions_NilInput(t *testing.T) {
	original := ErrConflict
	customized := original.WithSuggestions(nil)

	if customized.Suggestions != nil {
		t.Errorf("Suggestions: got %v, want nil", customized.Suggestions)
	}
	if customized.Code != original.Code {
		t.Errorf("Code: got %q, want %q", customized.Code, original.Code)
	}
	if customized.Status != original.Status {
		t.Errorf("Status: got %d, want %d", customized.Status, original.Status)
	}
}

func TestSentinels(t *testing.T) {
	tests := map[string]struct {
		err    *APIError
		code   string
		status int
	}{
		"ErrNotFound":      {ErrNotFound, "NOT_FOUND", http.StatusNotFound},
		"ErrBadRequest":    {ErrBadRequest, "BAD_REQUEST", http.StatusBadRequest},
		"ErrConflict":      {ErrConflict, "CONFLICT", http.StatusConflict},
		"ErrInternal":      {ErrInternal, "INTERNAL_ERROR", http.StatusInternalServerError},
		"ErrUnprocessable": {ErrUnprocessable, "UNPROCESSABLE_ENTITY", http.StatusUnprocessableEntity},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Code: got %q, want %q", tt.err.Code, tt.code)
			}
			if tt.err.Status != tt.status {
				t.Errorf("Status: got %d, want %d", tt.err.Status, tt.status)
			}
		})
	}
}

func TestMapPostgresError_NilInput(t *testing.T) {
	result := MapPostgresError(nil)
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}
}

func TestMapPostgresError_UniqueViolation(t *testing.T) {
	err := &pgconn.PgError{Code: helpers.UniqueViolationCode}
	result := MapPostgresError(err)

	if result.Code != ErrConflict.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrConflict.Code)
	}
	if result.Status != http.StatusConflict {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusConflict)
	}
	if result.Message != "a record with this value already exists" {
		t.Errorf("Message: got %q, want %q", result.Message, "a record with this value already exists")
	}
}

func TestMapPostgresError_ForeignKeyViolation(t *testing.T) {
	err := &pgconn.PgError{Code: helpers.ForeignKeyViolationCode}
	result := MapPostgresError(err)

	if result.Code != ErrBadRequest.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrBadRequest.Code)
	}
	if result.Status != http.StatusBadRequest {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusBadRequest)
	}
	if result.Message != "referenced resource does not exist" {
		t.Errorf("Message: got %q, want %q", result.Message, "referenced resource does not exist")
	}
}

func TestMapPostgresError_CheckViolation(t *testing.T) {
	err := &pgconn.PgError{Code: helpers.CheckViolationCode}
	result := MapPostgresError(err)

	if result.Code != ErrUnprocessable.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrUnprocessable.Code)
	}
	if result.Status != http.StatusUnprocessableEntity {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusUnprocessableEntity)
	}
	if result.Message != "the value violates a database constraint" {
		t.Errorf("Message: got %q, want %q", result.Message, "the value violates a database constraint")
	}
}

func TestMapPostgresError_ExclusionViolation(t *testing.T) {
	err := &pgconn.PgError{Code: helpers.ExclusionViolationCode}
	result := MapPostgresError(err)

	if result.Code != ErrConflict.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrConflict.Code)
	}
	if result.Status != http.StatusConflict {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusConflict)
	}
	if result.Message != "the dates overlap with an existing reservation" {
		t.Errorf("Message: got %q, want %q", result.Message, "the dates overlap with an existing reservation")
	}
}

func TestMapPostgresError_RaiseException_WithDetail(t *testing.T) {
	err := &pgconn.PgError{
		Code:   helpers.RaiseExceptionCode,
		Detail: "custom error detail",
	}
	result := MapPostgresError(err)

	if result.Code != ErrUnprocessable.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrUnprocessable.Code)
	}
	if result.Message != "custom error detail" {
		t.Errorf("Message: got %q, want %q", result.Message, "custom error detail")
	}
}

func TestMapPostgresError_RaiseException_WithoutDetail(t *testing.T) {
	err := &pgconn.PgError{Code: helpers.RaiseExceptionCode}
	result := MapPostgresError(err)

	if result.Code != ErrUnprocessable.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrUnprocessable.Code)
	}
	if result.Message != "the request violates a business rule" {
		t.Errorf("Message: got %q, want %q", result.Message, "the request violates a business rule")
	}
}

func TestMapPostgresError_UnknownCode(t *testing.T) {
	err := &pgconn.PgError{Code: "99999"}
	result := MapPostgresError(err)

	if result.Code != ErrInternal.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrInternal.Code)
	}
	if result.Status != http.StatusInternalServerError {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusInternalServerError)
	}
}

func TestMapPostgresError_NonPgError(t *testing.T) {
	err := fmt.Errorf("some other error")
	result := MapPostgresError(err)

	if result.Code != ErrInternal.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrInternal.Code)
	}
	if result.Status != http.StatusInternalServerError {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusInternalServerError)
	}
}

func TestMapStoreError_NoRows(t *testing.T) {
	result := MapStoreError(pgx.ErrNoRows)

	if result.Code != ErrNotFound.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrNotFound.Code)
	}
	if result.Status != http.StatusNotFound {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusNotFound)
	}
}

func TestMapStoreError_PostgresError(t *testing.T) {
	err := &pgconn.PgError{Code: helpers.UniqueViolationCode}
	result := MapStoreError(err)

	if result.Code != ErrConflict.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrConflict.Code)
	}
	if result.Status != http.StatusConflict {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusConflict)
	}
}

func TestMapStoreError_GenericError(t *testing.T) {
	err := fmt.Errorf("connection reset")
	result := MapStoreError(err)

	if result.Code != ErrInternal.Code {
		t.Errorf("Code: got %q, want %q", result.Code, ErrInternal.Code)
	}
	if result.Status != http.StatusInternalServerError {
		t.Errorf("Status: got %d, want %d", result.Status, http.StatusInternalServerError)
	}
}
