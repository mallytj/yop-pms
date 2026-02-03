package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbSalesLedgerAccounts(t *testing.T) {
	ctx := context.Background()

	// Create a property
	property := GenerateTestProperty(t, ctx)

	// Create a company profile
	companyProfile := GenerateTestCompanyProfile(t, ctx, property.ID)

	insertQuery := `INSERT INTO sales_ledgers.accounts (property_id, company_profile_id, name, code, credit_limit_pence, payment_terms_days) 
					VALUES ($1, $2, $3, $4, $5, $6)`

	baseParams := TestSLAccount{
		PropertyID:       property.ID,
		CompanyProfileID: companyProfile.ID,
		Name:             "Test Sales Ledger Account",
		Code:             "TSLA01",
		CreditLimitPence: 1000,
		PaymentTermDays:  30,
	}

	// Prepare altered params for FK tests
	// Create a new company profile for the altered params
	// Hacky, but who cares in tests!
	anotherCompanyProfile := GenerateTestCompanyProfile(t, ctx, property.ID)
	alteredParams := baseParams
	alteredParams.Name = "Altered Name"
	alteredParams.Code = "ALT01"
	alteredParams.CompanyProfileID = anotherCompanyProfile.ID

	t.Run("FK Existence Tests", func(t *testing.T) {
		t.Parallel()

		foreignKeyExistenceTests := []hf.FKExistenceTest{
			{
				Name:       "TC-SLACC-02 - Property must exist",
				FakeIDIdx:  0,
				RealID:     property.ID,
				BaseParams: hf.StructToSlice(baseParams),
			},
			{
				Name:       "TC-SLACC-03 - Company profile must exist if set",
				FakeIDIdx:  1,
				RealID:     anotherCompanyProfile.ID,
				BaseParams: hf.StructToSlice(alteredParams),
			},
		}

		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, foreignKeyExistenceTests)
	})

	constraintTests := []hf.ConstraintTest{
		{
			Name:        "TC-SLACC-03 - Account property is required",
			Field:       "",
			Value:       nil,
			FieldIndex:  0,
			ExpectedErr: hf.NotNullViolationCode,
		},
		{
			Name:        "TC-SLACC-04 - Credit limit must be a positive integer",
			Field:       "credit_limit",
			Value:       -100,
			FieldIndex:  4,
			ExpectedErr: hf.CheckViolationCode,
		},
		{
			Name:        "TC-SLACC-05 - Payment terms must be a positive integer",
			Field:       "payment_terms_days",
			Value:       -30,
			FieldIndex:  5,
			ExpectedErr: hf.CheckViolationCode,
		},
		{
			Name:        "TC-SLACC-09 - Name is required",
			Field:       "name",
			Value:       nil,
			FieldIndex:  2,
			ExpectedErr: hf.NotNullViolationCode,
		},
		{
			Name:        "TC-SLACC-10 - Name must not exceed 100 characters",
			Field:       "name",
			Value:       strings.Repeat("A", 101),
			FieldIndex:  2,
			ExpectedErr: hf.CheckViolationCode,
		},
		{
			Name:        "TC-SLACC-11 - Code is required",
			Field:       "code",
			Value:       nil,
			FieldIndex:  3,
			ExpectedErr: hf.NotNullViolationCode,
		},
		{
			Name:        "TC-SLACC-12 - Code must not exceed 10 characters",
			Field:       "code",
			Value:       strings.Repeat("A", 11),
			FieldIndex:  3,
			ExpectedErr: hf.CheckViolationCode,
		},
	}

	hf.RunConstraintTests(t, ctx, testDB, insertQuery, hf.StructToSlice(baseParams), constraintTests)

	t.Run("Unique Constraints", func(t *testing.T) {
		t.Parallel()

		defer func() {
			// Clean up any inserted test data
			testDB.Exec(ctx, `DELETE FROM sales_ledgers.accounts WHERE name = $1 OR code = $2`, baseParams.Name, baseParams.Code)
		}()

		// First, create a sales ledger account
		_, err := testDB.Exec(ctx, insertQuery, baseParams.PropertyID, baseParams.CompanyProfileID, baseParams.Name, baseParams.Code, baseParams.CreditLimitPence, baseParams.PaymentTermDays)
		assert.NoError(t, err, "Failed to create initial sales ledger account: %v", err)

		t.Run("TC-SLACC-14 - Name must be unique per property", func(t *testing.T) {
			t.Parallel()

			// Attempt to create another sales ledger account with the same name for the same property
			_, err = testDB.Exec(ctx, insertQuery, baseParams.PropertyID, baseParams.CompanyProfileID, baseParams.Name, "UNIQ01", baseParams.CreditLimitPence, baseParams.PaymentTermDays)

			// Check for unique violation error
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)
		})

		t.Run("TC-SLACC-15 - Code must be unique per property", func(t *testing.T) {
			t.Parallel()
			// Attempt to create another sales ledger account with the same code for the same property
			_, err = testDB.Exec(ctx, insertQuery, baseParams.PropertyID, baseParams.CompanyProfileID, "Another Name", baseParams.Code, baseParams.CreditLimitPence, baseParams.PaymentTermDays)

			// Check for unique violation error
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)
		})

		t.Run("TC-SLACC_06 - The company profile must be unique per account per property", func(t *testing.T) {
			t.Parallel()
			// Attempt to create another sales ledger account with the same company profile for the same property
			_, err = testDB.Exec(ctx, insertQuery, baseParams.PropertyID, baseParams.CompanyProfileID, "Different Name", "DIFF01", baseParams.CreditLimitPence, baseParams.PaymentTermDays)

			// Check for unique violation error
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)
		})
	})

	t.Run("TC-SLACC-13 - Company profile must belong to the same property", func(t *testing.T) {
		t.Parallel()
		// Create another property
		anotherProperty := GenerateTestProperty(t, ctx)

		// Create a company profile for the other property
		otherCompanyProfile := GenerateTestCompanyProfile(t, ctx, anotherProperty.ID)

		// Attempt to create a sales ledger account with a company profile from another property
		_, err := testDB.Exec(ctx, insertQuery, baseParams.PropertyID, otherCompanyProfile.ID, "RANFSJNKJG", "ALTER01", baseParams.CreditLimitPence, baseParams.PaymentTermDays)

		// Check for foreign key violation error
		assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error, got: %v", err)

		t.Run("Pass when company profile belongs to the same property", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a sales ledger account with the correct company profile
			_, err := testDB.Exec(ctx, insertQuery, anotherProperty.ID, otherCompanyProfile.ID, "XYZFSYHG", "ALTER02", baseParams.CreditLimitPence, baseParams.PaymentTermDays)

			assert.NoError(t, err)

			defer func() {
				// Clean up any inserted test data
				testDB.Exec(ctx, `DELETE FROM sales_ledgers.accounts WHERE name IN ($1, $2)`, "RANFSJNKJG", "XYZFSYHG")
			}()
		})
	})
}
