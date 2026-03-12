package service

import (
	"context"
	"time"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Planner interface {
	GetPlannerData(ctx context.Context, startDate, endDate time.Time) (*PlannerData, error)
}

type plannerService struct {
	*Svc
}

func NewPlannerService(r repo.Queries, db *pgxpool.Pool) Planner {
	return &plannerService{
		&Svc{
			repo: &r,
			db:   db,
		},
	}
}

// PlannerReservation represents a single reservation item displayed on the planner grid.
// Each item maps to a specific room for a specific stay period.
type PlannerReservation struct {
	ReservationItemID uuid.UUID `json:"reservation_item_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReservationID     uuid.UUID `json:"reservation_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	ReservationCode   string    `json:"reservation_code" example:"RES-12345"`
	AssignedRoomID    uuid.UUID `json:"assigned_room_id" example:"550e8400-e29b-41d4-a716-446655440000"`    // The assigned room (may be empty if unassigned)
	BookedRoomTypeID  uuid.UUID `json:"booked_room_type_id" example:"660e8400-e29b-41d4-a716-446655440000"` // The originally booked room type
	GuestID           uuid.UUID `json:"guest_id" example:"770e8400-e29b-41d4-a716-446655440000"`
	GuestName         string    `json:"guest_name" example:"John Doe"`
	CheckInDate       string    `json:"check_in_date" example:"2026-02-07"`
	CheckOutDate      string    `json:"check_out_date" example:"2026-02-14"`
	ItemStatus        string    `json:"item_status" example:"booked"`           // booked, checked_in, checked_out, etc.
	ReservationStatus string    `json:"reservation_status" example:"confirmed"` // hold, confirmed, checked_in, etc.
	StatusColor       string    `json:"status_color" example:"#FF5733"`         // Color code for UI representation
	TotalOccupancy    int       `json:"total_occupancy" example:"2"`
	StayPricePence    int       `json:"stay_price_pence" example:"10000"`
}

// PlannerRoom represents a single room row in the planner grid.
type PlannerRoom struct {
	RoomID       uuid.UUID            `json:"room_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RoomName     string               `json:"room_name" example:"101"`
	RoomTypeID   uuid.UUID            `json:"room_type_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	RoomTypeCode string               `json:"room_type_code" example:"DELUXE"`
	Reservations []PlannerReservation `json:"reservations"`	
}

// PlannerData is the complete response for the planner view.
type PlannerData struct {
	StartDate string        `json:"start_date" example:"2026-02-07"`
	EndDate   string        `json:"end_date" example:"2026-02-14"`
	Rooms     []PlannerRoom `json:"rooms"`
}

func (s *plannerService) GetPlannerData(ctx context.Context, startDate, endDate time.Time) (*PlannerData, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)

	// Set property ID from context
	propertyID := hf.GetPropertyIDFromCtx(ctx).String()
	
	qtx.SetCurrentPropertyID(ctx, propertyID)

	// Check cache here

	// TODO: Check user permissions here
	// If not permitted, return hf.ErrNotPermitted

	// Fetch rooms and reservation items in parallel
	dbRooms, err := qtx.GetRoomsForPlanner(ctx)
	if err != nil {
		return nil, err
	}

	dbReservations, err := qtx.GetReservationItemsForPlanner(ctx, repo.GetReservationItemsForPlannerParams{
		StartDate: pgtype.Timestamptz{Time: startDate, Valid: true},
		EndDate:   pgtype.Timestamptz{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Map reservation items by room ID
	roomReservationsMap := make(map[uuid.UUID][]PlannerReservation)
	for _, ri := range dbReservations {
		// Build guest name
		guestName := ""
		if ri.GuestFirstName.Valid || ri.GuestLastName.Valid {
			guestName = hf.StringOrEmpty(ri.GuestFirstName) + " " + hf.StringOrEmpty(ri.GuestLastName)
		}

		// Extract dates from stay_period range
		checkIn := ""
		checkOut := ""
		if ri.StayPeriod.Valid {
			if ri.StayPeriod.Lower.Valid {
				checkIn = ri.StayPeriod.Lower.Time.Format("2006-01-02")
			}
			if ri.StayPeriod.Upper.Valid {
				checkOut = ri.StayPeriod.Upper.Time.Format("2006-01-02")
			}
		}

		guestID := uuid.Nil
		if ri.GuestID.Valid {
			guestID = ri.GuestID.UUID
		}

		assignedRoomID := uuid.Nil
		if ri.AssignedRoomID.Valid {
			assignedRoomID = ri.AssignedRoomID.UUID
		}

		pr := PlannerReservation{
			ReservationItemID: ri.ReservationItemID,
			ReservationID:     ri.ReservationID,
			ReservationCode:   hf.StringOrEmpty(ri.ReservationCode),
			AssignedRoomID:    assignedRoomID,
			BookedRoomTypeID:  ri.BookedRoomTypeID,
			GuestID:           guestID,
			GuestName:         guestName,
			CheckInDate:       checkIn,
			CheckOutDate:      checkOut,
			ItemStatus:        string(ri.ItemStatus),
			ReservationStatus: string(ri.ReservationStatus),
			StayPricePence:    int(ri.StayPricePence),
			StatusColor:       getStatusColor(string(ri.ItemStatus), ri.ReservationID),
		}

		// Group by assigned room ID if assigned, otherwise we'll handle unassigned separately
		if ri.AssignedRoomID.Valid {
			roomID := ri.AssignedRoomID.UUID
			roomReservationsMap[roomID] = append(roomReservationsMap[roomID], pr)
		}
	}

	// Build room list with their reservations
	rooms := make([]PlannerRoom, 0, len(dbRooms))
	for _, r := range dbRooms {
		roomID := r.RoomID
		rooms = append(rooms, PlannerRoom{
			RoomID:       roomID,
			RoomName:     r.RoomName,
			RoomTypeID:   r.RoomTypeID,
			RoomTypeCode: r.RoomTypeCode,
			Reservations: roomReservationsMap[roomID],
		})
	}

	return &PlannerData{
		Rooms:     rooms,
		StartDate: startDate.Format("2006-01-02"),
		EndDate:   endDate.Format("2006-01-02"),
	}, nil
}

func getStatusColor(itemStatus string, resId uuid.UUID) string {
	// Define color codes based on status combinations
	statusColorMap := map[string]string{
		"booked":      getBookedColor(resId),
		"checked_in":  hf.StatusColorCheckedIn,
		"checked_out": hf.StatusColorCheckedOut,
		"no_show":     hf.StatusColorNoShow,
	}

	key := itemStatus

	if color, exists := statusColorMap[key]; exists {
		return color
	}
	return hf.StatusColorDefault
}

func getBookedColor(resId uuid.UUID) string {
	// Generate a consistent color based on reservation ID
	colors := []string{
		hf.StatusColorBooked,
		"#B8C5D6",
		"#a7aec6",
		"#9b96b3", // Magenta
		"#917e9d", // Purple
	}
	index := resId[15] % uint8(len(colors)) // Use last byte of UUID for indexing
	return colors[index]
}
