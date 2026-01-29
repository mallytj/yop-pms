package db_tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"github.com/stretchr/testify/assert"
)

func TestDbLicences(t *testing.T) {
	// TC-LICE-01 in db_migration_test.go
	ctx := context.Background()

	t.Run("TC-LICE-02 - There must be only one licence key of the same value", func(t *testing.T) {
		t.Parallel()
		// Fill params
		licenceKey := "YOP-12345"
		organisationName := "Test"
		contactEmail := "test@test.com"

		// Create query
		query := `INSERT INTO operations.licences (licence_key, organisation_name, contact_email) VALUES ($1, $2, $3)`

		// Create initial licence
		testDB.Exec(ctx, query, licenceKey, organisationName, contactEmail)

		// Attempt to create another licence with the same key
		_, err := testDB.Exec(ctx, query, licenceKey, organisationName, contactEmail)

		// Check for unique violation error
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), fmt.Sprintf("Expected unique violation error, got: %v", err))
	})

	t.Run("TC-LICE-03 - Licence key must follow the format 'YOP-XXXXX' where X is a digit", func(t *testing.T) {
		t.Parallel()
		// Fill params
		invalidLicenceKey := "INVALID-KEY"
		organisationName := "Test"
		contactEmail := "test@test.com"

		// Create query
		query := `INSERT INTO operations.licences (licence_key, organisation_name, contact_email) VALUES ($1, $2, $3)`

		// Attempt to create a licence with an invalid key
		_, err := testDB.Exec(ctx, query, invalidLicenceKey, organisationName, contactEmail)

		// Check for check violation error
		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), fmt.Sprintf("Expected check violation error, got: %v", err))
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()
		// Fill params
		licenceKey := "YOP-67890"
		contactEmail := "test@test.com"
		organisationName := "Test"

		// Create query
		query := `INSERT INTO operations.licences (licence_key, organisation_name, contact_email) VALUES ($1, $2, $3)`
		t.Run("TC-LICE-04 - The organisation name is required", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a licence with a missing organisation name
			_, err := testDB.Exec(ctx, query, licenceKey, nil, contactEmail)

			// Check for null value violation error
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), fmt.Sprintf("Expected not null violation error, got: %v", err))
		})

		t.Run("TC-LICE-06 - The contact email is required", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a licence with a missing contact email
			_, err := testDB.Exec(ctx, query, licenceKey, organisationName, nil)

			// Check for null value violation error
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), fmt.Sprintf("Expected not null violation error, got: %v", err))
		})

	})

	t.Run("Character Limits", func(t *testing.T) {
		// Fill params
		licenceKey := "YOP-54321"
		organisationName := "Valid Org Name"
		contactEmail := "test@test.com"

		// Create query
		query := `INSERT INTO operations.licences (licence_key, organisation_name, contact_email, licence_notes) VALUES ($1, $2, $3, $4)`

		t.Run("TC-LICE-05 - The organisation name must not exceed 50 characters", func(t *testing.T) {
			// Attempt to create a licence with an overly long organisation name
			_, err := testDB.Exec(ctx, query, licenceKey, strings.Repeat("a", 51), contactEmail, nil)

			// Check for check violation error
			assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), fmt.Sprintf("Expected check violation error, got: %v", err))
		})

		t.Run("TC-LICE-08 - The licence notes must not exceed 1500 characters", func(t *testing.T) {
			// Attempt to create a licence with overly long licence notes
			_, err := testDB.Exec(ctx, query, licenceKey, organisationName, contactEmail, strings.Repeat("a", 1501))

			// Check for check violation error
			assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), fmt.Sprintf("Expected check violation error, got: %v", err))
		})
	})
}
