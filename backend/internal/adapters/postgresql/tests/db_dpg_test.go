package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"testing"
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
				Name:       "TC-DPG-04 - Property must exist",
				FakeIDIdx:  0,
				RealID:     property.ID,
				BaseParams: hf.StructToSlice(params),
			},
			{
				Name:       "TC-DPG-02 - Room type must exist",
				FakeIDIdx:  1,
				RealID:     roomType.ID,
				BaseParams: hf.StructToSlice(paramsTwo),
			},
			{
				Name:       "TC-DPG-03 - Rate plan must exist",
				FakeIDIdx:  2,
				RealID:     ratePlan.ID,
				BaseParams: hf.StructToSlice(paramsThree),
			},
		})
	})

	t.Run)()

	// Tests for daily_price_grid table can be added here
}
