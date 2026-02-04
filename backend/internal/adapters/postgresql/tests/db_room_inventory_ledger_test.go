//go:build ignore
package db_tests

import (
	"context"
	"testing"
	"time"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbRoomInventoryLedger(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	room := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)
	reservation := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)
	checkoutSession := GenerateTestCheckoutSession(t, ctx, property.ID, reservation.ID)

	// Additional rooms for FK tests
	room2 := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)

	insertQuery := `INSERT INTO inventory.room_inventory_ledger
		(room_id, reservation_id, checkout_session_id, calendar_date, status)
		VALUES ($1, $2, $3, $4, $5)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		calendarDate1 := time.Now().AddDate(0, 0, 100).Format("2006-01-02")
		calendarDate2 := time.Now().AddDate(0, 0, 101).Format("2006-01-02")

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-RILE-02 - Room must exist",
				FakeIDIdx: 0,
				RealID:    room.ID,
				BaseParams: []interface{}{
					room.ID,
					nil,
					nil,
					calendarDate1,
					"available",
				},
			},
			{
				Name:      "TC-RILE-03 - Checkout session must exist if set",
				FakeIDIdx: 2,
				RealID:    checkoutSession.ID,
				BaseParams: []interface{}{
					room2.ID,
					nil,
					checkoutSession.ID,
					calendarDate2,
					"on_hold",
				},
			},
		})
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		constraintRoom := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)
		baseParams := []interface{}{
			constraintRoom.ID,
			nil,
			nil,
			time.Now().AddDate(0, 0, 102).Format("2006-01-02"),
			"available",
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, baseParams, []hf.ConstraintTest{
			{
				Name:        "TC-RILE-06 - Status must be a valid enum",
				Field:       "status",
				Value:       "invalid_status",
				ExpectedErr: hf.InvalidTextRepresentationCode,
				FieldIndex:  4,
			},
		})
	})

	t.Run("TC-RILE-04 - Calendar date is required", func(t *testing.T) {
		t.Parallel()

		reqRoom := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)

		_, err := testDB.Exec(ctx,
			`INSERT INTO inventory.room_inventory_ledger (room_id, status) VALUES ($1, $2)`,
			reqRoom.ID,
			"available",
		)
		assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode),
			"Expected not null violation for missing calendar_date, got: %v", err)
	})

	t.Run("TC-RILE-07 - Each row must be unique by room & calendar date", func(t *testing.T) {
		t.Parallel()

		uniqueRoom := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)
		calendarDate := time.Now().AddDate(0, 0, 103).Format("2006-01-02")

		// First insert
		_, err := testDB.Exec(ctx, insertQuery,
			uniqueRoom.ID,
			nil,
			nil,
			calendarDate,
			"available",
		)
		assert.NoError(t, err)

		// Second insert with same room and date should fail
		_, err = testDB.Exec(ctx, insertQuery,
			uniqueRoom.ID,
			nil,
			nil,
			calendarDate,
			"decommissioned",
		)
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode),
			"Expected unique violation for duplicate room_id + calendar_date, got: %v", err)
	})

	t.Run("TC-RILE-08 - Sold status requires a reservation", func(t *testing.T) {
		t.Parallel()

		soldRoom := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)
		calendarDate := time.Now().AddDate(0, 0, 104).Format("2006-01-02")

		// Sold without reservation should fail
		_, err := testDB.Exec(ctx, insertQuery,
			soldRoom.ID,
			nil,
			nil,
			calendarDate,
			"sold",
		)
		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode),
			"Expected check violation for sold status without reservation, got: %v", err)

		// Sold with reservation should succeed
		calendarDate2 := time.Now().AddDate(0, 0, 105).Format("2006-01-02")
		_, err = testDB.Exec(ctx, insertQuery,
			soldRoom.ID,
			reservation.ID,
			nil,
			calendarDate2,
			"sold",
		)
		assert.NoError(t, err, "Sold status with reservation should succeed")
	})

	t.Run("TC-RILE-09 - On hold status requires a checkout session", func(t *testing.T) {
		t.Parallel()

		holdRoom := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)
		calendarDate := time.Now().AddDate(0, 0, 106).Format("2006-01-02")

		// On hold without checkout session should fail
		_, err := testDB.Exec(ctx, insertQuery,
			holdRoom.ID,
			nil,
			nil,
			calendarDate,
			"on_hold",
		)
		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode),
			"Expected check violation for on_hold status without checkout_session, got: %v", err)

		// On hold with checkout session should succeed
		calendarDate2 := time.Now().AddDate(0, 0, 107).Format("2006-01-02")
		_, err = testDB.Exec(ctx, insertQuery,
			holdRoom.ID,
			nil,
			checkoutSession.ID,
			calendarDate2,
			"on_hold",
		)
		assert.NoError(t, err, "On hold status with checkout_session should succeed")
	})
}
