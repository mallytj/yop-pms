package apierror

import (
	"net/http"
)

// Standard error codes
const (
	CodeValidation   = "VALIDATION_ERROR"
	CodeNotFound     = "NOT_FOUND"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeConflict     = "CONFLICT"
	CodeRateLimited  = "RATE_LIMITED"
	CodeInternal     = "INTERNAL_ERROR"

	// Business logic errors
	CodeRoomUnavailable  = "ROOM_UNAVAILABLE"
	CodeInvalidDateRange = "INVALID_DATE_RANGE"
	CodeOverbooking      = "OVERBOOKING_DETECTED"
	CodePaymentFailed    = "PAYMENT_FAILED"
)

type APIError struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Suggestions map[string]interface{} `json:"suggestions,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

// Constructor helpers
func New(code, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

func (e *APIError) WithDetail(key string, value interface{}) *APIError {
	e.Details[key] = value
	return e
}

func (e *APIError) WithRequestID(requestID string) *APIError {
	e.RequestID = requestID
	return e
}

// HTTP status code mapping
func (e *APIError) StatusCode() int {
	switch e.Code {
	case CodeValidation:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeConflict, CodeRoomUnavailable, CodeOverbooking:
		return http.StatusConflict
	case CodeRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// Pre-defined common errors
func NotFound(resource string) *APIError {
	return New(CodeNotFound, resource+" not found")
}

func Unauthorized(message string) *APIError {
	return New(CodeUnauthorized, message)
}

func ValidationError(message string) *APIError {
	return New(CodeValidation, message)
}

func RoomUnavailable(roomID string, dates string) *APIError {
	return New(CodeRoomUnavailable, "Room is not available for selected dates").
		WithDetail("room_id", roomID).
		WithDetail("dates", dates)
}

func Internal(err error) *APIError {
	return New(CodeInternal, "An internal error occurred").
		WithDetail("error", err.Error())
}
