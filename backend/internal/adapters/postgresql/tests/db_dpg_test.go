package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbDailyPriceGride(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	roomType := GenerateTestRoomType(t, ctx, property.ID)
	ratePlan := GenerateTestRatePlan(t, ctx, property.ID)

	params := TestDailyPriceGrid{
		PropertyID:        property.ID,
		RoomTypeID:        roomType.ID,
		RatePlanID:        ratePlan.ID,
		CalendarDate:      "2026-12-25",
		BasePricePence:    15000,
		MinLOSRestriction: 2,
		MaxLOSRestriction: 14,
		IsAvailable:       true,
	}

	insertQuery := `INSERT INTO pricing.daily_price_grid
	 (property_id, room_type_id, rate_plan_id, calendar_date, base_price_pence, min_los_restriction, max_los_restriction, is_available) 
	 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		t.Parallel()

		// Alternate dates to avoid unique constraint conflicts during FK tests
		paramsTwo := params
		paramsTwo.CalendarDate = "2026-12-26"
		paramsThree := params
		paramsThree.CalendarDate = "2026-12-27"

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:       "TC-DPGR-04 - Property must exist",
				FakeIDIdx:  0,
				RealID:     property.ID,
				BaseParams: hf.StructToSlice(params),
			},
			{
				Name:       "TC-DPGR-02 - Room type must exist",
				FakeIDIdx:  1,
				RealID:     roomType.ID,
				BaseParams: hf.StructToSlice(paramsTwo),
			},
			{
				Name:       "TC-DPGR-03 - Rate plan must exist",
				FakeIDIdx:  2,
				RealID:     ratePlan.ID,
				BaseParams: hf.StructToSlice(paramsThree),
			},
		})
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-DPGR-05 - Calendar date is required",
				Field:       "calendar_date",
				Value:       nil,
				FieldIndex:  3,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-DPGR-08 - Base price pence is required",
				Field:       "base_price_pence",
				Value:       nil,
				FieldIndex:  4,
				ExpectedErr: hf.NotNullViolationCode,
			},
		}
		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), cases)
	})

	t.Run("TC-DPGR-06 - Calendar date in the future", func(t *testing.T) {
		t.Parallel()

		invalidDate := "2000-01-01" // Past date

		_, err := testDB.Exec(ctx, insertQuery,
			params.PropertyID,
			params.RoomTypeID,
			params.RatePlanID,
			invalidDate,
			params.BasePricePence,
			params.MinLOSRestriction,
			params.MaxLOSRestriction,
			params.IsAvailable,
		)

		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected data exception for invalid calendar date format, got: %v", err)
	})

	t.Run("Min/Max LOS Restrictions", func(t *testing.T) {
		t.Parallel()

		testCases := []hf.ConstraintTest{
			{
				Name:        "TC-DPGR-09 - Min LOS must be greater than 0",
				Field:       "min_los_restriction",
				Value:       0,
				FieldIndex:  5,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-DPGR-10 - Max LOS must be greater than 0",
				Field:       "max_los_restriction",
				Value:       0,
				FieldIndex:  6,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-DPGR-11 - Max LOS must be greater than Min LOS",
				Field:       "max_los_restriction",
				Value:       1,
				FieldIndex:  6,
				ExpectedErr: hf.CheckViolationCode,
			},
		}
		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), testCases)
	})

	t.Run("TC-DPGR-12 - Calendar date must be unique per property, rate plan and room type", func(t *testing.T) {
		t.Parallel()

		// First, insert a valid record
		_, err := testDB.Exec(ctx, insertQuery,
			params.PropertyID,
			params.RoomTypeID,
			params.RatePlanID,
			params.CalendarDate,
			params.BasePricePence,
			params.MinLOSRestriction,
			params.MaxLOSRestriction,
			params.IsAvailable,
		)
		assert.NoError(t, err, "Failed to insert initial daily price grid record: %v", err)

		// Now, attempt to insert a duplicate record
		_, err = testDB.Exec(ctx, insertQuery,
			params.PropertyID,
			params.RoomTypeID,
			params.RatePlanID,
			params.CalendarDate,
			params.BasePricePence,
			params.MinLOSRestriction,
			params.MaxLOSRestriction,
			params.IsAvailable,
		)

		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error for duplicate daily price grid entry, got: %v", err)

		t.Run("works if different room type", func(t *testing.T) {
			t.Parallel()
			otherRoomType := GenerateTestRoomType(t, ctx, property.ID)
			_, err := testDB.Exec(ctx, insertQuery,
				params.PropertyID,
				otherRoomType.ID,
				params.RatePlanID,
				params.CalendarDate,
				params.BasePricePence,
				params.MinLOSRestriction,
				params.MaxLOSRestriction,
				params.IsAvailable,
			)
			assert.NoError(t, err, "Expected no error when inserting daily price grid with different room type, got: %v", err)
		})

		t.Run("works if different rate plan", func(t *testing.T) {
			t.Parallel()
			otherRatePlan := GenerateTestRatePlan(t, ctx, property.ID)
			_, err := testDB.Exec(ctx, insertQuery,
				params.PropertyID,
				params.RoomTypeID,
				otherRatePlan.ID,
				params.CalendarDate,
				params.BasePricePence,
				params.MinLOSRestriction,
				params.MaxLOSRestriction,
				params.IsAvailable,
			)
			assert.NoError(t, err, "Expected no error when inserting daily price grid with different rate plan, got: %v", err)
		})
	})

	t.Run("Property Consistency", func(t *testing.T) {
		t.Parallel()
		// Create another property
		anotherProperty := GenerateTestProperty(t, ctx)

		otherRoomType := GenerateTestRoomType(t, ctx, anotherProperty.ID)
		otherRatePlan := GenerateTestRatePlan(t, ctx, anotherProperty.ID)

		t.Run("TC-DPGR-19 - The room type must belong to the same property", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a daily price grid with a room type from a different property
			_, err := testDB.Exec(ctx, insertQuery,
				anotherProperty.ID,
				otherRoomType.ID,
				params.RatePlanID,
				params.CalendarDate,
				params.BasePricePence,
				params.MinLOSRestriction,
				params.MaxLOSRestriction,
				params.IsAvailable,
			)

			assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error for mismatched property in room type, got: %v", err)
		})

		t.Run("TC-DPGR-20 - The rate plan must belong to the same property", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a daily price grid with a rate plan from a different property
			_, err := testDB.Exec(ctx, insertQuery,
				anotherProperty.ID,
				params.RoomTypeID,
				otherRatePlan.ID,
				params.CalendarDate,
				params.BasePricePence,
				params.MinLOSRestriction,
				params.MaxLOSRestriction,
				params.IsAvailable,
			)

			assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error for mismatched property in rate plan, got: %v", err)
		})

	})

	// Tests for daily_price_grid table can be added here
}
