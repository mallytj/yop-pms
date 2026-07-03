package booking

// See `docs/flows/reservations.md` for detailed flow diagrams and explanations of the reservation lifecycle, including status transitions,
// See `docs/requirements/reservations.md` for detailed requirements mapping, including which requirements are covered by which handlers and functions.
//
// Core Requirements:
//   - R-RES-CRUD-001: Create reservation with primary guest, >=1 items
//   - R-RES-CRUD-007: Reservation code RES-XXXXXX per property
//   - R-RES-CRUD-013: Lifecycle status per source rules
//   - ADR-015: State machine rollup rule
//   - ADR-018: Stay period time semantics (property check-in/out times)
//   - ADR-020: stay_period_envelope on reservations

import (
	"time"

	"github.com/google/uuid"
	"github.com/lexxcode1/yop-pms/internal/platform/types"
)

// ReservationSource is the origin of a reservation.
type ReservationSource string

const (
	// SourceWebsite Source for booking engine bookings (website, mobile app, etc.)
	// NOTE: The booking engine is deferred to a future PR - this is for future compat
	SourceWebsite ReservationSource = "website"
	// SourceInternal Source for internal bookings (phone, in-person, etc.)
	SourceInternal ReservationSource = "internal"
	// SourceOTA Source for online travel agencies
	SourceOTA ReservationSource = "ota"
)

// ReservationStatus represents the lifecycle of a reservation as a whole.
// Per ADR-015, some statuses are derived from item statuses via rollup,
// while others are set directly by business actions.
type ReservationStatus string

const (
	// StatusHold is for reservations that are on hold (not yet confirmed)
	StatusHold ReservationStatus = "hold"
	// StatusConfirmed is for confirmated reservations
	StatusConfirmed ReservationStatus = "confirmed"
	// StatusCheckedIn is for checked in bookings
	StatusCheckedIn ReservationStatus = "checked_in"
	// StatusCheckedOut is for checked out bookings
	StatusCheckedOut ReservationStatus = "checked_out"
	// StatusPendingCancellation is for reservations that are pending cancellation
	// (e.g. awaiting payment of cancellation fee, or within free cancellation window)
	// NOTE: Full implementation of cancellation flow, including pending cancellation status,
	// is deferred to a future PR - this is for future compat
	StatusPendingCancellation ReservationStatus = "pending_cancellation"
	// StatusCancelled is for cancelled reservations
	StatusCancelled ReservationStatus = "cancelled"
	// StatusArchived is for archived reservations (soft-deleted; not visible
	// in UI) still present in DB
	StatusArchived ReservationStatus = "archived"
)

// ItemStatus represents the lifecycle of a single reservation item (room).
type ItemStatus string

const (
	// ItemStatusBooked is for items that are booked but not yet checked in.
	// Items are always "booked" regardless of whether the reservation is "hold" or "confirmed"
	// — the hold/confirmed distinction is a property of the booking commitment, not the room.
	// This is deliberate: a multi-item reservation (e.g. two rooms for one party at the same
	// time) may have all items in "booked" while the reservation-level status reflects hold
	// (pending payment) or confirmed (settled). ADR-015 rollup provides a safety net for
	// deriving reservation status from item statuses, but the item status itself stays stable.
	ItemStatusBooked ItemStatus = "booked"
	// ItemStatusCheckedIn is for checked in items
	ItemStatusCheckedIn ItemStatus = "checked_in"
	// ItemStatusCheckedOut is for checked out items
	ItemStatusCheckedOut ItemStatus = "checked_out"
	// ItemStatusNoShow is for items that did not check in by the expected
	// arrival date (with grace period) without cancellation
	ItemStatusNoShow ItemStatus = "no_show"
	// ItemStatusOverstay is for items that checked in but checked out
	// before the expected departure date (with grace period)
	ItemStatusOverstay ItemStatus = "overstay"
	// ItemStatusCancelled is for cancelled items.
	// Item-level cancel does not imply reservation-level cancel — ADR-015 rollup
	// only promotes to StatusCancelled when ALL items are cancelled.
	// pending_cancellation is reservation-level only (fee settlement gate),
	// not item-level. See state_machine.go for item cancel transitions.
	ItemStatusCancelled ItemStatus = "cancelled"
	// ItemStatusArchived is for when a reservation has surpassed the property's
	// archival threshold (e.g. 90 days after departure) and is archived (soft-deleted).
	ItemStatusArchived ItemStatus = "archived"
)

// SourceToInitialStatus maps reservation source (+walkin flag) to initial
// reservation and item statuses. See PLAN.md Phase 3 source→status table.
// In short: if the source is from an OTA - they handle the holds etc. therefore confirmed
var SourceToInitialStatus = map[ReservationSource]struct {
	ReservationStatus ReservationStatus
	ItemStatus        ItemStatus
}{
	SourceWebsite:  {StatusHold, ItemStatusBooked},
	SourceInternal: {StatusHold, ItemStatusBooked},
	SourceOTA:      {StatusConfirmed, ItemStatusBooked},
}

// RefundAction describes what happens to payment on cancellation.
// NOTE: Full implementation of cancellation flow, including refund actions, is
// deferred to a future PR - this is for future compat.
type RefundAction string

const (
	// RefundNone means no refund is issued (e.g. for non-refundable bookings)
	RefundNone RefundAction = "none"
	// RefundOriginal means a refund is issued to the original payment method
	RefundOriginal RefundAction = "original"
	// RefundCredit means a refund is issued as a credit on the guest's account
	RefundCredit RefundAction = "credit"
)

// AdjustmentType describes how a rate adjustment is applied.
type AdjustmentType string

const (
	// AdjustmentPercent means the adjustment value is a percentage change to
	// the base rate (e.g. -10 for 10% off, +20 for 20% increase)
	AdjustmentPercent AdjustmentType = "percentage"
	// AdjustmentFixed means the adjustment value is a fixed amount change to
	// the base rate (e.g. -1000 for £10 off, +500 for £5 increase)
	AdjustmentFixed AdjustmentType = "fixed"
)

// --- I/O structs ---

// CreateReservationInput is the request body for `POST /reservations`.
type CreateReservationInput struct {
	Source         ReservationSource   `json:"source" example:"internal"`
	IsWalkin       bool                `json:"is_walkin" example:"false"`
	PropertyID     uuid.UUID           `json:"property_id" example:"00000000-0000-0000-0000-000000000000"`
	PrimaryGuestID *uuid.UUID          `json:"primary_guest_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	Guest          *GuestInlinePayload `json:"guest,omitempty"`
	Notes          string              `json:"notes" example:"Guest requested top floor"` // <=2500 (constraints)
	Items          []CreateItemInput   `json:"items" constraints:"operations.reservation_items"`
	GroupID        *uuid.UUID          `json:"group_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	TravelAgentID  *uuid.UUID          `json:"travel_agent_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
}

// CreateItemInput is a single room-night line within a reservation.
// It is also the request body for adding or updating an item via
// `POST /reservations/{id}/items` and `PATCH /reservations/{id}/items/{item_id}`.
type CreateItemInput struct {
	RoomTypeID     uuid.UUID           `json:"room_type_id" example:"00000000-0000-0000-0000-000000000000"`
	AssignedRoomID *uuid.UUID          `json:"assigned_room_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	ArrivalDate    types.ISO8601Date   `json:"arrival_date" example:"2026-06-01"`
	DepartureDate  types.ISO8601Date   `json:"departure_date" example:"2026-06-05"`
	RatePlanID     *uuid.UUID          `json:"rate_plan_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	AdultsCount    int                 `json:"adults_count" example:"2"`
	ChildrenCount  int                 `json:"children_count" example:"0"`
	Guest          *GuestInlinePayload `json:"guest,omitempty"`
}

// GuestInlinePayload is a simple guest payload for MVP.
// Links to primary_guest_id for existing guests.
// Optional ID field for recurring guests
// Expands (address, nationality, document fields) in guest-profile PR.
type GuestInlinePayload struct {
	ID        *uuid.UUID `json:"id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	FirstName string     `json:"first_name" example:"Jane"`
	LastName  string     `json:"last_name" example:"Doe"`
	Email     string     `json:"email" example:"jane.doe@example.com"`
	Phone     string     `json:"phone" example:"+441234567890"`
}

// CancelInput is the request body for POST /reservations/{id}/cancel.
// NOTE: Full implementation of cancellation flow, including cancellation reasons
// and fees, is deferred to a future PR - this is for future compat.
type CancelInput struct {
	ReasonCode        string       `json:"reason_code" example:"guest_request"`
	FeePence          int32        `json:"fee_pence" example:"5000"`
	WaiveFee          bool         `json:"waive_fee" example:"false"`
	FeeOverrideReason string       `json:"fee_override_reason" example:"Loyalty member"`
	RefundAction      RefundAction `json:"refund_action" example:"original"`
	ShortenStay       bool         `json:"shorten_stay" example:"false"` // force-cancel checked-in items (requires reservations:post_checkin_mutate)
}

// RateAdjustment is a single night's rate adjustment.
// Base rate is fetched separately (GetBookedRates) — no need to include it here.
type RateAdjustment struct {
	CalendarDate time.Time      `json:"calendar_date" example:"2026-06-01"`
	Type         AdjustmentType `json:"type" example:"percentage"`
	Value        int            `json:"value" example:"+10000"`
	Reason       string         `json:"reason" example:"Corp Rate"`
}

// RateAdjustInput is the request body for rate adjustment endpoints.
type RateAdjustInput struct {
	Adjustments []RateAdjustment `json:"adjustments"`
}

// UpdateItemStayPeriodInput is the request body for changing an item's dates.
type UpdateItemStayPeriodInput struct {
	ArrivalDate   types.ISO8601Date `json:"arrival_date" example:"2026-06-01"`
	DepartureDate types.ISO8601Date `json:"departure_date" example:"2026-06-05"`
}

// AssignRoomInput is the request body for assigning a room to an item.
type AssignRoomInput struct {
	RoomID            uuid.UUID `json:"room_id" example:"00000000-0000-0000-0000-000000000000"`
	OverrideDnmReason string    `json:"override_dnm_reason,omitempty" example:"Guest insists on room 101"`
}

// UpdateItemRoomTypeInput is the request body for changing an item's room type.
type UpdateItemRoomTypeInput struct {
	NewRoomTypeID uuid.UUID `json:"new_room_type_id" example:"00000000-0000-0000-0000-000000000000"`
	RetainPrice   bool      `json:"retain_price" example:"false"`
}

// UpdateItemRatePlanInput is the request body for changing an item's rate plan.
type UpdateItemRatePlanInput struct {
	NewRatePlanID uuid.UUID `json:"new_rate_plan_id" example:"00000000-0000-0000-0000-000000000000"`
	RetainPrice   bool      `json:"retain_price" example:"true"`
}

// UpdateMetadataInput is the request body for PATCH /reservations/{id}.
type UpdateMetadataInput struct {
	Notes          *string    `json:"notes,omitempty" example:"Updated contact info"`
	TravelAgentID  *uuid.UUID `json:"travel_agent_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	GroupID        *uuid.UUID `json:"group_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	PrimaryGuestID *uuid.UUID `json:"primary_guest_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
}

// AddItemInput is the request body for POST /reservations/{id}/items.
type AddItemInput struct {
	CreateItemInput
}

// ListParams controls pagination and filtering for GET /reservations.
// Cursor pagination per ADR-014: clients pass cursor_date + cursor_id from the last result.
type ListParams struct {
	PropertyID      uuid.UUID  `json:"property_id" example:"00000000-0000-0000-0000-000000000000"`
	Status          *string    `json:"status,omitempty" example:"confirmed"`
	CursorDate      *time.Time `json:"cursor_date,omitempty" example:"2026-06-01T00:00:00Z"`
	CursorID        *uuid.UUID `json:"cursor_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	StartDate       *time.Time `json:"start_date,omitempty" example:"2026-06-01T00:00:00Z"`
	EndDate         *time.Time `json:"end_date,omitempty" example:"2026-06-05T00:00:00Z"`
	Limit           int32      `json:"limit" example:"50"`
	IncludeArchived bool       `json:"include_archived,omitempty"`
}

// BatchResultItem is one entry in a 207 Multi-Status batch response.
type BatchResultItem struct {
	ItemID string        `json:"item_id" example:"00000000-0000-0000-0000-000000000000"`
	Status string        `json:"status" example:"ok"` // "ok" | "failed"
	Item   *ItemResponse `json:"reservation_item,omitempty"`
	Error  *BatchError   `json:"error,omitempty"`
}

// BatchError is the error payload inside a BatchResultItem.
type BatchError struct {
	Code    string `json:"code" example:"ROOM_UNAVAILABLE"`
	Message string `json:"message" example:"Room not available on selected dates"`
}

// BatchResult is the response for batch checkin/checkout endpoints (207 Multi-Status).
type BatchResult struct {
	Results []BatchResultItem `json:"results"`
}

// RollupResult is the result of the ADR-015 rollup computation.
type RollupResult struct {
	Status  ReservationStatus
	Changed bool
}

// DateAvailability represents availability for a single date.
type DateAvailability struct {
	Date      types.ISO8601Date `json:"date" swaggertype:"string" example:"2026-06-01"`
	Total     int               `json:"total" example:"4"`
	Available int               `json:"available" example:"1"`
	// Reason explains why unavailable (e.g. "no_rate_configured"). Empty when available.
	Reason string `json:"reason,omitempty" example:"no_rate_configured"`
}

// ConflictDate is a specific date causing a booking conflict.
type ConflictDate struct {
	Date time.Time `json:"date"`
}

// SourceIsWalkin checks whether a reservation source+walkin combo indicates a walk-in.
// Walk-in = internal source + is_walkin=true.
func SourceIsWalkin(source ReservationSource, isWalkin bool) bool {
	return source == SourceInternal && isWalkin
}

// ReservationResponse is the API response for a reservation.
// Clean JSON types — no pgtype structs leak into the API contract.
// TODO: auto-generate from sqlc models via `make gen-api-types` to avoid drift.
// This current solution means that if we change anything in the DB etc it won't update automatically
type ReservationResponse struct {
	ID                 uuid.UUID           `json:"id" example:"00000000-0000-0000-0000-000000000000"`
	PropertyID         uuid.UUID           `json:"property_id" example:"00000000-0000-0000-0000-000000000000"`
	PrimaryGuestID     *uuid.UUID          `json:"primary_guest_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	GroupID            *uuid.UUID          `json:"group_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	Source             ReservationSource   `json:"source" example:"internal"`
	TravelAgentID      *uuid.UUID          `json:"travel_agent_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	Notes              string              `json:"notes" example:"Guest requested top floor"`
	Status             ReservationStatus   `json:"status" example:"hold"`
	Version            int32               `json:"version" example:"1"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	DeletedAt          *time.Time          `json:"deleted_at,omitempty"`
	Sequential         int64               `json:"sequential" example:"42"`
	Code               string              `json:"code" example:"RES-111213"`
	StayPeriodEnvelope string              `json:"stay_period_envelope"`
	ExpiresAt          *time.Time          `json:"expires_at,omitempty"`
	CancellationIntent *CancellationIntent `json:"cancellation_intent,omitempty"`
	Guest              *GuestResponse      `json:"guest,omitempty"`
	Items              []ItemResponse      `json:"items,omitempty" constraints:"operations.reservation_items"`
}

// ItemResponse is the API response for a reservation item.
// TODO: auto-generate from sqlc models via `make gen-api-types` to avoid drift.
type ItemResponse struct {
	ID               uuid.UUID  `json:"id" example:"00000000-0000-0000-0000-000000000000"`
	PropertyID       uuid.UUID  `json:"property_id" example:"00000000-0000-0000-0000-000000000000"`
	ReservationID    uuid.UUID  `json:"reservation_id" example:"00000000-0000-0000-0000-000000000000"`
	BookedRoomTypeID uuid.UUID  `json:"booked_room_type_id" example:"00000000-0000-0000-0000-000000000000"`
	AssignedRoomID   *uuid.UUID `json:"assigned_room_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	GuestID          *uuid.UUID `json:"guest_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	RatePlanID       *uuid.UUID `json:"rate_plan_id,omitempty" example:"00000000-0000-0000-0000-000000000000"`
	StayPeriod       string     `json:"stay_period"`
	BaseRatePence    int32      `json:"base_rate_pence" example:"15000"`
	AdultsCount      int32      `json:"adults_count" example:"2"`
	ChildrenCount    int32      `json:"children_count" example:"0"`
	Status           ItemStatus `json:"status" example:"booked"`
	Version          int32      `json:"version" example:"1"`
	DoNotMove        bool       `json:"do_not_move" example:"false"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

// CancellationIntent stores the cancellation metadata recorded at cancel time.
type CancellationIntent struct {
	ReasonCode        string       `json:"reason_code" example:"guest_request"`
	FeePence          int32        `json:"fee_pence" example:"5000"`
	WaiveFee          bool         `json:"waive_fee" example:"false"`
	FeeOverrideReason string       `json:"fee_override_reason,omitempty" example:"Loyalty member"`
	RefundAction      RefundAction `json:"refund_action" example:"original"`
	CancelledByUserID *uuid.UUID   `json:"cancelled_by_user_id,omitempty"`
}

// IncludeFlags controls which related resources are embedded in API responses.
// Parsed from the ?include= query parameter with comma-separated values.
// Default (no param): items included, guest ID only, no folio.
// ?include=none: lightweight envelope without items.
type IncludeFlags struct {
	Items        bool // include items[] array (default true when param absent)
	Guest        bool // expand primary_guest_id into guest object
	FolioSummary bool // include folio balance and part info
	None         bool // explicitly exclude items (forces Items=false)
}

// IncludeItems returns true when items should be included in the response.
func (f IncludeFlags) IncludeItems() bool { return f.Items && !f.None }

// GuestResponse is the expanded guest object returned when ?include=guest.
// TODO: auto-generate from sqlc models via `make gen-api-types` to avoid drift.
type GuestResponse struct {
	ID         uuid.UUID `json:"id" example:"00000000-0000-0000-0000-000000000000"`
	PropertyID uuid.UUID `json:"property_id"`
	FirstName  string    `json:"first_name" example:"Jane"`
	LastName   string    `json:"last_name" example:"Doe"`
	Email      string    `json:"email,omitempty" example:"jane@example.com"`
	Phone      string    `json:"phone,omitempty" example:"+441234567890"`
}

// CancellationQuoteResponse is the stub response for GET /reservations/{id}/cancellation-quote.
// Finance PR replaces this with a real fee calculation.
type CancellationQuoteResponse struct {
	FeePence *int32 `json:"fee_pence"`
	Status   string `json:"status"`
}
