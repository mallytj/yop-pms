//go:build ignore
package db_tests

import (
	"context"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbFolios(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	reservation := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)
	slAccount := GenerateTestSLAccount(t, ctx, property.ID, uuid.Nil)

	insertQuery := `INSERT INTO finance.folios
		(property_id, reservation_id, sales_ledger_id, folio_part, balance_pence)
		VALUES ($1, $2, $3, $4, $5)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-FOLIO-02 - Property must exist",
				FakeIDIdx: 0,
				RealID:    property.ID,
				BaseParams: []interface{}{
					property.ID,
					nil,
					nil,
					"A",
					0,
				},
			},
			{
				Name:      "TC-FOLIO-03 - Reservation must exist if set",
				FakeIDIdx: 1,
				RealID:    reservation.ID,
				BaseParams: []interface{}{
					property.ID,
					reservation.ID,
					nil,
					"B",
					0,
				},
			},
			{
				Name:      "TC-FOLIO-04 - Sales ledger must exist if set",
				FakeIDIdx: 2,
				RealID:    slAccount.ID,
				BaseParams: []interface{}{
					property.ID,
					nil,
					slAccount.ID,
					"C",
					0,
				},
			},
		})
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		baseParams := []interface{}{
			property.ID,
			nil,
			nil,
			"A",
			0,
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, baseParams, []hf.ConstraintTest{
			{
				Name:        "TC-FOLIO-05 - Folio part must be a valid enum",
				Field:       "folio_part",
				Value:       "D", // Invalid enum value
				ExpectedErr: hf.InvalidTextRepresentationCode,
				FieldIndex:  3,
			},
		})
	})

	t.Run("TC-FOLIO-06 - The folio part is required", func(t *testing.T) {
		t.Parallel()

		_, err := testDB.Exec(ctx,
			`INSERT INTO finance.folios (property_id, balance_pence) VALUES ($1, $2)`,
			property.ID,
			0,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode),
			"Expected not null violation for missing folio_part, got: %v", err)
	})
}
