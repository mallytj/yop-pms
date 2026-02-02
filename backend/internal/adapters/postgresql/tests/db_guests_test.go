package db_tests

import (
	"context"
	"strings"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"github.com/stretchr/testify/assert"
)

func TestDbGuests(t *testing.T) {
	ctx := context.Background()

	t.Run("TC-GUEST-02 - The guests property must exist", func(t *testing.T) {
		t.Parallel()
		// Create test user
		params := TestGuest{
			FirstName: "Test",
			LastName:  "Guest",
			Email:     "test.guest@example.com",
			Phone:     "1234567890",
		}

		insertQuery := `INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number) VALUES ($1, $2, $3, $4, $5)`

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-GUEST-02 - Property must exist",
				FakeIDIdx: 0, // Non-existent property
				RealID:    GenerateTestProperty(t, ctx).ID,
				BaseParams: hf.StructToSlice(params),
			},
		})
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()
		// Create a property
		property := GenerateTestProperty(t, ctx)

		// Base params
		baseParams := TestGuest{
			PropertyID: property.ID,              // PropertyID
			FirstName:  "Test",                   // FirstName
			LastName:   "Guest",                  // LastName
			Email:      "test.guest@example.com", // Email
			Phone:      "1234567890",             // Phone
		}

		tests := []hf.ConstraintTest{
			{
				Name:        "TC-GUEST-03 - First name is required",
				Field:       "first_name",
				Value:       nil,
				ExpectedErr: hf.NotNullViolationCode,
				FieldIndex:  1, // First name is the 3rd field (index 2)
			},
			{
				Name:        "TC-GUEST-06 - Last name is required",
				Field:       "last_name",
				Value:       nil,
				ExpectedErr: hf.NotNullViolationCode,
				FieldIndex:  2, // Last name is the 4th field (index 3)
			},
		}

		insertQuery := `INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number) VALUES ($1, $2, $3, $4, $5)`

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(baseParams), tests)
	})

	t.Run("Character Limits", func(t *testing.T) {
		t.Parallel()
		// Create a property
		property := GenerateTestProperty(t, ctx)

		// Base params
		baseParams := TestGuest{
			PropertyID: property.ID,              // PropertyID
			FirstName:  "Test",                   // FirstName
			LastName:   "Guest",                  // LastName
			Email:      "test.guest@example.com", // Email
			Phone:      "1234567890",             // Phone
		}

		tests := []hf.ConstraintTest{
			{
				Name:        "TC-GUEST-04 - First name must not exceed 50 characters",
				Field:       "first_name",
				Value:       strings.Repeat("a", 51),
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  1, // First name is the 3rd field (index 2)
			},
			{
				Name:        "TC-GUEST-07 - Last name must not exceed 50 characters",
				Field:       "last_name",
				Value:       strings.Repeat("b", 51),
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  2, // Last name is the 4th field (index 3)
			},
		}

		insertQuery := `INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number) VALUES ($1, $2, $3, $4, $5)`

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(baseParams), tests)
	})

	t.Run("No special characters in names", func(t *testing.T) {
		t.Parallel()
		// Create a property
		property := GenerateTestProperty(t, ctx)

		// Base params
		baseParams := TestGuest{
			PropertyID: property.ID,              // PropertyID
			FirstName:  "Test",                   // FirstName
			LastName:   "Guest",                  // LastName
			Email:      "test.guest@example.com", // Email
			Phone:      "1234567890",             // Phone
		}

		tests := []hf.ConstraintTest{
			{
				Name:        "TC-GUEST-05 - First name must not contain special characters",
				Field:       "first_name",
				Value:       "Test@123",
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  1, // First name is the 3rd field (index 2)
			},
			{
				Name:        "TC-GUEST-08 - Last name must not contain special characters",
				Field:       "last_name",
				Value:       "Guest#456",
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  2, // Last name is the 4th field (index 3)
			},
		}

		insertQuery := `INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number) VALUES ($1, $2, $3, $4, $5)`

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(baseParams), tests)
	})

	t.Run("TC-GUEST-09 - Marketing opt-in defaults to false", func(t *testing.T) {
		t.Parallel()
		// Create a property
		property := GenerateTestProperty(t, ctx)

		// Insert guest without specifying marketing_opt_in
		var marketingOptIn bool
		err := testDB.QueryRow(ctx,
			`INSERT INTO identity.guests (property_id, first_name, last_name, email, phone_number)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING (marketing_opt_in)::BOOLEAN`,
			property.ID.String(), "OptIn", "Test", "optin.test@example.com", "1234567890").Scan(&marketingOptIn)
		assert.NoError(t, err)

		assert.False(t, marketingOptIn, "Expected marketing_opt_in to default to false")
	})
}
