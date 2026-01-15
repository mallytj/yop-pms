package models

import (
	"time"

	"github.com/google/uuid"
)

// Guest represents a hotel guest
type Guest struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Room represents a hotel room
type Room struct {
	ID          uuid.UUID `json:"id"`
	RoomNumber  string    `json:"room_number"`
	RoomType    string    `json:"room_type"`
	PricePerNight float64 `json:"price_per_night"`
	Status      string    `json:"status"` // available, occupied, maintenance
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Booking represents a hotel booking
type Booking struct {
	ID        uuid.UUID `json:"id"`
	GuestID   uuid.UUID `json:"guest_id"`
	RoomID    uuid.UUID `json:"room_id"`
	CheckIn   time.Time `json:"check_in"`
	CheckOut  time.Time `json:"check_out"`
	Status    string    `json:"status"` // pending, confirmed, cancelled, completed
	TotalPrice float64  `json:"total_price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
