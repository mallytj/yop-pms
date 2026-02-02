package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbMaintenanceBlocks(t *testing.T) {
	ctx := context.Background()

	// Create a property
	property := GenerateTestProperty(t, ctx)

	// Create a room
	room := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)

	// Create a second room to facilitate FK tests
	anotherRoom := GenerateTestRoom(t, ctx, property.ID, uuid.Nil)

	// Create a user
	user := GenerateTestUser(t, ctx)

	insertQuery := `INSERT INTO inventory.maintenance_blocks (room_id, block_period, reason, type, created_by_user_id) VALUES ($1, $2, $3, $4, $5)`

	params := TestMaintenaceBlock{
		RoomID:          room.ID,
		BlockPeriod:     *hf.ToPgTstzRange(time.Now(), time.Now().Add(24*time.Hour)),
		Reason:          "Routine Maintenance",
		Type:            "repair",
		CreatedByUserID: user.ID,
	}

	alteredParams := params
	alteredParams.RoomID = anotherRoom.ID // For FK tests

	t.Run("FK Existence Tests", func(t *testing.T) {
		t.Parallel()
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:       "TC-MAINT-02 - Room must exist",
				FakeIDIdx:  0,
				RealID:     room.ID,
				BaseParams: hf.StructToSlice(params),
			},
			{
				Name:       "TC-MAINT-03 - Created by user must exist",
				FakeIDIdx:  4,
				RealID:     user.ID,
				BaseParams: hf.StructToSlice(alteredParams),
			},
		})
	})

	t.Run("TC-MAINT-04 - A maintenace blocks block_period must be Start->End not end->start", func(t *testing.T) {
		t.Parallel()

		invalidBlockPeriod := *hf.ToPgTstzRange(time.Now().Add(24*time.Hour), time.Now())

		_, err := testDB.Exec(ctx, insertQuery,
			params.RoomID,
			invalidBlockPeriod,
			params.Reason,
			params.Type,
			params.CreatedByUserID,
		)

		assert.True(t, hf.CheckErrorCode(err, hf.DataExceptionCode), "Expected data exception for invalid block_period, got: %v", err)
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		// All fields are required
		tests := []hf.ConstraintTest{
			{
				Name:        "TC-MAINT-13 - Room ID is required",
				Field:       "room_id",
				Value:       nil,
				FieldIndex:  0,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-MAINT-14 - Block period is required",
				Field:       "block_period",
				Value:       nil,
				FieldIndex:  1,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-MAINT-15 - Reason is required",
				Field:       "reason",
				Value:       nil,
				FieldIndex:  2,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-MAINT-16 - Type is required",
				Field:       "type",
				Value:       nil,
				FieldIndex:  3,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-MAINT-17 - Created by user ID is required",
				Field:       "created_by_user_id",
				Value:       nil,
				FieldIndex:  4,
				ExpectedErr: hf.NotNullViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), tests)
	})

	t.Run("Character Limits", func(t *testing.T) {
		t.Parallel()

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-MAINT-06 - Reason exceeds character limit",
				Field:       "reason",
				Value:       strings.Repeat("A", 151),
				FieldIndex:  2,
				ExpectedErr: hf.CheckViolationCode,
			},
		}
		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), cases)
	})

	t.Run("TC-MAINT-07 - There must not be multiple maintenace blocks on one room at the same time", func(t *testing.T) {
		t.Parallel()

		// First, insert a valid maintenance block
		_, err := testDB.Exec(ctx, insertQuery,
			params.RoomID,
			params.BlockPeriod,
			params.Reason,
			params.Type,
			params.CreatedByUserID,
		)
		assert.NoError(t, err, "Failed to insert initial maintenance block: %v", err)

		// Attempt to insert another maintenance block with overlapping period
		overlappingBlockPeriod := *hf.ToPgTstzRange(time.Now().Add(12*time.Hour), time.Now().Add(36*time.Hour))

		_, err = testDB.Exec(ctx, insertQuery,
			params.RoomID,
			overlappingBlockPeriod,
			"Overlapping Maintenance",
			"repair",
			params.CreatedByUserID,
		)

		assert.True(t, hf.CheckErrorCode(err, hf.ExclusionViolationCode), "Expected exclusion violation error for overlapping maintenance blocks, got: %v", err)
	})
}
