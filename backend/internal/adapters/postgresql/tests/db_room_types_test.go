package db_tests

import (
	"context"
	"strings"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"github.com/stretchr/testify/assert"
)

func TestDbRoomTypes(t *testing.T) {
	ctx := context.Background()

	insertQuery := `INSERT INTO inventory.room_types (property_id, name, code, std_occupancy, min_occupancy, max_occupancy) VALUES ($1, $2, $3, $4, $5, $6)`

	t.Run("TC-RTYPE-02 - The room type's property must exist", func(t *testing.T) {
		t.Parallel()

		property := GenerateTestProperty(t, ctx)

		baseParams := TestRoomType{
			PropertyID:   property.ID,
			Name:         "Single",
			Code:         "SNG",
			StdOccupancy: 2,
			MinOccupancy: 1,
			MaxOccupancy: 2,
		}

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-RTYPE-02 - Property must exist",
				FakeIDIdx: 0,
				RealID:    property.ID,
				BaseParams: hf.StructToSlice(baseParams),
			},
		})
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		property := GenerateTestProperty(t, ctx)

		baseParams := TestRoomType{
			PropertyID:   property.ID,
			Name:         "Double",
			Code:         "DBL",
			StdOccupancy: 2,
			MinOccupancy: 1,
			MaxOccupancy: 2,
		}

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-RTYPE-03 - Name is required",
				Field:       "name",
				Value:       nil,
				FieldIndex:  1,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-RTYPE-05 - Code is required",
				Field:       "code",
				Value:       nil,
				FieldIndex:  2,
				ExpectedErr: hf.NotNullViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(baseParams), cases)
	})

	t.Run("Character Limits", func(t *testing.T) {
		t.Parallel()

		property := GenerateTestProperty(t, ctx)

		baseParams := TestRoomType{
			PropertyID:   property.ID,
			Name:         "Suite",
			Code:         "STE",
			StdOccupancy: 2,
			MinOccupancy: 1,
			MaxOccupancy: 2,
		}

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-RTYPE-04 - Name must not exceed 75 characters",
				Field:       "name",
				Value:       strings.Repeat("a", 76),
				FieldIndex:  1,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RTYPE-06 - Code must not exceed 7 characters",
				Field:       "code",
				Value:       strings.Repeat("A", 8),
				FieldIndex:  2,
				ExpectedErr: hf.CheckViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(baseParams), cases)
	})

	t.Run("Occupancy Constraints", func(t *testing.T) {
		t.Parallel()

		property := GenerateTestProperty(t, ctx)

		// Base occupancy values chosen so each test case isolates a single
		// constraint violation: std=4, min=2, max=6.
		baseParams := TestRoomType{
			PropertyID:   property.ID,
			Name:         "Family",
			Code:         "FAM",
			StdOccupancy: 4,
			MinOccupancy: 2,
			MaxOccupancy: 6,
		}

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-RTYPE-07 - Standard occupancy must be a positive integer",
				Field:       "std_occupancy",
				Value:       0,
				FieldIndex:  3,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RTYPE-08 - Min occupancy must be a positive integer",
				Field:       "min_occupancy",
				Value:       0,
				FieldIndex:  4,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RTYPE-09 - Min occupancy must be less than or equal to standard occupancy",
				Field:       "min_occupancy",
				Value:       5, // exceeds std_occupancy of 4
				FieldIndex:  4,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RTYPE-10 - Max occupancy must be a positive integer",
				Field:       "max_occupancy",
				Value:       0,
				FieldIndex:  5,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RTYPE-11 - Max occupancy must be greater than or equal to standard occupancy",
				Field:       "max_occupancy",
				Value:       3, // less than std_occupancy of 4
				FieldIndex:  5,
				ExpectedErr: hf.CheckViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(baseParams), cases)
	})

	t.Run("Unique Fields", func(t *testing.T) {
		t.Parallel()

		property := GenerateTestProperty(t, ctx)
		anotherProperty := GenerateTestProperty(t, ctx)

		existing := GenerateTestRoomType(t, ctx, property.ID)

		t.Run("TC-RTYPE-12 - Code must be unique per property", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx, insertQuery, property.ID, "Different Name", existing.Code, 2, 1, 2)
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)

			t.Run("Pass when code is reused for different property", func(t *testing.T) {
				t.Parallel()

				_, err := testDB.Exec(ctx, insertQuery, anotherProperty.ID, "Another Name", existing.Code, 2, 1, 2)
				assert.NoError(t, err, "Expected no error when reusing code for different property, got: %v", err)
			})
		})

		t.Run("TC-RTYPE-13 - Name must be unique per property", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx, insertQuery, property.ID, existing.Name, "NEWCD", 2, 1, 2)
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)

			t.Run("Pass when name is reused for different property", func(t *testing.T) {
				t.Parallel()

				_, err := testDB.Exec(ctx, insertQuery, anotherProperty.ID, existing.Name, "DIFF1", 2, 1, 2)
				assert.NoError(t, err, "Expected no error when reusing name for different property, got: %v", err)
			})
		})
	})
}
