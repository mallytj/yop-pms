//go:build ignore
package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbAuditLogs(t *testing.T) {
	ctx := context.Background()
	validChanges := `{"field": "id", "old_value": "old", "new_value": "new"}`

	t.Run("TC-AUDIT-02 - The audit log's user must exist", func(t *testing.T) {
		t.Parallel()
		// Fill params
		params := TestAuditLog{
			UserID:   uuid.New(), // Non-existent user
			Action:   "create",
			Entity:   "guest",
			EntityID: uuid.New(),
			Changes:  validChanges,
		}

		// Create query
		query := `INSERT INTO auth.audit_logs (user_id, action, entity, entity_id, changes) VALUES ($1, $2, $3, $4, $5)`

		// Attempt to create an audit log with a non-existent user
		_, err := testDB.Exec(ctx, query, params.UserID, params.Action, params.Entity, params.EntityID, params.Changes)

		// Check for foreign key violation error
		assert.True(t, hf.CheckErrorCode(err, hf.ForeignKeyViolationCode), "Expected foreign key violation error, got: %v", err)
	})

	t.Run("TC-AUDIT-03 - The entity must be one of the allowed values", func(t *testing.T) {
		t.Parallel()
		// First, create a test user
		user := GenerateTestUser(t, ctx)

		// Fill params
		params := TestAuditLog{
			UserID:   user.ID,
			Action:   "create",
			Entity:   "invalid_entity", // Invalid entity
			EntityID: uuid.New(),
			Changes:  validChanges,
		}

		// Create query
		query := `INSERT INTO auth.audit_logs (user_id, action, entity, entity_id, changes) VALUES ($1, $2, $3, $4, $5)`

		// Attempt to create an audit log with an invalid entity
		_, err := testDB.Exec(ctx, query, params.UserID, params.Action, params.Entity, params.EntityID, params.Changes)

		// Check for check violation error
		assert.True(t, hf.CheckErrorCode(err, hf.InvalidTextRepresentationCode), "Expected check violation error, got: %v", err)
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		// First, create a test user
		user := GenerateTestUser(t, ctx)

		params := TestAuditLog{
			UserID:   user.ID,
			Action:   "create",
			Entity:   "guest",
			EntityID: uuid.Nil,
			Changes:  validChanges,
		}

		// Build constraint tests
		tests := []hf.ConstraintTest{
			{
				Name:        "TC-AUDIT-04 - The action is required",
				Field:       "action",
				Value:       nil,
				ExpectedErr: hf.NotNullViolationCode,
				FieldIndex:  1,
			},
			{
				Name:        "TC-AUDIT-05 - The entity is required",
				Field:       "entity",
				Value:       nil,
				ExpectedErr: hf.NotNullViolationCode,
				FieldIndex:  2,
			},
			{
				Name:        "TC-AUDIT-13 - The entity ID is required",
				Field:       "entity_id",
				Value:       nil,
				ExpectedErr: hf.NotNullViolationCode,
				FieldIndex:  3,
			},
		}

		query := `INSERT INTO auth.audit_logs (user_id, action, entity, entity_id, changes) VALUES ($1, $2, $3, $4, $5)`

		hf.RunConstraintTests(t, ctx, testDB, query, hf.StructToSlice(params), tests)
	})
}
