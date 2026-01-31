package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// One file, as each relation table is small and has few constraints

func TestDbPropertyAmenities(t *testing.T) {
	ctx := context.Background()
	// Create a property
	property := GenerateTestProperty(t, ctx)

	// Create an amenity
	amenity := GenerateTestAmenity(t, ctx, property.ID)

	insertQuery := `INSERT INTO relations.property_amenities (property_id, amenity_id) VALUES ($1, $2)`

	t.Run("TC-PRAM-02 - The property's amenity must reference an existing property", func(t *testing.T) {
		t.Parallel()
		t.Run("Fail when property does not exist", func(t *testing.T) {
			t.Parallel()

			fakePropertyID := "00000000-0000-0000-0000-000000000000" // Non-existent property

			// Attempt to create a property amenity with a non-existent property
			_, err := testDB.Exec(ctx, insertQuery, fakePropertyID, uuid.New().String())

			assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error, got: %v", err)
		})

		t.Run("Pass when property exists", func(t *testing.T) {
			t.Parallel()

			// Attempt to create a property amenity with the existing property
			_, err := testDB.Exec(ctx, insertQuery, property.ID.String(), amenity.ID.String())

			assert.NoError(t, err)
		})
	})

	t.Run("TC-PRAM-03 - The property's amenity must reference an existing amenity", func(t *testing.T) {
		t.Parallel()
		t.Run("Fail when amenity does not exist", func(t *testing.T) {
			t.Parallel()

			fakeAmenityID := "00000000-0000-0000-0000-000000000000" // Non-existent amenity

			// Attempt to create a property amenity with a non-existent amenity
			_, err := testDB.Exec(ctx, insertQuery, property.ID.String(), fakeAmenityID)

			assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error, got: %v", err)
		})

		t.Run("Pass when amenity exists", func(t *testing.T) {
			t.Parallel()

			newAmenity := GenerateTestAmenity(t, ctx, property.ID)

			// Attempt to create a property amenity with the existing amenity
			_, err := testDB.Exec(ctx, insertQuery, property.ID.String(), newAmenity.ID.String())

			assert.NoError(t, err)
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
