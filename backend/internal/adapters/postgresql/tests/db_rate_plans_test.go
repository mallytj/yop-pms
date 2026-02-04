//go:build ignore
package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbRatePlans(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)

	insertQuery := `INSERT INTO pricing.rate_plans (property_id, name, code, description, is_active, parent_rate_plan_id) VALUES ($1, $2, $3, $4, $5, $6)`

	// Fill params
	params := TestRatePlan{
		PropertyID:       property.ID,
		Name:             "Standard Rate Plan",
		Code:             "STD",
		Description:      "A standard rate plan for testing",
		IsActive:         true,
		ParentRatePlanID: nil,
	}

	// Using this hacky way to get params as slice for reuse in tests
	// Because there is other fields in the struct, which are not needed for this test
	// TODO refactor helpers to make this cleaner
	// Not a priority right now, as we just need to get the tests done
	paramsSlice := []interface{}{
		params.PropertyID,
		params.Name,
		params.Code,
		params.Description,
		params.IsActive,
		params.ParentRatePlanID,
	}

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-RPLAN-02 - Name is required",
				Field:       "name",
				Value:       nil,
				FieldIndex:  1,
				ExpectedErr: hf.NotNullViolationCode,
			},
			{
				Name:        "TC-RPLAN-03 - Code is required",
				Field:       "code",
				Value:       nil,
				FieldIndex:  2,
				ExpectedErr: hf.NotNullViolationCode,
			},
		}
		hf.RunConstraintTests(t, ctx, testDB, insertQuery, paramsSlice, cases)
	})

	t.Run("TC-RPLAN-04 - Derivation rule must be in valid format", func(t *testing.T) {
		t.Parallel()
		parentRatePlan := GenerateTestRatePlan(t, ctx, property.ID)
		params.ParentRatePlanID = &parentRatePlan.ID

		// Derviation rule must be in format of
		// {
		//  "type": "percentage" | "fixed_amount",
		//  "value": int
		// }
		invalidDerivationRule := RPDerivationRule{
			Type:  "invalid_type",
			Value: 10,
		}

		params.DerivationRule = &invalidDerivationRule

		insertQueryWithDerivation := `INSERT INTO pricing.rate_plans (property_id, name, code, description, is_active, parent_rate_plan_id, derivation_rule) VALUES ($1, $2, $3, $4, $5, $6, $7)`

		_, err := testDB.Exec(ctx, insertQueryWithDerivation,
			params.PropertyID,
			params.Name,
			params.Code,
			params.Description,
			params.IsActive,
			params.ParentRatePlanID,
			params.DerivationRule,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for invalid derivation rule, got: %v", err)
	})

	t.Run("Character Limits", func(t *testing.T) {
		t.Parallel()

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-RPLAN-06 - Code must not exceed 7 characters",
				Field:       "code",
				FieldIndex:  2,
				Value:       strings.Repeat("A", 8), // 8 characters
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RPLAN-07 - Name must not exceed 30 characters",
				Field:       "name",
				FieldIndex:  1,
				Value:       strings.Repeat("A", 31), // 31 characters
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-RPLAN-08 - Description must not exceed 300 characters",
				Field:       "description",
				FieldIndex:  3,
				Value:       strings.Repeat("A", 301), // 301 characters
				ExpectedErr: hf.CheckViolationCode,
			},
		}
		hf.RunConstraintTests(t, ctx, testDB, insertQuery, paramsSlice, cases)
	})

	t.Run("TC-RPLAN-12 - If derivation rule is set, parent rate plan must be set", func(t *testing.T) {
		t.Parallel()
		derivationRule := RPDerivationRule{
			Type:  "percentage",
			Value: 10,
		}

		params.DerivationRule = &derivationRule
		params.ParentRatePlanID = nil

		insertQueryWithDerivation := `INSERT INTO pricing.rate_plans (property_id, name, code, description, is_active, parent_rate_plan_id, derivation_rule) VALUES ($1, $2, $3, $4, $5, $6, $7)`

		_, err := testDB.Exec(ctx, insertQueryWithDerivation,
			params.PropertyID,
			params.Name,
			params.Code,
			params.Description,
			params.IsActive,
			params.ParentRatePlanID,
			params.DerivationRule,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for missing parent rate plan when derivation rule is set, got: %v", err)
	})

	t.Run("TC-RPLAN-13 - The parent rate plan must belong to the same property", func(t *testing.T) {
		t.Parallel()
		// Create another property
		anotherProperty := GenerateTestProperty(t, ctx)
		// Create a rate plan for another property
		parentRatePlan := GenerateTestRatePlan(t, ctx, anotherProperty.ID)

		derivationRule := RPDerivationRule{
			Type:  "percentage",
			Value: 20,
		}

		params.DerivationRule = &derivationRule
		params.ParentRatePlanID = &parentRatePlan.ID

		insertQueryWithDerivation := `INSERT INTO pricing.rate_plans (property_id, name, code, description, is_active, parent_rate_plan_id, derivation_rule) VALUES ($1, $2, $3, $4, $5, $6, $7)`

		_, err := testDB.Exec(ctx, insertQueryWithDerivation,
			params.PropertyID,
			params.Name,
			params.Code,
			params.Description,
			params.IsActive,
			params.ParentRatePlanID,
			params.DerivationRule,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error for parent rate plan from different property, got: %v", err)
	})

	t.Run("TC-RPLAN-14 - If derivation rule is set, there must be a parent rate plan", func(t *testing.T) {
		t.Parallel()

		derivationRule := RPDerivationRule{
			Type:  "fixed_amount",
			Value: 50,
		}

		params.DerivationRule = &derivationRule
		params.ParentRatePlanID = nil

		insertQueryWithDerivation := `INSERT INTO pricing.rate_plans (property_id, name, code, description, is_active, parent_rate_plan_id, derivation_rule) VALUES ($1, $2, $3, $4, $5, $6, $7)`

		_, err := testDB.Exec(ctx, insertQueryWithDerivation,
			params.PropertyID,
			params.Name,
			params.Code,
			params.Description,
			params.IsActive,
			params.ParentRatePlanID,
			params.DerivationRule,
		)

		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for missing parent rate plan when derivation rule is set, got: %v", err)
	})
}
