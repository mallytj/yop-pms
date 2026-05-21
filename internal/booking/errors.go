package booking

// Core Requirements: R-RES-CRUD-005, R-RES-CRUD-006, R-RES-VALID-005, R-RES-VALID-007, R-RES-VALID-012, R-RES-VALID-013

import (
	"net/http"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
)

var (
	// ErrVersionMismatch - If-Match header version doesn't match DB row version.
	ErrVersionMismatch = apierror.New("VERSION_MISMATCH",
		"this record was modified by another user; your uncommitted changes may be lost", http.StatusPreconditionFailed)

	// ErrUnassignedItems - items missing room assignment before confirm/checkin.
	ErrUnassignedItems = apierror.New("UNASSIGNED_ITEMS",
		"items missing room assignment", http.StatusConflict).
		WithSuggestions(apierror.Suggestions{"assign a room to each item before confirming"})

	// ErrTerminal - reservation is in a terminal state (cancelled/archived).
	ErrTerminal = apierror.New("TERMINAL_RESERVATION",
		"reservation is in a terminal state", http.StatusConflict)

	// ErrOutstandingBalance - folio balance must be zero before checkout.
	// Stub in this PR (always checks balance=0). Finance PR implements real check.
	ErrOutstandingBalance = apierror.New("OUTSTANDING_FOLIO_BALANCE",
		"folio balance must be zero", http.StatusConflict).
		WithSuggestions(apierror.Suggestions{"settle the outstanding folio balance before checkout"})

	// ErrHasCheckedInItems - attempted cancel/checkout while items are checked in.
	ErrHasCheckedInItems = apierror.New("RESERVATION_HAS_CHECKED_IN_ITEMS",
		"checkout or cancel checked-in items first", http.StatusConflict)

	// ErrDoNotMove - do-not-move flag prevents room reassignment.
	ErrDoNotMove = apierror.New("DO_NOT_MOVE",
		"do-not-move flag is set on this item", http.StatusForbidden).
		WithSuggestions(apierror.Suggestions{"remove the do-not-move flag or contact a manager"})

	// ErrRatePlanCapacity - rate plan daily capacity exceeded.
	ErrRatePlanCapacity = apierror.New("RATE_PLAN_CAPACITY_EXCEEDED",
		"rate plan daily capacity exceeded", http.StatusConflict).
		WithSuggestions(apierror.Suggestions{"select a different rate plan or reduce the stay length"})

	// ErrInvalidTransition - state machine transition not allowed.
	ErrInvalidTransition = apierror.New("INVALID_TRANSITION",
		"state transition not allowed", http.StatusConflict)

	// ErrNoGuest - reservation is missing a guest reference.
	ErrNoGuest = apierror.New("NO_GUEST",
		"reservation must have a guest attached", http.StatusBadRequest)

	// ErrSourceDeferred - this source type is not yet implemented in the API.
	// NOTE: This needs to be removed or replaced with a more specific error
	// (e.g. ErrSourceOTA) when additional sources are supported.
	ErrSourceDeferred = apierror.New("SOURCE_DEFERRED",
		"this source type is not yet implemented in the API", http.StatusNotImplemented)

	// ErrNoPropertyContext - X-Property-ID header missing or invalid.
	ErrNoPropertyContext = apierror.New("NO_PROPERTY_CONTEXT",
		"X-Property-ID header is required", http.StatusBadRequest).
		WithSuggestions(apierror.Suggestions{"try refreshing your page - if you have further issues contact support"})

	// ErrMissingPermission - caller lacks required permission.
	ErrMissingPermission = apierror.New("MISSING_PERMISSION",
		"insufficient permissions for this action", http.StatusForbidden)

	// ErrGuestNotAttached - confirm requires a guest.
	ErrGuestNotAttached = apierror.New("GUEST_NOT_ATTACHED",
		"confirm requires a guest to be attached to the reservation", http.StatusConflict)

	// ErrHoldExpired - reservation hold has expired.
	ErrHoldExpired = apierror.New("HOLD_EXPIRED",
		"reservation hold has expired", http.StatusGone)

	// ErrRoomNotAvailable - requested room not available for the dates.
	ErrRoomNotAvailable = apierror.New("ROOM_NOT_AVAILABLE",
		"requested room is not available for the specified dates", http.StatusConflict)

	// ErrPaymentsNotSupported - payment operations not supported in this PR.
	// NOTE: this must be removed when implementing payment operations
	// (R-RES-CRUD-007, R-RES-CRUD-008, R-RES-CRUD-009).
	// However a ErrPaymentTypeNotSupported error may still be needed if only certain
	// payment types are supported.
	ErrPaymentsNotSupported = apierror.New("PAYMENTS_NOT_SUPPORTED",
		"payment operations are not yet supported", http.StatusNotImplemented)

	// ErrInvalidDates - stay period dates violate business rules.
	ErrInvalidDates = apierror.New("INVALID_DATES",
		"stay period dates violate business rules", http.StatusBadRequest)

	// ErrForbiddenSource - source is not allowed for this operation (e.g. OTA via API).
	ErrForbiddenSource = apierror.New("FORBIDDEN_SOURCE",
		"this operation is not allowed for the reservation's source", http.StatusForbidden)
)
