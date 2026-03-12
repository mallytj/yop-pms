package helpers

import "errors"

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
)

// Custom application error codes
var (
	ErrNotPermitted              = errors.New("operation not permitted for this user")
	ErrNotProvided               = errors.New("required data not provided")
	ErrUniqueViolation           = errors.New("unique constraint violation")
	ErrForeignKeyViolation       = errors.New("foreign key constraint violation")
	ErrCheckViolation            = errors.New("check constraint violation")
	ErrNotNullViolation          = errors.New("not null constraint violation")
	ErrInvalidTextRepresentation = errors.New("invalid text representation")
	ErrDataException             = errors.New("data exception")
	ErrExclusionViolation        = errors.New("exclusion constraint violation")
	ErrRelatedEntityNotFound     = errors.New("related entity not found")
)

// Status colors for planner items
const (
	StatusColorBooked     = "#89d5e5" // Cyan
	StatusColorCheckedIn  = "#008000" // Green
	StatusColorCheckedOut = "#0000FF" // Blue
	StatusColorHold       = "#FFFF00" // Yellow
	StatusColorNoShow     = "#FF0000" // Red
	StatusColorDefault    = "#D3DCDC" // Light Gray
)

// Test Property ID
const TestPropertyID = "00000000-0000-0000-0000-000000000001"
