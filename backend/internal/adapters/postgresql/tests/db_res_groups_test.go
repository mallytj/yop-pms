package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbReservationGroups(t *testing.T) {
	property := GenerateTestProperty(t, context.Background())

	insertQuery := `INSERT INTO operations.reservation_groups (property_id, name, notes) VALUES ($1, $2, $3)`

	baseParams := TestReservationGroup{
		PropertyID: property.ID,
		Name:       "Test Reservation Group",
		Notes:      "A test reservation group",
	}

	paramsSlice := []interface{}{
		baseParams.PropertyID,
		baseParams.Name,
		baseParams.Notes,
	}

	t.Run("FK Existence Tests", func(t *testing.T) {
		t.Parallel()

		tests := []hf.FKExistenceTest{
			{
				Name:       "TC-RESG-02 - Property must exist",
				FakeIDIdx:  0,
				RealID:     property.ID,
				BaseParams: paramsSlice,
			},
			// { -- Not implemented yet
			// 	Name: "TC-RESG-03 - Master folio must exist if set",
			// 	// Assuming MasterFolioID would be at index 3 if it were included in BaseParams
			// 	FakeIDIdx:  3,
			// 	RealID:     uuid.New(), // Replace with actual existing folio ID if needed
			// 	BaseParams: hf.StructToSlice(baseParams),
			// },
		}

		hf.RunFKExistenceTests(t, context.Background(), testDB, insertQuery, tests)
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		t.Parallel()

		constraintTests := []hf.ConstraintTest{
			{
				Name:        "TC-RGRP-07 - Name should not exceed 50 characters",
				Field:       "name",
				FieldIndex:  1,
				Value:       strings.Repeat("A", 51),
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-RGRP-08 - Notes should not exceed 2500 characters",
				Field:       "notes",
				FieldIndex:  2,
				Value:       strings.Repeat("A", 2501),
				ExpectedErr: hf.NotNullViolationCode,
			},
		}
		hf.RunConstraintTests(t, context.Background(), testDB, insertQuery, paramsSlice, constraintTests)
	})

	t.Run("TC-RGRP-05 - The code must be in format GRP-XXXXX where XXXXX is a 5 digit number", func(t *testing.T) {
		t.Parallel()
		createdGroup := &TestReservationGroup{}

		err := testDB.QueryRow(context.Background(),
			`INSERT INTO operations.reservation_groups (property_id, name, notes) VALUES ($1, $2, $3) RETURNING id, code`,
			baseParams.PropertyID,
			baseParams.Name,
			baseParams.Notes,
		).Scan(&createdGroup.ID, &createdGroup.Code)
		assert.NoError(t, err, "Failed to create reservation group: %v", err)

		// Validate code format
		matched, err := hf.MatchRegex(`^GRP-\d{5}$`, createdGroup.Code)
		assert.NoError(t, err, "Error matching regex: %v", err)
		assert.True(t, matched, "Reservation group code does not match expected format: got %s", createdGroup.Code)
	})

	t.Run("TC-RGRP-06 - Inserting an item must increment the code number correctly", func(t *testing.T) {
		t.Parallel()
		createdGroup1 := &TestReservationGroup{}
		createdGroup2 := &TestReservationGroup{}

		// Create first reservation group
		err := testDB.QueryRow(context.Background(),
			`INSERT INTO operations.reservation_groups (property_id, name, notes) VALUES ($1, $2, $3) RETURNING id, code`,
			baseParams.PropertyID,
			baseParams.Name,
			baseParams.Notes,
		).Scan(&createdGroup1.ID, &createdGroup1.Code)
		assert.NoError(t, err, "Failed to create first reservation group: %v", err)

		// Create second reservation group
		err = testDB.QueryRow(context.Background(),
			`INSERT INTO operations.reservation_groups (property_id, name, notes) VALUES ($1, $2, $3) RETURNING id, code`,
			baseParams.PropertyID,
			baseParams.Name,
			baseParams.Notes,
		).Scan(&createdGroup2.ID, &createdGroup2.Code)
		assert.NoError(t, err, "Failed to create second reservation group: %v", err)

		// Extract numeric parts of the codes
		num1Str := strings.TrimPrefix(createdGroup1.Code, "GRP-")
		num2Str := strings.TrimPrefix(createdGroup2.Code, "GRP-")

		num1, err := strconv.Atoi(num1Str)
		assert.NoError(t, err, "Error parsing first code number: %v", err)

		num2, err := strconv.Atoi(num2Str)
		assert.NoError(t, err, "Error parsing second code number: %v", err)

		// Check that the second code number is exactly one greater than the first
		assert.Equal(t, num1+1, num2, "Second reservation group code number is not incremented correctly: got %d and %d", num1, num2)
	})
}
