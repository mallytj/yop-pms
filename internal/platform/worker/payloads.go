package worker

import "time"

// Event type constants use dot-namespaced format: domain.action.
const (
	EventConfirmationEmail = "smtp.confirmation"
	EventPreArrivalEmail   = "smtp.pre_arrival"
	EventCancellationEmail = "smtp.cancellation"
)

// ConfirmationEmailPayload is the payload for EventConfirmationEmail.
type ConfirmationEmailPayload struct {
	ReservationID string `json:"reservation_id"`
	GuestEmail    string `json:"guest_email"`
	GuestName     string `json:"guest_name"`
	PropertyName  string `json:"property_name"`
}

// PreArrivalEmailPayload is the payload for EventPreArrivalEmail.
type PreArrivalEmailPayload struct {
	ReservationID string    `json:"reservation_id"`
	GuestEmail    string    `json:"guest_email"`
	GuestName     string    `json:"guest_name"`
	PropertyName  string    `json:"property_name"`
	CheckIn       time.Time `json:"check_in"`
}

// CancellationEmailPayload is the payload for EventCancellationEmail.
type CancellationEmailPayload struct {
	ReservationID string `json:"reservation_id"`
	GuestEmail    string `json:"guest_email"`
	GuestName     string `json:"guest_name"`
	PropertyName  string `json:"property_name"`
}
