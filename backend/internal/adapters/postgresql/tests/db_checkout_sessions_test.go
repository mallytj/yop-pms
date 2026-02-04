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

func TestDbCheckoutSessions(t *testing.T) {
	ctx := context.Background()

	property := GenerateTestProperty(t, ctx)
	reservation := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)
	reservation2 := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)

	insertQuery := `INSERT INTO operations.checkout_sessions
		(property_id, reservation_id, payment_intent_id, expires_at, status, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6)`

	t.Run("FK Existence Tests", func(t *testing.T) {
		hf.RunFKExistenceTests(t, ctx, testDB, insertQuery, []hf.FKExistenceTest{
			{
				Name:      "TC-COSES-02 - Property must exist",
				FakeIDIdx: 0,
				RealID:    property.ID,
				BaseParams: []interface{}{
					property.ID,
					reservation.ID,
					"pi_" + uuid.New().String()[:24],
					time.Now().Add(15 * time.Minute),
					"pending",
					nil,
				},
			},
			{
				Name:      "TC-COSES-03 - Reservation must exist",
				FakeIDIdx: 1,
				RealID:    reservation2.ID,
				BaseParams: []interface{}{
					property.ID,
					reservation2.ID,
					"pi_" + uuid.New().String()[:24],
					time.Now().Add(15 * time.Minute),
					"pending",
					nil,
				},
			},
		})
	})

	t.Run("Constraint Tests", func(t *testing.T) {
		constraintRes := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)
		baseParams := []interface{}{
			property.ID,
			constraintRes.ID,
			"pi_" + uuid.New().String()[:24],
			time.Now().Add(15 * time.Minute),
			"pending",
			nil,
		}

		hf.RunConstraintTests(t, ctx, testDB, insertQuery, baseParams, []hf.ConstraintTest{
			{
				Name:        "TC-COSES-07 - Status must be a valid enum",
				Field:       "status",
				Value:       "invalid_status",
				ExpectedErr: hf.InvalidTextRepresentationCode,
				FieldIndex:  4,
			},
		})
	})

	t.Run("TC-COSES-05 - Expiration time defaults to Now + 15 mins", func(t *testing.T) {
		t.Parallel()

		defaultRes := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)

		var expiresAt time.Time
		err := testDB.QueryRow(ctx,
			`INSERT INTO operations.checkout_sessions (property_id, reservation_id, payment_intent_id)
				VALUES ($1, $2, $3)
				RETURNING expires_at`,
			property.ID,
			defaultRes.ID,
			"pi_"+uuid.New().String()[:24],
		).Scan(&expiresAt)

		assert.NoError(t, err)

		// Check that expires_at is approximately 15 minutes from now (within 1 minute tolerance)
		expectedExpiry := time.Now().Add(15 * time.Minute)
		diff := expiresAt.Sub(expectedExpiry)
		assert.True(t, diff < time.Minute && diff > -time.Minute,
			"expires_at should default to ~15 minutes from now, got diff: %v", diff)
	})

	t.Run("TC-COSES-06 - Idempotency key must be unique if set", func(t *testing.T) {
		t.Parallel()

		res1 := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)
		res2 := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)

		idempotencyKey := "idem_" + uuid.New().String()

		// First insert with idempotency key
		_, err := testDB.Exec(ctx, insertQuery,
			property.ID,
			res1.ID,
			"pi_"+uuid.New().String()[:24],
			time.Now().Add(15*time.Minute),
			"pending",
			idempotencyKey,
		)
		assert.NoError(t, err)

		// Second insert with same idempotency key should fail
		_, err = testDB.Exec(ctx, insertQuery,
			property.ID,
			res2.ID,
			"pi_"+uuid.New().String()[:24],
			time.Now().Add(15*time.Minute),
			"pending",
			idempotencyKey,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode),
			"Expected unique violation for duplicate idempotency_key, got: %v", err)
	})

	t.Run("TC-COSES-unique - Property and reservation must be unique", func(t *testing.T) {
		t.Parallel()

		uniqueRes := GenerateTestReservation(t, ctx, property.ID, uuid.Nil)

		// First insert
		_, err := testDB.Exec(ctx, insertQuery,
			property.ID,
			uniqueRes.ID,
			"pi_"+uuid.New().String()[:24],
			time.Now().Add(15*time.Minute),
			"pending",
			nil,
		)
		assert.NoError(t, err)

		// Second insert with same property and reservation should fail
		_, err = testDB.Exec(ctx, insertQuery,
			property.ID,
			uniqueRes.ID,
			"pi_"+uuid.New().String()[:24],
			time.Now().Add(15*time.Minute),
			"pending",
			nil,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode),
			"Expected unique violation for duplicate property_id + reservation_id, got: %v", err)
	})
}
