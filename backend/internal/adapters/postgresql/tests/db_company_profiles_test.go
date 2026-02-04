//go:build ignore
package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbCompanyProfiles(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	negotiatedRatePlan := GenerateTestRatePlan(t, ctx, property.ID)

	params := TestCompanyProfile{
		PropertyID:           property.ID,
		TaxID:                "VAT123456",
		NegotiatedRatePlanID: &negotiatedRatePlan.ID,
		CompanyName:          "Test Company Ltd",
		ContactEmail:         "contact@testcompany.com",
		ContactPhone:         "123-456-7890",
		BillingAddress:       "123 Test St, Test City",
		CompanyNotes:         "Test notes",
		HasCreditFacility:    true,
	}

	insertQuery := `INSERT INTO identity.company_profiles 
		(property_id, tax_id, negotiated_rate_plan_id, company_name, contact_email, 
		contact_phone, billing_address, company_notes, has_credit_facility)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	t.Run("Unique Constraint Tests", func(t *testing.T) {
		t.Parallel()

		t.Run("TC-CPROF-02 - Company name must be unique per property", func(t *testing.T) {
			t.Parallel()
			// First insert should succeed
			_, err := testDB.Exec(ctx, insertQuery,
				params.PropertyID,
				params.TaxID,
				params.NegotiatedRatePlanID,
				params.CompanyName,
				params.ContactEmail,
				params.ContactPhone,
				params.BillingAddress,
				params.CompanyNotes,
				params.HasCreditFacility,
			)
			assert.NoError(t, err, "Failed to insert first company profile: %v", err)
			// Second insert with same company name and property should fail
			_, err = testDB.Exec(ctx, insertQuery,
				params.PropertyID,
				"VAT654321", // Different tax ID
				params.NegotiatedRatePlanID,
				params.CompanyName, // Same company name
				"contact2@testcompany.com",
				"987-654-3210",
				"456 Test St, Test City",
				"Test notes 2",
				false,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error for company name, got: %v", err)
		})

		t.Run("TC-CPROF-16 - Tax ID must be unique per property", func(t *testing.T) {
			t.Parallel()
			// First insert should succeed
			_, err := testDB.Exec(ctx, insertQuery,
				params.PropertyID,
				"VAT999999",
				params.NegotiatedRatePlanID,
				"Another Company Ltd",
				"contact@anothercompany.com",
				"555-123-4567",
				"789 Test St, Test City",
				"Test notes 3",
				true,
			)
			assert.NoError(t, err, "Failed to insert first company profile: %v", err)
			// Second insert with same tax ID and property should fail
			_, err = testDB.Exec(ctx, insertQuery,
				params.PropertyID,
				"VAT999999", // Same tax ID
				params.NegotiatedRatePlanID,
				"Different Company Ltd",
				"contact@differentcompany.com",
				"111-222-3333",
				"101 Test St, Test City",
				"Test notes 4",
				false,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error for tax ID, got: %v", err)
		})
	})

	t.Run("FK Existence Tests", func(t *testing.T) {
		t.Parallel()

		paramsTwo := TestCompanyProfile{
			PropertyID:           property.ID,
			TaxID:                "VAT777777",
			NegotiatedRatePlanID: &negotiatedRatePlan.ID,
			CompanyName:          "FK Test Company Ltd",
			ContactEmail:         "contact@testcompany.com",
			ContactPhone:         "123-456-7890",
			BillingAddress:       "123 Test St, Test City",
			CompanyNotes:         "Test notes 5",
			HasCreditFacility:    true,
		}

		// TC-CPROF-04 - The company profile's property must exist
		// TC-CPROF-05 - The negotiated rate plan ID must exist (if not null)
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:       "TC-CPROF-04 - Property must exist",
				FakeIDIdx:  0,
				RealID:     property.ID,
				BaseParams: hf.StructToSlice(params),
			},
			{
				Name:       "TC-CPROF-05 - Negotiated rate plan must exist",
				FakeIDIdx:  2,
				RealID:     *paramsTwo.NegotiatedRatePlanID,
				BaseParams: hf.StructToSlice(paramsTwo),
			},
		})

		t.Run("TC-CPROF-05b - Negotiated rate plan can be null", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a company profile with null negotiated rate plan
			_, err := testDB.Exec(ctx, insertQuery,
				params.PropertyID,
				"VAT777778",
				nil, // Null negotiated rate plan
				"Null Rate Plan Co",
				"contact@nullrateplan.com",
				"444-555-6666",
				"222 Test St, Test City",
				"Test notes 5",
				true,
			)
			assert.NoError(t, err, "Failed to insert company profile with null negotiated rate plan: %v", err)
		})
	})

	t.Run("TC-CPROF-07 - Company name is required", func(t *testing.T) {
		t.Parallel()
		// Attempt to create a company profile with empty company name
		_, err := testDB.Exec(ctx, insertQuery,
			params.PropertyID,
			"VAT777779",
			params.NegotiatedRatePlanID,
			"", // Empty company name
			params.ContactEmail,
			params.ContactPhone,
			params.BillingAddress,
			params.CompanyNotes,
			params.HasCreditFacility,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for company name, got: %v", err)
	})

	t.Run("Char Limit Tests", func(t *testing.T) {
		t.Parallel()

		cases := []hf.ConstraintTest{
			{
				Name:        "TC-CPROF-06 - Company name must not exceed 50 characters",
				Field:       "company_name",
				Value:       strings.Repeat("a", 51), // 51 characters
				FieldIndex:  3,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-CPROF-11 - Billing address must not exceed 300 characters",
				Field:       "billing_address",
				Value:       strings.Repeat("a", 301), // 301 characters
				FieldIndex:  6,
				ExpectedErr: hf.CheckViolationCode,
			},
			{
				Name:        "TC-CPROF-12 - Company notes must not exceed 1500 characters",
				Field:       "company_notes",
				Value:       strings.Repeat("a", 1501), // 1501 characters
				FieldIndex:  7,
				ExpectedErr: hf.CheckViolationCode,
			},
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(params), cases)
	})

	t.Run("TC-CPROF-15 - The negotiated rate plan must belong to the same property", func(t *testing.T) {
		t.Parallel()
		// Create another property
		anotherProperty := GenerateTestProperty(t, ctx)
		// Create a rate plan for the other property
		otherRatePlan := GenerateTestRatePlan(t, ctx, anotherProperty.ID)

		// Attempt to create a company profile with a negotiated rate plan from a different property
		_, err := testDB.Exec(ctx, insertQuery,
			params.PropertyID,
			"VAT888888",
			&otherRatePlan.ID, // Rate plan from another property
			"Cross Property Co",
			"contact@crossproperty.com",
			"777-888-9999",
			"333 Test St, Test City",
			"Test notes 6",
			true,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error for negotiated rate plan from different property, got: %v", err)
	})
}
