package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexxcode1/ollerod-pms/backend/internal/models"
)

// BookingRepository implements the repository pattern for bookings
type BookingRepository struct {
	db *pgxpool.Pool
}

// NewBookingRepository creates a new booking repository
func NewBookingRepository(db *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{db: db}
}

// Create inserts a new booking
func (r *BookingRepository) Create(ctx context.Context, booking *models.Booking) error {
	query := `
		INSERT INTO bookings (id, guest_id, room_id, check_in, check_out, status, total_price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		booking.ID,
		booking.GuestID,
		booking.RoomID,
		booking.CheckIn,
		booking.CheckOut,
		booking.Status,
		booking.TotalPrice,
		booking.CreatedAt,
		booking.UpdatedAt,
	)
	return err
}

// GetByID retrieves a booking by ID
func (r *BookingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Booking, error) {
	query := `
		SELECT id, guest_id, room_id, check_in, check_out, status, total_price, created_at, updated_at
		FROM bookings
		WHERE id = $1
	`
	var booking models.Booking
	err := r.db.QueryRow(ctx, query, id).Scan(
		&booking.ID,
		&booking.GuestID,
		&booking.RoomID,
		&booking.CheckIn,
		&booking.CheckOut,
		&booking.Status,
		&booking.TotalPrice,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("booking not found")
		}
		return nil, err
	}
	return &booking, nil
}

// Update updates a booking
func (r *BookingRepository) Update(ctx context.Context, booking *models.Booking) error {
	query := `
		UPDATE bookings
		SET guest_id = $2, room_id = $3, check_in = $4, check_out = $5, status = $6, total_price = $7, updated_at = $8
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		booking.ID,
		booking.GuestID,
		booking.RoomID,
		booking.CheckIn,
		booking.CheckOut,
		booking.Status,
		booking.TotalPrice,
		booking.UpdatedAt,
	)
	return err
}

// Delete deletes a booking
func (r *BookingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM bookings WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// FindOverlapping finds bookings that overlap with the given date range
func (r *BookingRepository) FindOverlapping(ctx context.Context, roomID uuid.UUID, checkIn, checkOut time.Time) ([]*models.Booking, error) {
	query := `
		SELECT id, guest_id, room_id, check_in, check_out, status, total_price, created_at, updated_at
		FROM bookings
		WHERE room_id = $1
		  AND status NOT IN ('cancelled', 'completed')
		  AND NOT (check_out <= $2 OR check_in >= $3)
	`
	rows, err := r.db.Query(ctx, query, roomID, checkIn, checkOut)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		var booking models.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.GuestID,
			&booking.RoomID,
			&booking.CheckIn,
			&booking.CheckOut,
			&booking.Status,
			&booking.TotalPrice,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, &booking)
	}

	return bookings, rows.Err()
}

// RoomRepository implements the repository pattern for rooms
type RoomRepository struct {
	db *pgxpool.Pool
}

// NewRoomRepository creates a new room repository
func NewRoomRepository(db *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{db: db}
}

// GetByID retrieves a room by ID
func (r *RoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error) {
	query := `
		SELECT id, room_number, room_type, price_per_night, status, created_at, updated_at
		FROM rooms
		WHERE id = $1
	`
	var room models.Room
	err := r.db.QueryRow(ctx, query, id).Scan(
		&room.ID,
		&room.RoomNumber,
		&room.RoomType,
		&room.PricePerNight,
		&room.Status,
		&room.CreatedAt,
		&room.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("room not found")
		}
		return nil, err
	}
	return &room, nil
}

// UpdateStatus updates the status of a room
func (r *RoomRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE rooms
		SET status = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status, time.Now())
	return err
}
