package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexxcode1/ollerod-pms/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBookingRepository is a mock implementation of BookingRepository
type MockBookingRepository struct {
	mock.Mock
}

func (m *MockBookingRepository) Create(ctx context.Context, booking *models.Booking) error {
	args := m.Called(ctx, booking)
	return args.Error(0)
}

func (m *MockBookingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Booking, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Booking), args.Error(1)
}

func (m *MockBookingRepository) Update(ctx context.Context, booking *models.Booking) error {
	args := m.Called(ctx, booking)
	return args.Error(0)
}

func (m *MockBookingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBookingRepository) FindOverlapping(ctx context.Context, roomID uuid.UUID, checkIn, checkOut time.Time) ([]*models.Booking, error) {
	args := m.Called(ctx, roomID, checkIn, checkOut)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Booking), args.Error(1)
}

// MockRoomRepository is a mock implementation of RoomRepository
type MockRoomRepository struct {
	mock.Mock
}

func (m *MockRoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Room), args.Error(1)
}

func (m *MockRoomRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func TestCreateBooking_Success(t *testing.T) {
	mockBookingRepo := new(MockBookingRepository)
	mockRoomRepo := new(MockRoomRepository)
	service := NewBookingService(mockBookingRepo, mockRoomRepo)

	roomID := uuid.New()
	guestID := uuid.New()
	checkIn := time.Now().Add(24 * time.Hour)
	checkOut := checkIn.Add(48 * time.Hour)

	room := &models.Room{
		ID:            roomID,
		RoomNumber:    "101",
		RoomType:      "deluxe",
		PricePerNight: 150.0,
		Status:        "available",
	}

	booking := &models.Booking{
		GuestID:  guestID,
		RoomID:   roomID,
		CheckIn:  checkIn,
		CheckOut: checkOut,
	}

	mockRoomRepo.On("GetByID", mock.Anything, roomID).Return(room, nil)
	mockBookingRepo.On("FindOverlapping", mock.Anything, roomID, checkIn, checkOut).Return([]*models.Booking{}, nil)
	mockBookingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Booking")).Return(nil)

	err := service.CreateBooking(context.Background(), booking)

	assert.NoError(t, err)
	assert.Equal(t, "pending", booking.Status)
	assert.Equal(t, 300.0, booking.TotalPrice) // 2 nights * $150
	mockBookingRepo.AssertExpectations(t)
	mockRoomRepo.AssertExpectations(t)
}

func TestCreateBooking_InvalidDateRange(t *testing.T) {
	mockBookingRepo := new(MockBookingRepository)
	mockRoomRepo := new(MockRoomRepository)
	service := NewBookingService(mockBookingRepo, mockRoomRepo)

	checkIn := time.Now()
	checkOut := checkIn.Add(-24 * time.Hour) // Check-out before check-in

	booking := &models.Booking{
		GuestID:  uuid.New(),
		RoomID:   uuid.New(),
		CheckIn:  checkIn,
		CheckOut: checkOut,
	}

	err := service.CreateBooking(context.Background(), booking)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidDateRange, err)
}

func TestCreateBooking_RoomNotAvailable(t *testing.T) {
	mockBookingRepo := new(MockBookingRepository)
	mockRoomRepo := new(MockRoomRepository)
	service := NewBookingService(mockBookingRepo, mockRoomRepo)

	roomID := uuid.New()
	checkIn := time.Now().Add(24 * time.Hour)
	checkOut := checkIn.Add(48 * time.Hour)

	room := &models.Room{
		ID:            roomID,
		Status:        "occupied", // Room not available
		PricePerNight: 150.0,
	}

	booking := &models.Booking{
		GuestID:  uuid.New(),
		RoomID:   roomID,
		CheckIn:  checkIn,
		CheckOut: checkOut,
	}

	mockRoomRepo.On("GetByID", mock.Anything, roomID).Return(room, nil)

	err := service.CreateBooking(context.Background(), booking)

	assert.Error(t, err)
	assert.Equal(t, ErrRoomNotAvailable, err)
	mockRoomRepo.AssertExpectations(t)
}

func TestCreateBooking_OverlappingBooking(t *testing.T) {
	mockBookingRepo := new(MockBookingRepository)
	mockRoomRepo := new(MockRoomRepository)
	service := NewBookingService(mockBookingRepo, mockRoomRepo)

	roomID := uuid.New()
	checkIn := time.Now().Add(24 * time.Hour)
	checkOut := checkIn.Add(48 * time.Hour)

	room := &models.Room{
		ID:            roomID,
		Status:        "available",
		PricePerNight: 150.0,
	}

	existingBooking := &models.Booking{
		ID:       uuid.New(),
		RoomID:   roomID,
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Status:   "confirmed",
	}

	booking := &models.Booking{
		GuestID:  uuid.New(),
		RoomID:   roomID,
		CheckIn:  checkIn,
		CheckOut: checkOut,
	}

	mockRoomRepo.On("GetByID", mock.Anything, roomID).Return(room, nil)
	mockBookingRepo.On("FindOverlapping", mock.Anything, roomID, checkIn, checkOut).Return([]*models.Booking{existingBooking}, nil)

	err := service.CreateBooking(context.Background(), booking)

	assert.Error(t, err)
	assert.Equal(t, ErrOverlappingBooking, err)
	mockBookingRepo.AssertExpectations(t)
	mockRoomRepo.AssertExpectations(t)
}

func TestCancelBooking_Success(t *testing.T) {
	mockBookingRepo := new(MockBookingRepository)
	mockRoomRepo := new(MockRoomRepository)
	service := NewBookingService(mockBookingRepo, mockRoomRepo)

	bookingID := uuid.New()
	booking := &models.Booking{
		ID:     bookingID,
		Status: "confirmed",
	}

	mockBookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
	mockBookingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Booking")).Return(nil)

	err := service.CancelBooking(context.Background(), bookingID)

	assert.NoError(t, err)
	assert.Equal(t, "cancelled", booking.Status)
	mockBookingRepo.AssertExpectations(t)
}

func TestConfirmBooking_Success(t *testing.T) {
	mockBookingRepo := new(MockBookingRepository)
	mockRoomRepo := new(MockRoomRepository)
	service := NewBookingService(mockBookingRepo, mockRoomRepo)

	bookingID := uuid.New()
	booking := &models.Booking{
		ID:     bookingID,
		Status: "pending",
	}

	mockBookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
	mockBookingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Booking")).Return(nil)

	err := service.ConfirmBooking(context.Background(), bookingID)

	assert.NoError(t, err)
	assert.Equal(t, "confirmed", booking.Status)
	mockBookingRepo.AssertExpectations(t)
}
