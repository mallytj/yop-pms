package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestDbReservationItem(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)

	roomType := GenerateTestRoomType(t, ctx, property.ID)

	room := GenerateTestRoom(t, ctx, property.ID, roomType.ID)

	guest := GenerateTestGuest(t, ctx, property.ID)

	reservation := GenerateTestReservation(t, ctx, property.ID, guest.ID)

	ratePlan := GenerateTestRatePlan(t, ctx, property.ID)

	checkInDate := time.Now().AddDate(0, 1, 0)   // One month from now
	checkOutDate := checkInDate.AddDate(0, 0, 4) // 4 nights stay

	insertQuery := `
		INSERT INTO operations.reservation_items 
		(property_id, reservation_id, booked_room_type_id, assigned_room_id, rate_plan_id, stay_period,
		 base_rate_pence, adults_count, children_count, status) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	baseParams := TestReservationItem{
		PropertyID:       property.ID,
		ReservationID:    reservation.ID,
		BookedRoomTypeID: roomType.ID,
		AssignedRoomID:   nil, // Not assigned yet
		RatePlanID:       ratePlan.ID,
		StayPeriod:       *hf.ToPgTstzRange(checkInDate, checkOutDate),
		BaseRatePence:    10000,
		AdultsCount:      2,
		ChildrenCount:    0,
		Status:           "booked",
	}

	t.Run("FK Existence Tests", func(t *testing.T) {
		t.Parallel()
		roomTestParams := baseParams
		roomTestParams.AssignedRoomID = &room.ID
		paramsSlice := hf.StructToSlice(baseParams)

		tests := []hf.FKExistenceTest{
			{
				Name:       "TC-RESI-02 - Reservation must exist",
				FakeIDIdx:  1,
				RealID:     reservation.ID,
				BaseParams: paramsSlice,
			},
			{
				Name:       "TC-RESI-03 - Booked Room Type must exist",
				FakeIDIdx:  2,
				RealID:     roomType.ID,
				BaseParams: paramsSlice,
			},
			{
				Name:       "TC-RESI-04 - Assigned Room must exist if set",
				FakeIDIdx:  3,
				RealID:     room.ID,
				BaseParams: hf.StructToSlice(roomTestParams),
			},
			{
				Name:       "TC-RESI-05 - Rate Plan must exist",
				FakeIDIdx:  4,
				RealID:     ratePlan.ID,
				BaseParams: paramsSlice,
			},
			{
				Name:       "TC-RESI-33 - Property must exist",
				FakeIDIdx:  0,
				RealID:     property.ID,
				BaseParams: paramsSlice,
			},
		}
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, tests)
	})

	t.Run("Constraints", func(t *testing.T) {
		t.Parallel()
		paramsSlice := hf.StructToSlice(baseParams)

		tests := []hf.ConstraintTest{
			{
				Name:        "TC-RESI-06 - Stay period is required",
				Field:       "stay_period",
				Value:       nil,
				FieldIndex:  5,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-RESI-07 - Stay period must be in chronological order",
				Field:       "stay_period",
				Value:       hf.ToPgTstzRange(checkOutDate, checkInDate),
				FieldIndex:  5,
				ExpectedErr: hf.DataExceptionCode,
			},
			{
				Name:        "TC-RESI-08 - Stay period must have both upper and lower bounds",
				Field:       "stay_period",
				Value:       pgtype.Range[pgtype.Timestamptz]{Lower: pgtype.Timestamptz{Time: checkInDate, Valid: true}, Upper: pgtype.Timestamptz{Valid: false}, LowerType: pgtype.Inclusive, UpperType: pgtype.Unbounded},
				FieldIndex:  5,
				ExpectedErr: hf.NotNullViolationCode, // Because of the CHECK constraint on both bounds being NOT NULL
			},
			// Backlog for MVP
			// Need to update rate plan to have min stay requirement first
			// Not necessary for initial launch
			// TODO fix backlog item
			// {
			// 	Name:        "TC-RESI-09 - Stay period must satisfy minimum stay requirements",
			// 	Field:       "stay_period",
			// 	Value:       hf.ToPgTstzRange(checkInDate, checkInDate.AddDate(0, 0, 1)), // 1 night stay, assuming min stay is 2 nights
			// 	FieldIndex:  5,
			// 	ExpectedErr: hf.CheckViolationCode,
			// },
			{
				Name:        "TC-RESI-10 - Stay period must not be in the past",
				Field:       "stay_period",
				Value:       hf.ToPgTstzRange(time.Now().AddDate(0, 0, -5), time.Now().AddDate(0, 0, -1)), // 5 days ago to 1 day ago
				FieldIndex:  5,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RESI-12 - The base rate must be non-negative",
				Field:       "base_rate_pence",
				Value:       -5000,
				FieldIndex:  6,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RESI-14 - Adults count must be at least 1",
				Field:       "adults_count",
				Value:       0,
				FieldIndex:  7,
				ExpectedErr: hf.RaiseExceptionCode, // Due to the DB trigger implementation
			},
			{
				Name:        "TC-RESI-15 - Children count must be non-negative",
				Field:       "children_count",
				Value:       -2,
				FieldIndex:  8,
				ExpectedErr: hf.RaiseExceptionCode, // Due to the DB trigger implementation
			},
			{
				Name:        "TC-RESI-18 - Total guests must not exceed max room type capacity",
				Field:       "adults_count",
				Value:       5, // Assuming room type capacity is 4
				FieldIndex:  7,
				ExpectedErr: hf.RaiseExceptionCode, // Due to the DB trigger implementation
			},
			{
				Name:        "TC-RESI-20 - Total guests must exceed min room type capacity",
				Field:       "adults_count",
				Value:       0, // Assuming room type capacity is 1
				FieldIndex:  7,
				ExpectedErr: hf.RaiseExceptionCode, // Due to the DB trigger implementation
			},
		}
		hf.RunConstraintTests(t, ctx, testDB, insertQuery, paramsSlice, tests)
	})

	t.Run("TC-RESI-20 - There must only ever be one reservation item in assigned to a room at any instance of time", func(t *testing.T) {
		t.Parallel()
		// First, create and assign a reservation item to a room for a specific stay period

		params := baseParams
		params.AssignedRoomID = &room.ID
		for i := range 2 {
			_, err := testDB.Exec(ctx, insertQuery,
				params.PropertyID,
				params.ReservationID,
				params.BookedRoomTypeID,
				params.AssignedRoomID,
				params.RatePlanID,
				params.StayPeriod,
				params.BaseRatePence,
				params.AdultsCount,
				params.ChildrenCount,
				params.Status,
			)
			if i == 0 {
				assert.NoError(t, err, "Failed to create initial reservation item: %v", err)
			}
			if i == 1 {
				// Second insertion should fail due to gist overlap exclusion constraint
				assert.True(t, hf.CheckErrorCode(err, hf.ExclusionViolationCode), "Expected exclusion violation due to room occupancy constraint, got: %v", err)
			}
		}
	})

	t.Run("TC-RESI-27 - Reservation must be in the same property as the reservation item", func(t *testing.T) {
		t.Parallel()
		// Create another property
		anotherProperty := GenerateTestProperty(t, ctx)

		// Create a reservation in the other property
		anotherReservation := GenerateTestReservation(t, ctx, anotherProperty.ID, guest.ID)

		// Create a room type in the other property
		anotherRoomType := GenerateTestRoomType(t, ctx, anotherProperty.ID)

		// Create a room in the other property
		anotherRoom := GenerateTestRoom(t, ctx, anotherProperty.ID, anotherRoomType.ID)

		// Create a rate plan in the other property
		anotherRatePlan := GenerateTestRatePlan(t, ctx, anotherProperty.ID)
		type PropertyConsistencyTest struct {
			name   string
			modify func(p *TestReservationItem)
		}

		tests := []PropertyConsistencyTest{
			{
				name: "Reservation from another property",
				modify: func(p *TestReservationItem) {
					p.ReservationID = anotherReservation.ID
				},
			},
			{
				name: "Booked Room Type from another property",
				modify: func(p *TestReservationItem) {
					p.BookedRoomTypeID = anotherRoomType.ID
				},
			},
			{
				name: "Assigned Room from another property",
				modify: func(p *TestReservationItem) {
					p.AssignedRoomID = &anotherRoom.ID
				},
			},
			{
				name: "Rate Plan from another property",
				modify: func(p *TestReservationItem) {
					p.RatePlanID = anotherRatePlan.ID
				},
			},
		}

		for _, tt := range tests {
			t.Run("Fail when "+tt.name, func(t *testing.T) {
				t.Parallel()
				tcParams := baseParams
				tt.modify(&tcParams)
				_, err := testDB.Exec(ctx, insertQuery,
					tcParams.PropertyID,
					tcParams.ReservationID,
					tcParams.BookedRoomTypeID,
					tcParams.AssignedRoomID,
					tcParams.RatePlanID,
					tcParams.StayPeriod,
					tcParams.BaseRatePence,
					tcParams.AdultsCount,
					tcParams.ChildrenCount,
					tcParams.Status,
				)

				// Check for foreign key violation error
				assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected raise exception due to property mismatch, got: %v", err)
			})

		}

	})
}
