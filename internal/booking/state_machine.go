package booking

// Core Requirements: R-RES-CRUD-005, R-RES-CRUD-006, R-RES-VALID-006, R-RES-VALID-007, ADR-015, §7.4

import (
	"fmt"
)

// These are direct transitions set by business actions.
// Statuses derived from item rollup (checked_in, checked_out) are NOT listed here
// — they are computed by ADR-015 rollup.

var reservationTransitions = map[ReservationStatus]map[ReservationStatus]bool{
	StatusHold: {
		StatusConfirmed: true,
		StatusCancelled: true, // staff cancel / worker expiry
	},
	StatusConfirmed: {
		StatusCancelled: true,
	},
	// no_show→cancelled: handled by direct cancel on the reservation level
	// no_show is an item-level status; reservation-level no_show goes via item rollup
	StatusCheckedIn: {
		StatusCancelled: true, // special: cancelled via CancelReservation when items
		// not all checked_out (i.e. shorten stay)
	},
	StatusCheckedOut: {
		StatusArchived: true, // archival worker
	},
	StatusCancelled: {
		StatusConfirmed: true, // reactivation
	},
	StatusArchived: {}, // terminal
	// NOTE: pending_cancellation→cancelled will be added in finance PR.
	StatusPendingCancellation: {}, // terminal (no code path emits it this PR)
}

// ValidateReservationTransition checks if a direct reservation-level transition is allowed.
// Returns ErrInvalidTransition if disallowed.
// Does NOT cover rollup-driven transitions (checked_in, checked_out).
func ValidateReservationTransition(from, to ReservationStatus) error {
	allowed, ok := reservationTransitions[from]
	if !ok {
		return ErrInvalidTransition.WithMessage(fmt.Sprintf("reservation status %q has no defined transitions", from))
	}
	if !allowed[to] {
		return ErrInvalidTransition.WithMessage(fmt.Sprintf("transition from %q to %q is not allowed", from, to))
	}
	return nil
}

var itemTransitions = map[ItemStatus]map[ItemStatus]bool{
	ItemStatusBooked: {
		ItemStatusCheckedIn: true,
		ItemStatusNoShow:    true,
		ItemStatusCancelled: true, // cancel before check-in
	},
	ItemStatusCheckedIn: {
		ItemStatusCheckedOut: true,
		ItemStatusOverstay:   true, // overstay worker
		ItemStatusNoShow:     true, // no-show override (rare)
		ItemStatusCancelled:  true, // cancel item while checked in (cancels future nights via ShortenStay)
	},
	ItemStatusCheckedOut: {
		ItemStatusArchived: true, // archival worker
	},
	ItemStatusNoShow: {
		ItemStatusCancelled: true, // staff closes no-show record
	},
	ItemStatusOverstay: {
		ItemStatusCheckedOut: true, // resolve overstay → check-out
		ItemStatusCancelled:  true, // staff closes stale overstay (guest left without checkout)
	},
	ItemStatusCancelled: {
		ItemStatusBooked: true, // reactivation of a cancelled item
	},
	ItemStatusArchived: {},
}

// ValidateItemTransition checks if an item-level status transition is allowed.
func ValidateItemTransition(from, to ItemStatus) error {
	allowed, ok := itemTransitions[from]
	if !ok {
		return ErrInvalidTransition.WithMessage(fmt.Sprintf("item status %q has no defined transitions", from))
	}
	if !allowed[to] {
		return ErrInvalidTransition.WithMessage(fmt.Sprintf("item transition from %q to %q is not allowed", from, to))
	}
	return nil
}

// RollupReservationStatus computes the derived reservation status from item statuses.
// Per ADR-015: first match wins.
//
//	All items cancelled → cancelled
//	All items terminal AND >=1 checked_out → checked_out
//	>=1 item checked_in AND no items booked → checked_in
//	Otherwise → unchanged
func RollupReservationStatus(current ReservationStatus, items []ItemStatus) RollupResult {
	if len(items) == 0 {
		return RollupResult{Status: StatusCancelled, Changed: current != StatusCancelled}
	}

	var (
		total      = len(items)
		terminal   int
		checkedOut int
		checkedIn  int
		booked     int
		cancelled  int
		noShow     int
		overstay   int
	)

	for _, s := range items {
		switch s {
		case ItemStatusCheckedOut:
			terminal++
			checkedOut++
		case ItemStatusNoShow:
			terminal++
			noShow++
		case ItemStatusCancelled:
			terminal++
			cancelled++
		case ItemStatusArchived:
			terminal++
		case ItemStatusCheckedIn:
			checkedIn++
		case ItemStatusOverstay:
			terminal++
			overstay++
		case ItemStatusBooked:
			booked++
		}
	}

	// Rule 1: All items cancelled (or archived-from-cancel)
	if cancelled == total {
		return RollupResult{Status: StatusCancelled, Changed: current != StatusCancelled}
	}

	// Rule 2: All items terminal AND >=1 checked_out
	if terminal == total && checkedOut > 0 {
		return RollupResult{Status: StatusCheckedOut, Changed: current != StatusCheckedOut}
	}

	// Rule 3: >=1 item checked_in AND no items in booked
	if checkedIn > 0 && booked == 0 {
		return RollupResult{Status: StatusCheckedIn, Changed: current != StatusCheckedIn}
	}

	// Rule 4: unchanged (includes all-overstay, mixed cancelled+noShow, mixed booked+checkedIn)
	return RollupResult{Status: current, Changed: false}
}

// --- Action-level idempotency (§7.4) ---
type idempotencyKey struct {
	Endpoint      string
	DesiredStatus any
	CurrentStatus any
}

// ActionIdempotency determines whether an action is a no-op or a conflict
// based on current vs desired state. Returns:
//   - noOp=true: action was already applied (handler → 200 with current body)
//   - conflict=true: current state forbids the action (handler → 409)
//   - both false: action should proceed normally
//
// Lookup table per §7.4 (endpoint × desiredStatus × currentStatus → outcome).
func ActionIdempotency(endpoint string, desiredStatus, currentStatus any) (noOp, conflict bool) {
	key := idempotencyKey{Endpoint: endpoint, DesiredStatus: desiredStatus, CurrentStatus: currentStatus}

	switch key {
	// --- Reservation confirm ---
	case idempotencyKey{"confirm", StatusConfirmed, StatusConfirmed}:
		return true, false
	case idempotencyKey{"confirm", StatusConfirmed, StatusHold}:
		return false, false
	case idempotencyKey{"confirm", StatusConfirmed, StatusCancelled}:
		return false, true
	// confirm is invalid for: checked_in, checked_out, archived

	// --- Reservation cancel ---
	case idempotencyKey{"cancel", StatusCancelled, StatusCancelled}:
		return true, false
	case idempotencyKey{"cancel", StatusCancelled, StatusArchived}:
		return false, true
	case idempotencyKey{"cancel", StatusCancelled, StatusHold}:
		return false, false
	case idempotencyKey{"cancel", StatusCancelled, StatusConfirmed}:
		return false, false
	case idempotencyKey{"cancel", StatusCancelled, StatusCheckedIn}:
		return false, false

	// --- Item check-in ---
	case idempotencyKey{"checkin_item", ItemStatusCheckedIn, ItemStatusCheckedIn}:
		return true, false
	case idempotencyKey{"checkin_item", ItemStatusCheckedIn, ItemStatusCheckedOut}:
		return false, true
	case idempotencyKey{"checkin_item", ItemStatusCheckedIn, ItemStatusCancelled}:
		return false, true
	case idempotencyKey{"checkin_item", ItemStatusCheckedIn, ItemStatusBooked}:
		return false, false

	// --- Item check-out ---
	case idempotencyKey{"checkout_item", ItemStatusCheckedOut, ItemStatusCheckedOut}:
		return true, false
	case idempotencyKey{"checkout_item", ItemStatusCheckedOut, ItemStatusCancelled}:
		return false, true
	case idempotencyKey{"checkout_item", ItemStatusCheckedOut, ItemStatusCheckedIn}:
		return false, false

	// --- Item cancel ---
	case idempotencyKey{"cancel_item", ItemStatusCancelled, ItemStatusCancelled}:
		return true, false
	case idempotencyKey{"cancel_item", ItemStatusCancelled, ItemStatusCheckedOut}:
		return false, true

	// --- Mark no-show ---
	case idempotencyKey{"no_show", ItemStatusNoShow, ItemStatusNoShow}:
		return true, false
	case idempotencyKey{"no_show", ItemStatusNoShow, ItemStatusBooked}:
		return false, false
	case idempotencyKey{"no_show", ItemStatusNoShow, ItemStatusCheckedIn}:
		return false, false
	case idempotencyKey{"no_show", ItemStatusNoShow, ItemStatusCancelled}:
		return false, true

	// --- Reactivate ---
	case idempotencyKey{"reactivate", StatusConfirmed, StatusConfirmed}:
		return true, false
	case idempotencyKey{"reactivate", StatusConfirmed, StatusCancelled}:
		return false, false

	default:
		return false, false
	}
}

// IsTerminalItemStatus returns true if the item status is terminal
// (no automatic transitions out of it).
func IsTerminalItemStatus(s ItemStatus) bool {
	switch s {
	case ItemStatusCheckedOut, ItemStatusCancelled, ItemStatusArchived:
		return true
	default:
		return false
	}
}

// IsTerminalReservationStatus returns true if the reservation status is terminal.
func IsTerminalReservationStatus(s ReservationStatus) bool {
	switch s {
	case StatusCancelled, StatusArchived:
		return true
	default:
		return false
	}
}
