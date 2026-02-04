//go:build ignore
package db_tests

import (
	"context"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbFolioTransactions(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	folio := GenerateTestFolio(t, ctx, property.ID)
	ledgerCode := GenerateTestLedgerCode(t, ctx, property.ID, uuid.Nil)
	taxRule := GenerateTestTaxRule(t, ctx, property.ID)
	user := GenerateTestUser(t, ctx)

	// Additional folios for FK tests to avoid unique conflicts
	folio2 := GenerateTestFolio(t, ctx, property.ID)
	folio3 := GenerateTestFolio(t, ctx, property.ID)
	folio4 := GenerateTestFolio(t, ctx, property.ID)

	insertQuery := `INSERT INTO finance.folio_transactions
		(folio_id, ledger_code_id, description, net_unit_price_pence, quantity, tax_rule_id, tax_rate_snapshot, posted_by_user_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-FOLTX-02 - Folio must exist",
				FakeIDIdx: 0,
				RealID:    folio.ID,
				BaseParams: []interface{}{
					folio.ID,
					nil,
					"Test transaction",
					5000,
					1,
					nil,
					10.00,
					nil,
					"pending",
				},
			},
			{
				Name:      "TC-FOLTX-03 - Ledger code must exist if set",
				FakeIDIdx: 1,
				RealID:    ledgerCode.ID,
				BaseParams: []interface{}{
					folio2.ID,
					ledgerCode.ID,
					"Test transaction",
					5000,
					1,
					nil,
					10.00,
					nil,
					"pending",
				},
			},
			{
				Name:      "TC-FOLTX-08 - Tax rule must exist if set",
				FakeIDIdx: 5,
				RealID:    taxRule.ID,
				BaseParams: []interface{}{
					folio3.ID,
					nil,
					"Test transaction",
					5000,
					1,
					taxRule.ID,
					10.00,
					nil,
					"pending",
				},
			},
			{
				Name:      "TC-FOLTX-15 - Posted by user must exist if set",
				FakeIDIdx: 7,
				RealID:    user.ID,
				BaseParams: []interface{}{
					folio4.ID,
					nil,
					"Test transaction",
					5000,
					1,
					nil,
					10.00,
					user.ID,
					"posted",
				},
			},
		})
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		constraintFolio := GenerateTestFolio(t, ctx, property.ID)
		baseParams := []interface{}{
			constraintFolio.ID,
			nil,
			"Test transaction",
			5000,
			1,
			nil,
			10.00,
			nil,
			"pending",
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, baseParams, []hf.ConstraintTest{
			{
				Name:        "TC-FOLTX-07 - Quantity must be greater than 0",
				Field:       "quantity",
				Value:       0,
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  4,
			},
			{
				Name:        "TC-FOLTX-16 - Status must be a valid enum",
				Field:       "status",
				Value:       "invalid_status",
				ExpectedErr: hf.InvalidTextRepresentationCode,
				FieldIndex:  8,
			},
		})
	})

	t.Run("TC-FOLTX-09 to TC-FOLTX-13 - Generated columns", func(t *testing.T) {
		t.Parallel()

		genFolio := GenerateTestFolio(t, ctx, property.ID)

		var totalNet, taxAmount, grossAmount int
		err := testDB.QueryRow(ctx,
			`INSERT INTO finance.folio_transactions
				(folio_id, description, net_unit_price_pence, quantity, tax_rate_snapshot, status)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING total_net_price_pence, tax_amount_pence, gross_amount_pence`,
			genFolio.ID,
			"Test generated columns",
			5000,  // £50.00 per unit
			2,     // 2 units
			20.00, // 20% tax
			"pending",
		).Scan(&totalNet, &taxAmount, &grossAmount)

		assert.NoError(t, err)

		// TC-FOLTX-09: total_net = net_unit * quantity = 5000 * 2 = 10000
		assert.Equal(t, 10000, totalNet, "TC-FOLTX-09: Total net price should be net_unit * quantity")

		// TC-FOLTX-12: tax_amount = total_net * tax_rate / 100 = 10000 * 20 / 100 = 2000
		assert.Equal(t, 2000, taxAmount, "TC-FOLTX-12: Tax amount should be total_net * tax_rate / 100")

		// TC-FOLTX-13: gross_amount = total_net + tax_amount = 10000 + 2000 = 12000
		assert.Equal(t, 12000, grossAmount, "TC-FOLTX-13: Gross amount should be total_net + tax_amount")
	})
}
