//go:build ignore
package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbBookedDailyRates(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	reservation := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)
	reservationItem := GenerateTestReservationItem(t, ctx, reservation.ID, property.ID)
	ratePlan := GenerateTestRatePlan(t, ctx, property.ID)
	user := GenerateTestUser(t, ctx)

	baseParams := TestBookedDailyRate{
		ReservationItemID:          reservationItem.ID,
		CalendarDate:               time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		RatePlanID:                 ratePlan.ID,
		BasePricePence:             10000,
		Adjustment:                 nil,
		AdjustmentApproved:         false,
		AdjustmentApprovedByUserID: nil,
		FinalPricePence:            0,
	}

	paramsSlice := hf.StructToSlice(baseParams)

	insertQuery := `INSERT INTO pricing.booked_daily_rates
	(reservation_item_id, calendar_date, rate_plan_id, base_price_pence, adjustment, adjustment_approved, adjustment_approved_by_user_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		// Additional entities for FK tests
		anotherRatePlan := GenerateTestRatePlan(t, ctx, property.ID)

		paramsSlice = paramsSlice[:7] // Only first 7 fields are FK related

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:       "TC-BDR-02 - Reservation item must exist",
				FakeIDIdx:  0,
				RealID:     reservationItem.ID,
				BaseParams: paramsSlice,
			},
			{
				Name:       "TC-BDR-03 - Rate plan must exist",
				FakeIDIdx:  2,
				RealID:     anotherRatePlan.ID,
				BaseParams: paramsSlice,
			},
			{
				Name:       "TC-BDR-09 - Approving user must exist",
				FakeIDIdx:  6,
				RealID:     user.ID,
				BaseParams: paramsSlice,
			},
		})
	})

	t.Run("TC-BDR-10 - Final price calculation", func(t *testing.T) {
		t.Parallel()

		type adjustmentTestCase struct {
			name           string
			basePricePence int
			adjustment     *string
			expectedFinal  int
		}

		testCases := []adjustmentTestCase{
			{
				name:           "No adjustment - final equals base",
				basePricePence: 10000,
				adjustment:     nil,
				expectedFinal:  10000,
			},
			{
				name:           "Percentage adjustment +10%",
				basePricePence: 10000,
				adjustment:     strPtr(`{"type": "percentage", "value": 10, "reason": "Peak season"}`),
				expectedFinal:  11000,
			},
			{
				name:           "Percentage adjustment -10%",
				basePricePence: 10000,
				adjustment:     strPtr(`{"type": "percentage", "value": -10, "reason": "Discount"}`),
				expectedFinal:  9000,
			},
			{
				name:           "Fixed adjustment +500",
				basePricePence: 10000,
				adjustment:     strPtr(`{"type": "fixed", "value": 500, "reason": "Extra service"}`),
				expectedFinal:  10500,
			},
			{
				name:           "Fixed adjustment -500",
				basePricePence: 10000,
				adjustment:     strPtr(`{"type": "fixed", "value": -500, "reason": "Loyalty discount"}`),
				expectedFinal:  9500,
			},
		}

		for i, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				baseParams.CalendarDate = time.Now().AddDate(0, 0, i).Format("2006-01-02")

				var finalPrice int
				err := testDB.QueryRow(ctx,
					`INSERT INTO pricing.booked_daily_rates
						(reservation_item_id, calendar_date, rate_plan_id, base_price_pence, adjustment)
						VALUES ($1, $2, $3, $4, $5)
						RETURNING final_price_pence`,
					baseParams.ReservationItemID,
					baseParams.CalendarDate,
					baseParams.RatePlanID,
					tc.basePricePence,
					tc.adjustment,
				).Scan(&finalPrice)

				assert.NoError(t, err)
				assert.Equal(t, tc.expectedFinal, finalPrice, "Final price calculation mismatch")
			})
		}
	})

	t.Run("TC-BDR-11 - Final price can not be negative - I.E adjustment cannot reduce below zero", func(t *testing.T) {
		t.Parallel()

		baseParams.CalendarDate = time.Now().AddDate(0, 1, 15).Format("2006-01-02") // 1.5 months from now

		var finalPrice int
		err := testDB.QueryRow(ctx,
			`INSERT INTO pricing.booked_daily_rates
				(reservation_item_id, calendar_date, rate_plan_id, base_price_pence, adjustment)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING final_price_pence`,
			baseParams.ReservationItemID,
			baseParams.CalendarDate,
			baseParams.RatePlanID,
			1000,
			`{"type": "fixed", "value": -1500, "reason": "Excessive discount"}`,
		).Scan(&finalPrice)

		assert.True(t, hf.CheckErrorCode(err, hf.RaiseExceptionCode), "Expected raise exception for negative final price, got: %v", err)
		assert.Equal(t, 0, finalPrice, "Final price should not be negative")
	})

	t.Run("TC-BDR-12 - Each day must have only one row per reservation item", func(t *testing.T) {
		_, err := testDB.Exec(ctx, insertQuery,
			baseParams.ReservationItemID,
			baseParams.CalendarDate,
			baseParams.RatePlanID,
			baseParams.BasePricePence,
			baseParams.Adjustment,
			baseParams.AdjustmentApproved,
			baseParams.AdjustmentApprovedByUserID,
		)
		assert.NoError(t, err)

		// Second insert with same item and date should fail
		_, err = testDB.Exec(ctx, insertQuery,
			baseParams.ReservationItemID,
			baseParams.CalendarDate,
			baseParams.RatePlanID,
			baseParams.BasePricePence,
			baseParams.Adjustment,
			baseParams.AdjustmentApproved,
			baseParams.AdjustmentApprovedByUserID,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode),
			"Expected unique violation for duplicate reservation_item + calendar_date, got: %v", err)
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		// All constraint tests fail, so no row is inserted - same base params can be used
		t.Parallel()
		params := baseParams
		params.CalendarDate = time.Now().AddDate(0, 1, 0).Format("2006-01-02") // One month from now

		paramsSlice = hf.StructToSlice(params)[:7]

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, paramsSlice, []hf.ConstraintTest{
			{
				Name:        "TC-BDR-06 - Base price pence must be non-negative",
				Field:       "base_price_pence",
				Value:       -100,
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  3,
			},
			{
				Name:        "TC-BDR-07a - Adjustment missing type",
				Field:       "adjustment",
				Value:       `{"value": 10, "reason": "test"}`,
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  4,
			},
			{
				Name:        "TC-BDR-07b - Adjustment missing value",
				Field:       "adjustment",
				Value:       `{"type": "percentage", "reason": "test"}`,
				ExpectedErr: hf.NotNullViolationCode,
				FieldIndex:  4,
			},
			{
				Name:        "TC-BDR-07c - Adjustment missing reason",
				Field:       "adjustment",
				Value:       `{"type": "percentage", "value": 10}`,
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  4,
			},
			{
				Name:        "TC-BDR-07d - Adjustment invalid type",
				Field:       "adjustment",
				Value:       `{"type": "invalid", "value": 10, "reason": "test"}`,
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  4,
			},
			{
				Name:        "TC-BDR-07e - Adjustment zero value",
				Field:       "adjustment",
				Value:       `{"type": "percentage", "value": 0, "reason": "test"}`,
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  4,
			},
		})
	})
}

func strPtr(s string) *string {
	return &s
}
