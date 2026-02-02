package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"testing"

	"github.com/stretchr/testify/assert"
)

// One file, as each relation table is small and has few constraints

func TestDbPropertyAmenities(t *testing.T) {
	ctx := context.Background()
	// Create a property
	property := GenerateTestProperty(t, ctx)

	// Create an amenity
	amenity := GenerateTestAmenity(t, ctx, property.ID)

	// Second amenity to facilitate FK tests
	anotherAmenity := GenerateTestAmenity(t, ctx, property.ID)

	insertQuery := `INSERT INTO relations.property_amenities (property_id, amenity_id) VALUES ($1, $2)`

	t.Run("TC-PRAM-02 - The property's amenity must reference an existing property", func(t *testing.T) {
		t.Parallel()

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-PRAM-02 - Property must exist",
				FakeIDIdx: 0,
				RealID:    property.ID,
				BaseParams: []interface{}{
					property.ID,
					amenity.ID,
				},
			},
			{
				Name:      "TC-PRAM-03 - Amenity must exist",
				FakeIDIdx: 1,
				RealID:    anotherAmenity.ID,
				BaseParams: []interface{}{
					property.ID,
					anotherAmenity.ID,
				},
			},
		})
	})

	t.Run("TC-PRAM-06 - The amenity must have the same property as the property amenity", func(t *testing.T) {
		t.Parallel()

		// Create another property
		anotherProperty := GenerateTestProperty(t, ctx)

		// Attempt to create a property amenity with mismatched property and amenity
		_, err := testDB.Exec(ctx, insertQuery, anotherProperty.ID.String(), amenity.ID.String())

		assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected raise exception error, got: %v", err)
	})
}

func TestDbRoomTypeAmenities(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	roomType := GenerateTestRoomType(t, ctx, property.ID)
	amenity := GenerateTestAmenity(t, ctx, property.ID)
	anotherAmenity := GenerateTestAmenity(t, ctx, property.ID)

	insertQuery := `INSERT INTO relations.room_type_amenities (property_id, room_type_id, amenity_id) VALUES ($1, $2, $3)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		// t.Parallel()

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-RTYPE-16 - The room type must exist",
				FakeIDIdx: 1,
				RealID:    roomType.ID,
				BaseParams: []interface{}{
					property.ID,
					roomType.ID,
					amenity.ID,
				},
			},
			{
				Name:      "TC-RTYPE-17 - The amenity must exist",
				FakeIDIdx: 2,
				RealID:    anotherAmenity.ID,
				BaseParams: []interface{}{
					property.ID,
					roomType.ID,
					anotherAmenity.ID,
				},
			},
		})
	})

	t.Run("TC-RTYPE-20 - The amenity must be of the same property as the room type", func(t *testing.T) {
		t.Parallel()

		anotherProperty := GenerateTestProperty(t, ctx)
		crossPropertyAmenity := GenerateTestAmenity(t, ctx, anotherProperty.ID)

		_, err := testDB.Exec(ctx, insertQuery, property.ID, roomType.ID, crossPropertyAmenity.ID)

		assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation for cross-property amenity, got: %v", err)
	})
}

func TestDbRoomAmenities(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	roomType := GenerateTestRoomType(t, ctx, property.ID)
	room := GenerateTestRoom(t, ctx, property.ID, roomType.ID)
	amenity := GenerateTestAmenity(t, ctx, property.ID)
	anotherAmenity := GenerateTestAmenity(t, ctx, property.ID)

	insertQuery := `INSERT INTO relations.room_amenities (property_id, room_id, amenity_id) VALUES ($1, $2, $3)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		t.Parallel()

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-RMAM-02 - Room must exist",
				FakeIDIdx: 0,
				RealID:    property.ID,
				BaseParams: []interface{}{
					property.ID,
					room.ID,
					amenity.ID,
				},
			},
			{
				Name:      "TC-RMAM-03 - Amenity must exist",
				FakeIDIdx: 2,
				RealID:    anotherAmenity.ID,
				BaseParams: []interface{}{
					property.ID,
					room.ID,
					anotherAmenity.ID,
				},
			},
		})
	})

	t.Run("TC-RMAM-06 - The amenity must have the same property as the room", func(t *testing.T) {
		t.Parallel()

		anotherProperty := GenerateTestProperty(t, ctx)
		crossPropertyAmenity := GenerateTestAmenity(t, ctx, anotherProperty.ID)

		_, err := testDB.Exec(ctx, insertQuery, property.ID, room.ID, crossPropertyAmenity.ID)

		assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation for cross-property amenity, got: %v", err)
	})
}
