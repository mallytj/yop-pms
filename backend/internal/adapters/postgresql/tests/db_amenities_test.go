package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDBAmenities(t *testing.T) {
	ctx := context.Background()

	t.Run("TC-AMEN-02 - The amenity's owner property must exist", func(t *testing.T) {
		t.Parallel()

		t.Run("Fail when property does not exist", func(t *testing.T) {
			t.Parallel()

			fakePropertyID := "00000000-0000-0000-0000-000000000000" // Non-existent property

			insertQuery := `INSERT INTO operations.amenities (property_id, name, description, is_active) VALUES ($1, $2, $3, $4)`

			// Attempt to create an amenity with a non-existent property
			_, err := testDB.Exec(ctx, insertQuery, fakePropertyID, "Pool", "Outdoor swimming pool", true)

			assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error, got: %v", err)
		})

		t.Run("Pass when property exists", func(t *testing.T) {
			t.Parallel()
			// Create a property
			property := GenerateTestProperty(t, ctx)

			insertQuery := `INSERT INTO operations.amenities (property_id, name, description, is_active) VALUES ($1, $2, $3, $4)`

			// Attempt to create an amenity with the existing property
			_, err := testDB.Exec(ctx, insertQuery, property.ID.String(), "Gym", "Fully equipped gym", true)

			assert.NoError(t, err)
		})
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		// Create a property
		property := GenerateTestProperty(t, ctx)

		baseParams := TestAmenity{
			PropertyID:  property.ID,
			Name:        "Spa",
			ShortCode:   "SPA01",
			Description: "Relaxing spa services",
			IsActive:    true,
		}

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-AMEN-03 - Name is required",
				Field:       "name",
				Value:       nil,
				FieldIndex:  1, // Index of the 'name' field in the insert query
				ExpectedErr: hf.NotNullViolationCode,
			},
		}

		query := `INSERT INTO operations.amenities (property_id, name, short_code, description, is_active) VALUES ($1, $2, $3, $4, $5)`

		paramsSlice := hf.StructToSlice(baseParams)
		paramsSlice = paramsSlice[1:] // Exclude PropertyID for the tests

		hf.RunConstraintTests(t, ctx, testDB, query, paramsSlice, cases)
	})

	t.Run("Character Limits", func(t *testing.T) {
		t.Parallel()

		// Create a property
		property := GenerateTestProperty(t, ctx)

		baseParams := TestAmenity{
			PropertyID:  property.ID,
			Name:        "Valid Amenity Name",
			ShortCode:   "AMN01",
			Description: "A valid description for the amenity.",
			IsActive:    true,
		}

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-AMEN-04 - Name must not exceed 100 characters",
				Field:       "name",
				Value:       strings.Repeat("a", 101),
				FieldIndex:  1, // Index of the 'name' field in the insert query
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-AMEN-05 - Short code must not exceed 5 characters",
				Field:       "short_code",
				Value:       strings.Repeat("a", 6),
				FieldIndex:  2, // Index of the 'short_code' field in the insert query
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-AMEN-06 - Amenities description must not exceed 250 characters",
				Field:       "description",
				Value:       strings.Repeat("a", 251),
				FieldIndex:  3, // Index of the 'description' field in the insert query
				ExpectedErr: hf.CheckViolationCode,
			},
		}
		query := `INSERT INTO operations.amenities (property_id, name, short_code, description, is_active) VALUES ($1, $2, $3, $4, $5)`

		paramsSlice := hf.StructToSlice(baseParams)
		paramsSlice = paramsSlice[1:] // Exclude ID PK for the tests

		hf.RunConstraintTests(t, ctx, testDB, query, paramsSlice, cases)
	})

	t.Run("Unique Fields", func(t *testing.T) {
		t.Parallel()

		// Create a property
		property := GenerateTestProperty(t, ctx)

		// Create another property
		anotherProperty := GenerateTestProperty(t, ctx)

		// First, create an amenity
		existingAmenity := TestAmenity{
			PropertyID:  property.ID,
			Name:        "Tennis Court",
			ShortCode:   "TNC01",
			Description: "Outdoor tennis court",
			IsActive:    true,
		}

		insertQuery := `INSERT INTO operations.amenities (property_id, name, short_code, description, is_active) VALUES ($1, $2, $3, $4, $5)`

		_, err := testDB.Exec(ctx, insertQuery, existingAmenity.PropertyID, existingAmenity.Name, existingAmenity.ShortCode, existingAmenity.Description, existingAmenity.IsActive)
		assert.NoError(t, err, "Failed to create initial amenity: %v", err)

		t.Run("TC-AMEN-07 - Short code must be unique per property", func(t *testing.T) {
			t.Parallel()

			// Attempt to create another amenity with the same short code for the same property
			_, err := testDB.Exec(ctx, insertQuery, property.ID, "New Amenity", existingAmenity.ShortCode, "Another description", true)

			// Check for unique violation error
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)

			// Subtest to ensure short code can be reused for different properties
			t.Run("Pass when short code is reused for different property", func(t *testing.T) {
				t.Parallel()

				// Attempt to create an amenity with the same short code for a different property
				_, err := testDB.Exec(ctx, insertQuery, anotherProperty.ID, "Another Amenity", existingAmenity.ShortCode, "Description for another property", true)

				// Should succeed without error
				assert.NoError(t, err, "Expected no error when reusing short code for different property, got: %v", err)
			})
		})

		t.Run("TC-AMEN-08 - Name must be unique per property", func(t *testing.T) {
			t.Parallel()

			// Attempt to create another amenity with the same name for the same property
			_, err := testDB.Exec(ctx, insertQuery, property.ID, existingAmenity.Name, "NEWSH", "Another description", true)

			// Check for unique violation error
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)

			// Subtest to ensure name can be reused for different properties
			t.Run("Pass when name is reused for different property", func(t *testing.T) {
				t.Parallel()

				// Attempt to create an amenity with the same name for a different property
				_, err := testDB.Exec(ctx, insertQuery, anotherProperty.ID, existingAmenity.Name, "DIFF1", "Description for another property", true)

				// Should succeed without error
				assert.NoError(t, err, "Expected no error when reusing name for different property, got: %v", err)
			})
		})
	})

	t.Run("TC-AMEN-11 - Short code must follow the format '^[A-Z0-9_/]{2,5}$'", func(t *testing.T) {
		t.Parallel()

		// Create a property
		property := GenerateTestProperty(t, ctx)

		tests := []FieldTestCase{
			{
				example: "AMN1", // Valid short code
				result:  true,
			},
			{
				example: "A_2", // Underscore allowed
				result:  true,
			},
			{
				example: "B/3", // Slash allowed
				result:  true,
			},
			{
				example: "123", // Numeric only
				result:  true,
			},
			{
				example: "!@#", // Special characters not allowed
				result:  false,
			},
			{
				example: "A", // Less than 2 characters
				result:  false,
			},
			{
				example: "TOOLONG", // More than 5 characters
				result:  false,
			},
			{
				example: "AB C", // Space included
				result:  false,
			},
		}

		insertQuery := `INSERT INTO operations.amenities (property_id, name, short_code, description, is_active) VALUES ($1, $2, $3, $4, $5)`

		for i, tc := range tests {
			t.Run("Invalid short code: "+tc.example, func(t *testing.T) {
				t.Parallel()

				// Attempt to create an amenity with an invalid short code
				_, err := testDB.Exec(ctx, insertQuery, property.ID, "Amenity with invalid code "+strings.Repeat("a", i), tc.example, "Description", true)
				if tc.result {
					// Should succeed without error
					assert.NoError(t, err, "Expected no error for valid short code '%s', got: %v", tc.example, err)
					return
				}

				// Check for check violation error
				assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for short code '%s', got: %v", tc.example, err)
			})
		}
	})
}
