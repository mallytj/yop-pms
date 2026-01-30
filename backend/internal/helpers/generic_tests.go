package helpers

import (
	"context"
	"testing"

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
		t.Run(tc.Name, func(t *testing.T) {
			// Clone params so we don't mutate the base for other parallel tests
			testParams := make([]interface{}, len(baseParams))
			copy(testParams, baseParams)

			// Inject the "bad" value into the specific column we are testing
			testParams[tc.FieldIndex] = tc.Value

			_, err := db.Exec(ctx, query, testParams...)

			// hf.CheckErrorCode is your custom helper
			assert.True(t, CheckErrorCode(err, tc.ExpectedErr),
				"Expected error code %s for field %s, but got: %v", tc.ExpectedErr, tc.Field, err)
		})
	}
}
