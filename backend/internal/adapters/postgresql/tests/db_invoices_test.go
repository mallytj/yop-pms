//go:build ignore

package db_tests

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	hf "ollerod-pms/internal/helpers"

	"github.com/stretchr/testify/assert"
)

func TestDbInvoices(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	folio := GenerateTestFolio(t, ctx, property.ID)

	insertQuery := `INSERT INTO finance.invoices
		(property_id, folio_id, property_code, fiscal_year, fiscal_sequential, billing_address, is_pro_forma, issue_date, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-INVO-02 - Property must exist",
				FakeIDIdx: 0,
				RealID:    property.ID,
				BaseParams: []interface{}{
					property.ID,
					nil,
					"RES",
					2024,
					generateUniqueInvoiceSeq(),
					"123 Test St",
					false,
					time.Now(),
					time.Now().AddDate(0, 0, 30),
				},
			},
			{
				Name:      "TC-INVO-03 - Folio must exist if set",
				FakeIDIdx: 1,
				RealID:    folio.ID,
				BaseParams: []interface{}{
					property.ID,
					folio.ID,
					"RES",
					2024,
					generateUniqueInvoiceSeq(),
					"123 Test St",
					false,
					time.Now(),
					time.Now().AddDate(0, 0, 30),
				},
			},
		})
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		baseParams := []interface{}{
			property.ID,
			nil,
			"RES",
			2024,
			generateUniqueInvoiceSeq(),
			"123 Test St",
			false,
			time.Now(),
			time.Now().AddDate(0, 0, 30),
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, baseParams, []hf.ConstraintTest{
			{
				Name:        "TC-INVO-04a - Property code too short (2 chars)",
				Field:       "property_code",
				Value:       "AB",
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  2,
			},
			{
				Name:        "TC-INVO-04b - Property code too long (5 chars)",
				Field:       "property_code",
				Value:       "ABCDE",
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  2,
			},
			{
				Name:        "TC-INVO-12 - Due date before issue date",
				Field:       "due_date",
				Value:       time.Now().AddDate(0, 0, -1),
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  8,
			},
		})
	})

	t.Run("TC-INVO-04 - Valid property codes", func(t *testing.T) {
		t.Parallel()

		validCodes := []string{"RES", "ABCD"}
		for _, code := range validCodes {
			code := code
			t.Run("Valid code: "+code, func(t *testing.T) {
				t.Parallel()

				_, err := testDB.Exec(ctx, insertQuery,
					property.ID,
					nil,
					code,
					2024,
					generateUniqueInvoiceSeq(),
					"123 Test St",
					false,
					time.Now(),
					time.Now().AddDate(0, 0, 30),
				)
				assert.NoError(t, err, "Expected no error for valid property_code: %s", code)
			})
		}
	})

	t.Run("TC-INVO-05 - The fiscal year is required", func(t *testing.T) {
		t.Parallel()

		_, err := testDB.Exec(ctx,
			`INSERT INTO finance.invoices (property_id, property_code, fiscal_sequential, billing_address)
				VALUES ($1, $2, $3, $4)`,
			property.ID,
			"RES",
			generateUniqueInvoiceSeq(),
			"123 Test St",
		)
		assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode),
			"Expected not null violation for missing fiscal_year, got: %v", err)
	})

	t.Run("TC-INVO-07 - Fiscal sequential must be unique by year by property", func(t *testing.T) {
		t.Parallel()

		fiscalSeq := generateUniqueInvoiceSeq()

		// First insert should succeed
		_, err := testDB.Exec(ctx, insertQuery,
			property.ID,
			nil,
			"RES",
			2024,
			fiscalSeq,
			"123 Test St",
			false,
			time.Now(),
			time.Now().AddDate(0, 0, 30),
		)
		assert.NoError(t, err)

		// Second insert with same property, year, sequential should fail
		_, err = testDB.Exec(ctx, insertQuery,
			property.ID,
			nil,
			"RES",
			2024,
			fiscalSeq,
			"456 Other St",
			false,
			time.Now(),
			time.Now().AddDate(0, 0, 30),
		)
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode),
			"Expected unique violation for duplicate fiscal_sequential per property per year, got: %v", err)
	})

	t.Run("TC-INVO-08 - Invoice number must be generated correctly", func(t *testing.T) {
		t.Parallel()

		fiscalSeq := generateUniqueInvoiceSeq()

		var invoiceNumber string
		err := testDB.QueryRow(ctx,
			`INSERT INTO finance.invoices
				(property_id, property_code, fiscal_year, fiscal_sequential, billing_address)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING invoice_number`,
			property.ID,
			"RES",
			2024,
			fiscalSeq,
			"123 Test St",
		).Scan(&invoiceNumber)

		assert.NoError(t, err)

		expectedFormat := fmt.Sprintf("RES-2024-%06d", fiscalSeq)
		assert.Equal(t, expectedFormat, invoiceNumber,
			"Invoice number should be PROPERTY_CODE-FISCAL_YEAR-FISCAL_SEQUENTIAL")
	})

	t.Run("TC-INVO-09 - Billing address is required", func(t *testing.T) {
		t.Parallel()

		_, err := testDB.Exec(ctx,
			`INSERT INTO finance.invoices (property_id, property_code, fiscal_year, fiscal_sequential)
				VALUES ($1, $2, $3, $4)`,
			property.ID,
			"RES",
			2024,
			generateUniqueInvoiceSeq(),
		)
		assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode),
			"Expected not null violation for missing billing_address, got: %v", err)
	})
}

var invoiceSeqCounter int = 200000

func generateUniqueInvoiceSeq() int {
	invoiceSeqCounter++
	return invoiceSeqCounter + rand.Intn(1000)
}
