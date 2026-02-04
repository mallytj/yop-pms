//go:build ignore

package db_tests

import (
	"context"
	"testing"
	"time"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbSalesLedgerTransactions(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	slAccount := GenerateTestSLAccount(t, ctx, property.ID, uuid.Nil)
	invoice := GenerateTestInvoice(t, ctx, property.ID)
	user := GenerateTestUser(t, ctx)

	// Additional account for FK tests
	slAccount2 := GenerateTestSLAccount(t, ctx, property.ID, uuid.Nil)
	slAccount3 := GenerateTestSLAccount(t, ctx, property.ID, uuid.Nil)

	insertQuery := `INSERT INTO sales_ledgers.transactions
		(ledger_account_id, source_invoice_id, amount_pence, due_date, posted_by_user_id, type)
		VALUES ($1, $2, $3, $4, $5, $6)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-SLTX-02 - Ledger account must exist",
				FakeIDIdx: 0,
				RealID:    slAccount.ID,
				BaseParams: []interface{}{
					slAccount.ID,
					nil,
					10000,
					time.Now().AddDate(0, 0, 30),
					user.ID,
					"charge",
				},
			},
			{
				Name:      "TC-SLTX-03 - Source invoice must exist if set",
				FakeIDIdx: 1,
				RealID:    invoice.ID,
				BaseParams: []interface{}{
					slAccount2.ID,
					invoice.ID,
					10000,
					time.Now().AddDate(0, 0, 30),
					user.ID,
					"charge",
				},
			},
			{
				Name:      "TC-SLTX-08 - Posted by user must exist",
				FakeIDIdx: 4,
				RealID:    user.ID,
				BaseParams: []interface{}{
					slAccount3.ID,
					nil,
					10000,
					time.Now().AddDate(0, 0, 30),
					user.ID,
					"charge",
				},
			},
		})
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		constraintAccount := GenerateTestSLAccount(t, ctx, property.ID, uuid.Nil)
		baseParams := []interface{}{
			constraintAccount.ID,
			nil,
			10000,
			time.Now().AddDate(0, 0, 30),
			user.ID,
			"charge",
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, baseParams, []hf.ConstraintTest{
			{
				Name:        "TC-SLTX-09 - Type must be a valid enum",
				Field:       "type",
				Value:       "invalid_type",
				ExpectedErr: hf.InvalidTextRepresentationCode,
				FieldIndex:  5,
			},
		})
	})

	t.Run("TC-SLTX-04 - Amount in pence is required", func(t *testing.T) {
		t.Parallel()

		reqAccount := GenerateTestSLAccount(t, ctx, property.ID, uuid.Nil)

		_, err := testDB.Exec(ctx,
			`INSERT INTO sales_ledgers.transactions
				(ledger_account_id, posted_by_user_id, type)
				VALUES ($1, $2, $3)`,
			reqAccount.ID,
			user.ID,
			"charge",
		)
		assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode),
			"Expected not null violation for missing amount_pence, got: %v", err)
	})

	t.Run("TC-SLTX-07 - is_fully_paid generated column", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name         string
			amountPence  int
			expectedPaid bool
		}{
			{"Positive amount - not paid", 10000, false},
			{"Zero amount - paid", 0, true},
			{"Negative amount - paid", -5000, true},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				account := GenerateTestSLAccount(t, ctx, property.ID, uuid.Nil)

				var isFullyPaid bool
				err := testDB.QueryRow(ctx,
					`INSERT INTO sales_ledgers.transactions
						(ledger_account_id, amount_pence, posted_by_user_id, type)
						VALUES ($1, $2, $3, $4)
						RETURNING is_fully_paid`,
					account.ID,
					tc.amountPence,
					user.ID,
					"charge",
				).Scan(&isFullyPaid)

				assert.NoError(t, err)
				assert.Equal(t, tc.expectedPaid, isFullyPaid,
					"is_fully_paid should be %v for amount %d", tc.expectedPaid, tc.amountPence)
			})
		}
	})
}
