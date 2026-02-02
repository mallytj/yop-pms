package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"
)

func TestDbTaxRules(t *testing.T) {
	ctx := context.Background()

	// Create a property
	property := GenerateTestProperty(t, ctx)

	params := TestTaxRule{
		PropertyID:     property.ID,
		Name:           "Standard Tax",
		Description:    "Standard tax rate for testing",
		TaxPercentage:  10.00,
		IsTaxInclusive: false,
	}

	insertQuery := `INSERT INTO finance.tax_rules (property_id, name, description, tax_percentage, is_tax_inclusive, created_at, updated_at) 
					VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`

	hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
		{
			Name:       "TC-TAXR-02 - Property must exist",
			FakeIDIdx:  0,
			RealID:     property.ID,
			BaseParams: hf.StructToSlice(params),
		},
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-TAXR-02 - Name is required",
				Field:       "name",
				Value:       nil,
				FieldIndex:  1,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-TAXR-07 - Tax percentage is positive (subtest for NOT NULL)",
				Field:       "tax_percentage",
				Value:       nil,
				FieldIndex:  3,
				ExpectedErr: hf.NotNullViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), cases)
	})

	t.Run("Tax percentage Limits", func(t *testing.T) {
		t.Parallel()

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-TAXR-08 - Tax percentage must be > 0.00",
				Field:       "tax_percentage",
				Value:       -5.00,
				FieldIndex:  3,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-TAXR-06 - Tax percentage must be <= 75.00",
				Field:       "tax_percentage",
				Value:       80.00,
				FieldIndex:  3,
				ExpectedErr: hf.CheckViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), cases)
	})

	t.Run("Character Limits", func(t *testing.T) {
		t.Parallel()

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-TAXR-04 - Name must not exceed 50 characters",
				Field:       "name",
				Value:       strings.Repeat("a", 51),
				FieldIndex:  1,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-TAXR-05 - Description must not exceed 250 characters",
				Field:       "description",
				Value:       strings.Repeat("a", 251),
				FieldIndex:  2,
				ExpectedErr: hf.CheckViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), cases)
	})

	t.Run("Unique Constraints", func(t *testing.T) {
		t.Parallel()

		// First, insert a valid tax rule
		_, err := testDB.Exec(ctx, insertQuery,
			params.PropertyID,
			params.Name,
			params.Description,
			params.TaxPercentage,
			params.IsTaxInclusive,
		)
		if err != nil {
			t.Fatalf("Failed to insert initial tax rule: %v", err)
		}

		// Now, attempt to insert another tax rule with the same name for the same property
		duplicateParams := params
		duplicateParams.Description = "Another description" // Change description to avoid conflict there

		_, err = testDB.Exec(ctx, insertQuery,
			duplicateParams.PropertyID,
			duplicateParams.Name,
			duplicateParams.Description,
			duplicateParams.TaxPercentage,
			duplicateParams.IsTaxInclusive,
		)
		if !hf.CheckErrorCode(err, hf.UniqueViolationCode) {
			t.Errorf("Expected unique violation error for duplicate tax rule name, got: %v", err)
		}

		t.Run("pass if different property", func(t *testing.T) {
			t.Parallel()

			// Create another property
			anotherProperty := GenerateTestProperty(t, ctx)

			// Attempt to insert a tax rule with the same name but for a different property
			_, err = testDB.Exec(ctx, insertQuery,
				anotherProperty.ID,
				params.Name,
				"Different property description",
				params.TaxPercentage,
				params.IsTaxInclusive,
			)
			if err != nil {
				t.Errorf("Expected no error when inserting tax rule with same name for different property, got: %v", err)
			}
		})
	})
}
