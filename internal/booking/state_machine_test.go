package booking

import (
	"testing"
)

type StatusConstraint interface {
	ReservationStatus | ItemStatus
}

type TransitionTest[T StatusConstraint] struct {
	name string
	from T
	to   T
}

// R-RES-VALID-006: Status transitions follow state machine (§7.1).
func TestValidateReservationTransition_Valid(t *testing.T) {
	tests := []TransitionTest[ReservationStatus]{
		{"hold→confirmed", StatusHold, StatusConfirmed},
		{"hold→cancelled", StatusHold, StatusCancelled},
		{"confirmed→cancelled", StatusConfirmed, StatusCancelled},
		{"checked_out→archived", StatusCheckedOut, StatusArchived},
		{"cancelled→confirmed", StatusCancelled, StatusConfirmed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateReservationTransition(tt.from, tt.to); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// R-RES-VALID-006: Invalid transitions rejected.
// R-RES-VALID-007: Cancelled reservation not mutable except via reactivation.
func TestValidateReservationTransition_Invalid(t *testing.T) {
	tests := []TransitionTest[ReservationStatus]{
		{"hold→checked_in", StatusHold, StatusCheckedIn},
		{"hold→archived", StatusHold, StatusArchived},
		{"confirmed→archived", StatusConfirmed, StatusArchived},
		{"confirmed→hold", StatusConfirmed, StatusHold},
		{"cancelled→hold", StatusCancelled, StatusHold},
		{"archived→confirmed", StatusArchived, StatusConfirmed},
		{"checked_in→archived", StatusCheckedIn, StatusArchived},
		{"checked_in→cancelled", StatusCheckedIn, StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateReservationTransition(tt.from, tt.to); err == nil {
				t.Errorf("expected error for transition %s→%s, got nil", tt.from, tt.to)
			}
		})
	}
}

func TestValidateReservationTransition_UnknownSource(t *testing.T) {
	if err := ValidateReservationTransition("unknown", StatusConfirmed); err == nil {
		t.Error("expected error for unknown source status")
	}
}

// --- Item transition tests ---

// R-RES-VALID-006: Item state transitions per §7.2.
func TestValidateItemTransition_Valid(t *testing.T) {
	tests := []TransitionTest[ItemStatus]{
		{"booked→checked_in", ItemStatusBooked, ItemStatusCheckedIn},
		{"booked→no_show", ItemStatusBooked, ItemStatusNoShow},
		{"booked→cancelled", ItemStatusBooked, ItemStatusCancelled},
		{"checked_in→checked_out", ItemStatusCheckedIn, ItemStatusCheckedOut},
		{"checked_in→overstay", ItemStatusCheckedIn, ItemStatusOverstay},
		{"no_show→cancelled", ItemStatusNoShow, ItemStatusCancelled},
		{"overstay→checked_in", ItemStatusOverstay, ItemStatusCheckedIn},
		{"overstay→checked_out", ItemStatusOverstay, ItemStatusCheckedOut},
		{"cancelled→booked", ItemStatusCancelled, ItemStatusBooked},
		{"checked_out→archived", ItemStatusCheckedOut, ItemStatusArchived},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateItemTransition(tt.from, tt.to); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateItemTransition_Invalid(t *testing.T) {
	tests := []TransitionTest[ItemStatus]{
		{"booked→overstay", ItemStatusBooked, ItemStatusOverstay},
		{"booked→archived", ItemStatusBooked, ItemStatusArchived},
		{"checked_in→cancelled", ItemStatusCheckedIn, ItemStatusCancelled},
		{"checked_in→booked", ItemStatusCheckedIn, ItemStatusBooked},
		{"overstay→cancelled", ItemStatusOverstay, ItemStatusCancelled},
		{"checked_out→booked", ItemStatusCheckedOut, ItemStatusBooked},
		{"checked_out→checked_in", ItemStatusCheckedOut, ItemStatusCheckedIn},
		{"cancelled→checked_in", ItemStatusCancelled, ItemStatusCheckedIn},
		{"archived→booked", ItemStatusArchived, ItemStatusBooked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateItemTransition(tt.from, tt.to); err == nil {
				t.Errorf("expected error for transition %s→%s, got nil", tt.from, tt.to)
			}
		})
	}
}

// --- Rollup tests (ADR-009) ---

func TestRollupReservationStatus_EmptyItems(t *testing.T) {
	result := RollupReservationStatus(StatusConfirmed, nil)
	if result.Status != StatusCancelled {
		t.Errorf("expected cancelled, got %s", result.Status)
	}
	if !result.Changed {
		t.Error("expected changed=true for empty items")
	}
}

func TestRollupReservationStatus_AllCancelled(t *testing.T) {
	result := RollupReservationStatus(StatusConfirmed, []ItemStatus{ItemStatusCancelled, ItemStatusCancelled})
	if result.Status != StatusCancelled {
		t.Errorf("expected cancelled, got %s", result.Status)
	}
	if !result.Changed {
		t.Error("expected changed=true")
	}
}

func TestRollupReservationStatus_AllTerminalWithCheckout(t *testing.T) {
	result := RollupReservationStatus(StatusCheckedIn, []ItemStatus{ItemStatusCheckedOut, ItemStatusCancelled})
	if result.Status != StatusCheckedOut {
		t.Errorf("expected checked_out, got %s", result.Status)
	}
	if !result.Changed {
		t.Error("expected changed=true")
	}
}

func TestRollupReservationStatus_AllTerminalWithoutCheckout(t *testing.T) {
	// All terminal but no checked_out → unchanged
	result := RollupReservationStatus(StatusConfirmed, []ItemStatus{ItemStatusNoShow, ItemStatusCancelled})
	if result.Changed {
		t.Error("expected unchanged for all-terminal-without-checkout")
	}
	if result.Status != StatusConfirmed {
		t.Errorf("expected unchanged (confirmed), got %s", result.Status)
	}
}

func TestRollupReservationStatus_AllCheckedIn(t *testing.T) {
	result := RollupReservationStatus(StatusConfirmed, []ItemStatus{ItemStatusCheckedIn, ItemStatusCheckedIn})
	if result.Status != StatusCheckedIn {
		t.Errorf("expected checked_in, got %s", result.Status)
	}
	if !result.Changed {
		t.Error("expected changed=true")
	}
}

func TestRollupReservationStatus_PartialCheckin(t *testing.T) {
	// One checked_in, one booked → unchanged
	result := RollupReservationStatus(StatusConfirmed, []ItemStatus{ItemStatusCheckedIn, ItemStatusBooked})
	if result.Changed {
		t.Error("expected unchanged for partial check-in")
	}
	if result.Status != StatusConfirmed {
		t.Errorf("expected unchanged (confirmed), got %s", result.Status)
	}
}

func TestRollupReservationStatus_AllOverstay(t *testing.T) {
	// All overstay → unchanged (rule 4, no checked_out)
	result := RollupReservationStatus(StatusCheckedIn, []ItemStatus{ItemStatusOverstay, ItemStatusOverstay})
	if result.Changed {
		t.Error("expected unchanged for all-overstay")
	}
	if result.Status != StatusCheckedIn {
		t.Errorf("expected unchanged (checked_in), got %s", result.Status)
	}
}

func TestRollupReservationStatus_MixedTerminalNonTerminal(t *testing.T) {
	// checked_out + checked_in → checked_in (rule 3: >=1 checked_in AND no booked)
	result := RollupReservationStatus(StatusConfirmed, []ItemStatus{ItemStatusCheckedOut, ItemStatusCheckedIn})
	if result.Status != StatusCheckedIn {
		t.Errorf("expected checked_in, got %s", result.Status)
	}
	if !result.Changed {
		t.Error("expected changed=true for mixed terminal/non-terminal")
	}
}

func TestRollupReservationStatus_NoChange(t *testing.T) {
	result := RollupReservationStatus(StatusConfirmed, []ItemStatus{ItemStatusBooked, ItemStatusBooked})
	if result.Changed {
		t.Error("expected unchanged when status matches")
	}
	if result.Status != StatusConfirmed {
		t.Errorf("expected confirmed, got %s", result.Status)
	}
}

// --- ActionIdempotency tests ---

// R-RES-EDGE-061: Action endpoint called with target already in destination state.
// Constructive (confirm) → 200 no-op. Per §7.4.
func TestActionIdempotency_Confirm(t *testing.T) {
	tests := []struct {
		name         string
		desired      any
		current      any
		wantNoOp     bool
		wantConflict bool
	}{
		{"already confirmed", StatusConfirmed, StatusConfirmed, true, false},
		{"from hold", StatusConfirmed, StatusHold, false, false},
		{"from cancelled", StatusConfirmed, StatusCancelled, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noOp, conflict := ActionIdempotency("confirm", tt.desired, tt.current)
			if noOp != tt.wantNoOp {
				t.Errorf("noOp = %v, want %v", noOp, tt.wantNoOp)
			}
			if conflict != tt.wantConflict {
				t.Errorf("conflict = %v, want %v", conflict, tt.wantConflict)
			}
		})
	}
}

// R-RES-EDGE-040: Cancel on terminal → 409.
// R-RES-EDGE-061: Destructive action on already-applied state → 409.
func TestActionIdempotency_Cancel(t *testing.T) {
	tests := []struct {
		name         string
		desired      any
		current      any
		wantNoOp     bool
		wantConflict bool
	}{
		{"already cancelled", StatusCancelled, StatusCancelled, false, true}, // §7.4: cancel is destructive → 409
		{"from hold", StatusCancelled, StatusHold, false, false},
		{"from confirmed", StatusCancelled, StatusConfirmed, false, false},
		{"from archived", StatusCancelled, StatusArchived, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noOp, conflict := ActionIdempotency("cancel", tt.desired, tt.current)
			if noOp != tt.wantNoOp {
				t.Errorf("noOp = %v, want %v", noOp, tt.wantNoOp)
			}
			if conflict != tt.wantConflict {
				t.Errorf("conflict = %v, want %v", conflict, tt.wantConflict)
			}
		})
	}
}

func TestActionIdempotency_CheckinItem(t *testing.T) {
	tests := []struct {
		name         string
		desired      any
		current      any
		wantNoOp     bool
		wantConflict bool
	}{
		{"already checked in", ItemStatusCheckedIn, ItemStatusCheckedIn, true, false},
		{"from booked", ItemStatusCheckedIn, ItemStatusBooked, false, false},
		{"checked out terminal", ItemStatusCheckedIn, ItemStatusCheckedOut, false, true},
		{"cancelled terminal", ItemStatusCheckedIn, ItemStatusCancelled, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noOp, conflict := ActionIdempotency("checkin_item", tt.desired, tt.current)
			if noOp != tt.wantNoOp {
				t.Errorf("noOp = %v, want %v", noOp, tt.wantNoOp)
			}
			if conflict != tt.wantConflict {
				t.Errorf("conflict = %v, want %v", conflict, tt.wantConflict)
			}
		})
	}
}

func TestActionIdempotency_CheckoutItem(t *testing.T) {
	tests := []struct {
		name         string
		desired      any
		current      any
		wantNoOp     bool
		wantConflict bool
	}{
		{"already checked out", ItemStatusCheckedOut, ItemStatusCheckedOut, true, false},
		{"from checked in", ItemStatusCheckedOut, ItemStatusCheckedIn, false, false},
		{"cancelled terminal", ItemStatusCheckedOut, ItemStatusCancelled, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noOp, conflict := ActionIdempotency("checkout_item", tt.desired, tt.current)
			if noOp != tt.wantNoOp {
				t.Errorf("noOp = %v, want %v", noOp, tt.wantNoOp)
			}
			if conflict != tt.wantConflict {
				t.Errorf("conflict = %v, want %v", conflict, tt.wantConflict)
			}
		})
	}
}

// R-RES-EDGE-041: Reactivation on past reservation rejection model.
// R-RES-EDGE-061: Reactivate only valid from cancelled — 409 otherwise.
func TestActionIdempotency_Reactivate(t *testing.T) {
	tests := []struct {
		name         string
		desired      any
		current      any
		wantNoOp     bool
		wantConflict bool
	}{
		{"already confirmed", StatusConfirmed, StatusConfirmed, true, false},
		{"from cancelled", StatusConfirmed, StatusCancelled, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noOp, conflict := ActionIdempotency("reactivate", tt.desired, tt.current)
			if noOp != tt.wantNoOp {
				t.Errorf("noOp = %v, want %v", noOp, tt.wantNoOp)
			}
			if conflict != tt.wantConflict {
				t.Errorf("conflict = %v, want %v", conflict, tt.wantConflict)
			}
		})
	}
}

func TestActionIdempotency_Unknown(t *testing.T) {
	noOp, conflict := ActionIdempotency("unknown_action", "desired", "current")
	if noOp || conflict {
		t.Errorf("expected noOp=false, conflict=false for unknown action; got noOp=%v, conflict=%v", noOp, conflict)
	}
}

// --- IsTerminal tests ---

func TestIsTerminalItemStatus(t *testing.T) {
	tests := []struct {
		status ItemStatus
		want   bool
	}{
		{ItemStatusCheckedOut, true},
		{ItemStatusCancelled, true},
		{ItemStatusArchived, true},
		{ItemStatusBooked, false},
		{ItemStatusCheckedIn, false},
		{ItemStatusNoShow, false},
		{ItemStatusOverstay, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := IsTerminalItemStatus(tt.status); got != tt.want {
				t.Errorf("IsTerminalItemStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsTerminalReservationStatus(t *testing.T) {
	tests := []struct {
		status ReservationStatus
		want   bool
	}{
		{StatusCancelled, true},
		{StatusArchived, true},
		{StatusHold, false},
		{StatusConfirmed, false},
		{StatusCheckedIn, false},
		{StatusCheckedOut, false},
		{StatusPendingCancellation, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := IsTerminalReservationStatus(tt.status); got != tt.want {
				t.Errorf("IsTerminalReservationStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
