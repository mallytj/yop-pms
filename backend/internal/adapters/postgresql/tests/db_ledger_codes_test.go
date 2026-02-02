package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbLedgerCodes(t *testing.T) {
	ctx := context.Background()

	// Create a property
	property := GenerateTestProperty(t, ctx)

	// Create a tax rule
	taxRule := GenerateTestTaxRule(t, ctx, property.ID)

	params := TestLedgerCode{
		PropertyID:  property.ID,
		Code:        "LC-1001",
		Description: "Sample Ledger Code",
		TaxRuleID:   taxRule.ID,
	}

	paramsTwo := params // For cleanup
	paramsTwo.Code = "LC-1002"

	t.Cleanup(func() {
		testDB.Exec(ctx, "DELETE FROM finance.ledger_codes WHERE property_id = $1", property.ID)
	})

	insertQuery := `INSERT INTO finance.ledger_codes (property_id, code, description, tax_rule) 
					VALUES ($1, $2, $3, $4)`

	hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
		{
			Name:       "TC-LEDG-02 - Property must exist",
			FakeIDIdx:  0,
			RealID:     property.ID,
			BaseParams: hf.StructToSlice(params),
		},
		{
			Name:       "TC-LEDG-07 - Tax rule must exist",
			FakeIDIdx:  3,
			RealID:     taxRule.ID,
			BaseParams: hf.StructToSlice(paramsTwo),
		},
	})

	constraintTests := []hf.ConstraintTest{
		{
			Name:        "TC-LEDG-03 - Code is required",
			Field:       "code",
			Value:       nil,
			FieldIndex:  1,
			ExpectedErr: hf.NotNullViolationCode,
		},
		{
			Name:        "TC-LEDG-05 - Code must not exceed 50 characters",
			Field:       "code",
			Value:       strings.Repeat("a", 51),
			FieldIndex:  1,
			ExpectedErr: hf.CheckViolationCode,
		},
		{
			Name:        "TC-LEDG-06 - Description must not exceed 250 characters",
			Field:       "description",
			Value:       strings.Repeat("a", 251),
			FieldIndex:  2,
			ExpectedErr: hf.CheckViolationCode,
		},
	}

	hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), constraintTests)

	t.Run("TC-LEDC-04 - Unique code per property", func(t *testing.T) {
		t.Parallel()

		_, err := testDB.Exec(ctx, insertQuery,
			params.PropertyID,
			params.Code, // Duplicate code
			"Another Description",
			params.TaxRuleID,
		)
		assert.NoError(t, err)

		_, err = testDB.Exec(ctx, insertQuery,
			params.PropertyID,
			params.Code, // Duplicate code
			"Another Description",
			params.TaxRuleID,
		)

		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "expected unique violation error, got: %v", err)

		t.Run("Works for different properties", func(t *testing.T) {
			t.Parallel()
			// Create another property
			anotherProperty := GenerateTestProperty(t, ctx)

			_, err := testDB.Exec(ctx, insertQuery,
				anotherProperty.ID,
				params.Code, // Same code but different property
				"Another Description",
				params.TaxRuleID,
			)
			assert.NoError(t, err, "failed to insert ledger code for different property: %v", err)
		})
	})
}
