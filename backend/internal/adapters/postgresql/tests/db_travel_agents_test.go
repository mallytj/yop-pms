package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbTravelAgents(t *testing.T) {
	ctx := context.Background()

	// Create a property
	property := GenerateTestProperty(t, ctx)
	baseParams := TestTravelAgent{
		PropertyID:        property.ID,
		Name:              "Global Travels",
		ContactEmail:      "contact@globaltravels.com",
		ContactPhone:      "+123456789",
		AgencyNotes:       "Global Travel Agency",
		IATACode:          "GLO",
		CommissionPercent: 10.0,
	}

	// Prepare parameters slice for reuse in tests
	// Indexes correspond to the insert query placeholders
	paramsSlice := []interface{}{
			baseParams.PropertyID,
			baseParams.Name,
			baseParams.ContactEmail, 
			baseParams.ContactPhone,
			baseParams.AgencyNotes, 
			baseParams.IATACode, 
			baseParams.CommissionPercent,
		}

	insertQuery := `INSERT INTO identity.travel_agents (property_id, name, contact_email, contact_phone, agency_notes, iata_code, commission_percent) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		t.Run("TC-TRAV-02 - Name is required", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx, insertQuery, baseParams.PropertyID, nil, baseParams.ContactEmail, baseParams.ContactPhone, baseParams.IATACode, baseParams.CommissionPercent)

			// Check for null value violation error
			assert.Equal(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), true, "Expected not null violation error, got: %v", err)
		})
	})
	t.Run("Char Limits", func(t *testing.T) {
		t.Parallel()

		tests := []hf.ConstraintTest{
			{
				Name:        "TC-TRAV-05 - Name must not exceed 100 characters",
				Field:       "name",
				FieldIndex:  1,
				Value:       strings.Repeat("A", 101), // 101 characters
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-TRAV-06 - Agency notes must not exceed 1000 characters",
				Field:       "agency_notes",
				FieldIndex:  4,
				Value:       strings.Repeat("A", 1001), // 1001 characters
				ExpectedErr: hf.CheckViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, paramsSlice, tests)
	})

	t.Run("TC-TRAV-07 - Name must be unique per property", func(t *testing.T) {
		t.Parallel()

		// First, create a travel agent
		_, err := testDB.Exec(ctx, insertQuery, baseParams.PropertyID, baseParams.Name, baseParams.ContactEmail, baseParams.ContactPhone, baseParams.AgencyNotes, baseParams.IATACode, baseParams.CommissionPercent)
		assert.NoError(t, err, "Failed to create initial travel agent: %v", err)

		// Attempt to create another travel agent with the same name for the same property
		_, err = testDB.Exec(ctx, insertQuery, baseParams.PropertyID, baseParams.Name, baseParams.ContactEmail, baseParams.ContactPhone, baseParams.AgencyNotes, baseParams.IATACode, baseParams.CommissionPercent)

		// Check for unique violation error
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)
	})


	t.Run("Commission Percentage Bounds", func(t *testing.T) {
		t.Parallel()

		tests := []hf.ConstraintTest{
			{
				Name:        "TC-TRAV-08 - Commission percent must be positive",
				Field:       "commission_percent",
				FieldIndex:  6,
				Value:       -5.0,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-TRAV-09 - Commission percent must not exceed 75.00",
				Field:       "commission_percent",
				FieldIndex:  6,
				Value:       80.0,
				ExpectedErr: hf.CheckViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, paramsSlice, tests)
	})
}
