package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lexxcode1/ollerod-pms/backend/internal/models"
)

var (
	ErrBookingNotFound     = errors.New("booking not found")
	ErrRoomNotAvailable    = errors.New("room not available")
	ErrInvalidDateRange    = errors.New("invalid date range")
	ErrOverlappingBooking  = errors.New("overlapping booking exists")
)

// BookingRepository defines the interface for booking data access
type BookingRepository interface {
	Create(ctx context.Context, booking *models.Booking) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Booking, error)
	Update(ctx context.Context, booking *models.Booking) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindOverlapping(ctx context.Context, roomID uuid.UUID, checkIn, checkOut time.Time) ([]*models.Booking, error)
}

// RoomRepository defines the interface for room data access
type RoomRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

// BookingService handles booking business logic
type BookingService struct {
	bookingRepo BookingRepository
	roomRepo    RoomRepository
}

// NewBookingService creates a new booking service
func NewBookingService(bookingRepo BookingRepository, roomRepo RoomRepository) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		roomRepo:    roomRepo,
	}
}

// CreateBooking creates a new booking with validation
func (s *BookingService) CreateBooking(ctx context.Context, booking *models.Booking) error {
	// Validate date range
	if booking.CheckOut.Before(booking.CheckIn) || booking.CheckOut.Equal(booking.CheckIn) {
		return ErrInvalidDateRange
	}

	// Check if room exists and is available
	room, err := s.roomRepo.GetByID(ctx, booking.RoomID)
	if err != nil {
		return err
	}

	if room.Status != "available" {
		return ErrRoomNotAvailable
	}

	// Check for overlapping bookings
	overlapping, err := s.bookingRepo.FindOverlapping(ctx, booking.RoomID, booking.CheckIn, booking.CheckOut)
	if err != nil {
		return err
	}

	if len(overlapping) > 0 {
		return ErrOverlappingBooking
	}

	// Calculate total price
	nights := int(booking.CheckOut.Sub(booking.CheckIn).Hours() / 24)
	booking.TotalPrice = room.PricePerNight * float64(nights)

	// Set default values
	booking.ID = uuid.New()
	booking.Status = "pending"
	booking.CreatedAt = time.Now()
	booking.UpdatedAt = time.Now()

	return s.bookingRepo.Create(ctx, booking)
}

// GetBooking retrieves a booking by ID
func (s *BookingService) GetBooking(ctx context.Context, id uuid.UUID) (*models.Booking, error) {
	return s.bookingRepo.GetByID(ctx, id)
}

// CancelBooking cancels a booking
func (s *BookingService) CancelBooking(ctx context.Context, id uuid.UUID) error {
	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if booking.Status == "cancelled" || booking.Status == "completed" {
		return errors.New("cannot cancel booking with status: " + booking.Status)
	}

	booking.Status = "cancelled"
	booking.UpdatedAt = time.Now()

	return s.bookingRepo.Update(ctx, booking)
}

// ConfirmBooking confirms a booking
func (s *BookingService) ConfirmBooking(ctx context.Context, id uuid.UUID) error {
	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if booking.Status != "pending" {
		return errors.New("can only confirm pending bookings")
	}

	booking.Status = "confirmed"
	booking.UpdatedAt = time.Now()

	return s.bookingRepo.Update(ctx, booking)
}
