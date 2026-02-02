package helpers

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

type ConstraintTest struct {
	Name        string
	Field       string
	Value       interface{}
	ExpectedErr string // e.g., hf.CheckViolationCode
	FieldIndex  int    // Index of the field in the parameter list
}

func RunConstraintTests(t *testing.T, ctx context.Context, db *pgxpool.Pool, query string, baseParams []interface{}, cases []ConstraintTest) {
	for _, tc := range cases {
		tc := tc // capture range variable
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			testParams := make([]interface{}, len(baseParams))
			copy(testParams, baseParams)

			testParams[tc.FieldIndex] = tc.Value

			_, err := db.Exec(ctx, query, testParams...)

			assert.True(t, CheckErrorCode(err, tc.ExpectedErr),
				"Expected error code %s for field %s, but got: %v", tc.ExpectedErr, tc.Field, err)
		})
	}
}

// FKExistenceTest describes a single foreign-key existence check.
// FakeID is the index in baseParams that holds the FK column to override.
// RealID is the valid parent ID that should be substituted for the "pass" case.
type FKExistenceTest struct {
	Name       string
	FakeIDIdx  int       // index in baseParams to replace with a non-existent UUID
	RealID     uuid.UUID // valid parent ID for the success case
	BaseParams []interface{}
}

// RunFKExistenceTests runs the standard two-part FK check for each case:
//  1. Insert with a random (non-existent) UUID at FakeIDIdx → expects ForeignKeyViolationCode.
//  2. Insert with RealID at the same index → expects success.
func RunFKExistenceTests(t *testing.T, ctx context.Context, db *pgxpool.Pool, query string, cases []FKExistenceTest) {
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			t.Run("Fail when parent does not exist", func(t *testing.T) {
				// Can not be ran in parallel, due to unique constraints

				params := make([]interface{}, len(tc.BaseParams))
				copy(params, tc.BaseParams)
				params[tc.FakeIDIdx] = uuid.New()

				_, err := db.Exec(ctx, query, params...)
				assert.True(t, CheckErrorCode(err, ForeignKeyViolationCode),
					"Expected foreign key violation error, got: %v", err)
			})

			t.Run("Pass when parent exists", func(t *testing.T) {
				// Can not be ran in parallel, due to unique constraints

				params := make([]interface{}, len(tc.BaseParams))
				copy(params, tc.BaseParams)
				params[tc.FakeIDIdx] = tc.RealID

				_, err := db.Exec(ctx, query, params...)
				assert.NoError(t, err, "Expected no error with valid parent ID, got: %v", err)
			})
		})
	}
}
