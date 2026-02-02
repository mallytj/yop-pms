package db_tests

import (
	"context"
	"strings"
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"

	hf "ollerod-pms/internal/helpers"
)

func TestDbRooms(t *testing.T) {
	ctx := context.Background()

	t.Run("TC-ROOM-07 - A room's name must be unique by property", func(t *testing.T) {
		t.Parallel()
		property := GenerateTestProperty(t, ctx)
		roomType := GenerateTestRoomType(t, ctx, property.ID)
		roomName := "Unique Room Name"

		// Create a room
		_, err := testDB.Exec(ctx,
			`INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1, $2, $3)`,
			property.ID, roomType.ID, roomName)
		assert.NoError(t, err)

		// Attempt to create another room with the same name in the same property
		_, err = testDB.Exec(ctx,
			`INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1, $2, $3)`,
			property.ID, roomType.ID, roomName)
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)

		// Create a different property and ensure the same room name can be used
		otherProperty := GenerateTestProperty(t, ctx)
		_, err = testDB.Exec(ctx,
			`INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1, $2, $3)`,
			otherProperty.ID, roomType.ID, roomName)
		assert.NoError(t, err, "Should be able to create a room with the same name in a different property")
	})

	t.Run("Field Constraints", func(t *testing.T) {
		t.Parallel()
		property := GenerateTestProperty(t, ctx)
		roomType := GenerateTestRoomType(t, ctx, property.ID)

		baseParams := []interface{}{
			property.ID,
			roomType.ID,
			"Standard Room", // name
		}

		query := `INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1, $2, $3)`

		constraintCases := []hf.ConstraintTest{
			{
				Name:        "TC-ROOM-02 - The rooms name must not exceed 75 characters",
				Field:       "name",
				Value:       strings.Repeat("a", 76),
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  2,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, query, baseParams, constraintCases)
	})

	t.Run("Foreign Key Constraints", func(t *testing.T) {
		t.Parallel()
		property := GenerateTestProperty(t, ctx)
		roomType := GenerateTestRoomType(t, ctx, property.ID)

		baseParamsOne := []interface{}{
			property.ID,
			roomType.ID,
			faker.Word(), // name
		}

		baseParamsTwo := []interface{}{
			property.ID,
			roomType.ID,
			faker.Word() + "_", // Different name to avoid unique constraint violation
		}

		query := `INSERT INTO inventory.rooms (property_id, room_type_id, name) VALUES ($1, $2, $3)`

		fkCases := []hf.FKExistenceTest{
			{
				Name:       "TC-ROOM-03 - The rooms property must exist",
				FakeIDIdx:  0,
				RealID:     property.ID,
				BaseParams: baseParamsOne,
			},
			{
				Name:       "TC-ROOM-04 - The rooms room type must exist",
				FakeIDIdx:  1,
				RealID:     roomType.ID,
				BaseParams: baseParamsTwo,
			},
		}

		hf.RunFKExistenceTests(t, ctx, testDB, query, fkCases)
	})
}
