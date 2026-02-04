//go:build ignore

package db_tests

import (
	"context"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
)

func TestDbHousekeepingLogs(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	room := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)
	user := GenerateTestUser(t, ctx)

	// Additional rooms for FK tests
	room2 := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)
	room3 := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)

	insertQuery := `INSERT INTO inventory.housekeeping_logs
		(property_id, user_id, room_id, status_from, status_to, notes)
		VALUES ($1, $2, $3, $4, $5, $6)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-HSKL-02 - Property must exist",
				FakeIDIdx: 0,
				RealID:    property.ID,
				BaseParams: []interface{}{
					property.ID,
					nil,
					room.ID,
					"dirty",
					"clean",
					nil,
				},
			},
			{
				Name:      "TC-HSKL-03 - User must exist if set",
				FakeIDIdx: 1,
				RealID:    user.ID,
				BaseParams: []interface{}{
					property.ID,
					user.ID,
					room2.ID,
					"dirty",
					"in_progress",
					nil,
				},
			},
			{
				Name:      "TC-HSKL-04 - Room must exist",
				FakeIDIdx: 2,
				RealID:    room3.ID,
				BaseParams: []interface{}{
					property.ID,
					nil,
					room3.ID,
					"in_progress",
					"clean",
					nil,
				},
			},
		})
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		constraintRoom := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)
		baseParams := []interface{}{
			property.ID,
			nil,
			constraintRoom.ID,
			"dirty",
			"clean",
			nil,
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, baseParams, []hf.ConstraintTest{
			{
				Name:        "TC-HSKL-05a - status_from must be a valid enum",
				Field:       "status_from",
				Value:       "invalid_status",
				ExpectedErr: hf.InvalidTextRepresentationCode,
				FieldIndex:  3,
			},
			{
				Name:        "TC-HSKL-05b - status_to must be a valid enum",
				Field:       "status_to",
				Value:       "invalid_status",
				ExpectedErr: hf.InvalidTextRepresentationCode,
				FieldIndex:  4,
			},
			{
				Name:        "TC-HSKL-05c - status_to and status_from must be different",
				Field:       "status_to",
				Value:       "dirty", // Same as status_from in baseParams
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  4,
			},
		})
	})
}
