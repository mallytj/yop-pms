package db_tests

import (
	"context"
	"strings"
	"testing"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDbProperties(t *testing.T) {
	ctx := context.Background()
	// TC-PROP-01 in db_migration_test.go

	t.Run("TC-PROP-02 - The licence must exist & be active to add properties", func(t *testing.T) {
		t.Parallel()
		// Fill params
		params := TestCreatePropertyParams{
			Name:     "Test Property",
			Address:  "123 Test St, Test City",
			Timezone: "UTC",
		}

		t.Run("TC-PROP-02a - Licence must exist", func(t *testing.T) {
			t.Parallel()
			// Fill params

			params.LicenceID = uuid.New() // Non-existent licence

			// Create query
			query := `INSERT INTO operations.properties (licence_id, name, address, timezone) VALUES ($1, $2, $3, $4)`

			// Attempt to create a property with a non-existent licence
			_, err := testDB.Exec(ctx, query, params.LicenceID, params.Name, params.Address, params.Timezone)

			// Check for raise exception error
			assert.True(t, hf.CheckErrorCode(err, hf.RaiseExceptionCode), "Expected raise exception error, got: %v", err)
		})

		t.Run("TC-PROP-02b - Licence must be active", func(t *testing.T) {
			t.Parallel()
			// First, create an inactive licence
			inactiveLicence := GenerateTestLicence(t, ctx, false)

			params.LicenceID = inactiveLicence.ID

			// Create query
			query := `INSERT INTO operations.properties (licence_id, name, address, timezone) VALUES ($1, $2, $3, $4)`

			// Attempt to create a property with an inactive licence
			_, err := testDB.Exec(ctx, query, params.LicenceID, params.Name, params.Address, params.Timezone)

			// Check for raise exception error
			assert.True(t, hf.CheckErrorCode(err, hf.RaiseExceptionCode), "Expected raise exception error, got: %v", err)
		})
	})

	t.Run("Required fields", func(t *testing.T) {
		t.Parallel()

		activeLicence := GenerateTestLicence(t, ctx, true)

		params := TestCreatePropertyParams{
			LicenceID: activeLicence.ID,
			Name:      "Test Property",
			Address:   "123 Test St, Test City",
			Timezone:  "UTC",
		}

		// Create query
		query := `INSERT INTO operations.properties (licence_id, name, address, timezone) VALUES ($1, $2, $3, $4)`

		t.Run("TC-PROP-03 - The property name is required", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a property with a missing name
			_, err := testDB.Exec(ctx, query, params.LicenceID, nil, params.Address, params.Timezone)

			// Check for null value violation error
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error, got: %v", err)
		})

		t.Run("TC-PROP-05 - The property address is required", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a property with a missing address
			_, err := testDB.Exec(ctx, query, params.LicenceID, params.Name, nil, params.Timezone)

			// Check for null value violation error
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error, got: %v", err)
		})

		t.Run("TC-PROP-07 - The property timezone is required", func(t *testing.T) {
			t.Parallel()
			// Attempt to create a property with a missing timezone
			_, err := testDB.Exec(ctx, query, params.LicenceID, params.Name, params.Address, nil)

			// Check for null value violation error
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error, got: %v", err)
		})
	})

	t.Run("Character Limits", func(t *testing.T) {
		t.Parallel()

		// Fill params
		activeLicence := GenerateTestLicence(t, ctx, true)

		params := TestCreatePropertyParams{
			LicenceID: activeLicence.ID,
			Name:      "Valid Property Name",
			Address:   "123 Test St, Test City",
			Timezone:  "UTC",
		}

		// Create query
		query := `INSERT INTO operations.properties (licence_id, name, address, timezone, property_notes) VALUES ($1, $2, $3, $4, $5)`

		t.Run("TC-PROP-04 - The name must not exceed 50 characters", func(t *testing.T) {
			t.Parallel()

			// Attempt to create a property with a overly long name
			_, err := testDB.Exec(ctx, query, params.LicenceID, strings.Repeat("a", 51), params.Address, params.Timezone, nil)

			// Check for check violation error
			assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error, got: %v", err)
		})

		t.Run("TC-PROP-06 - The address must not exceed 150 characters", func(t *testing.T) {
			t.Parallel()

			// Attempt to create a property with overly long address
			_, err := testDB.Exec(ctx, query, params.LicenceID, params.Name, strings.Repeat("a", 251), params.Timezone, nil)

			// Check for check violation error
			assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error, got: %v", err)
		})

		t.Run("TC-PROP-09 - The property notes must not exceed 1500 characters", func(t *testing.T) {
			t.Parallel()

			// Attempt to create a property with overly long notes
			_, err := testDB.Exec(ctx, query, params.LicenceID, params.Name, params.Address, params.Timezone, strings.Repeat("a", 1501))

			// Check for check violation error
			assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error, got: %v", err)
		})
	})

	t.Run("TC-PROP-10 - There must be only one property per address per licence", func(t *testing.T) {
		t.Parallel()
		// Fill params
		activeLicence := GenerateTestLicence(t, ctx, true)

		params := TestCreatePropertyParams{
			LicenceID: activeLicence.ID,
			Name:      "Test Property",
			Address:   "123 Test St, Test City",
			Timezone:  "UTC",
		}

		// Create query
		query := `INSERT INTO operations.properties (licence_id, name, address, timezone) VALUES ($1, $2, $3, $4)`

		// First, create a property with the given address
		_, err := testDB.Exec(ctx, query, params.LicenceID, params.Name, params.Address, params.Timezone)
		assert.NoError(t, err)

		// Attempt to create another property with the same address under the same licence
		_, err = testDB.Exec(ctx, query, params.LicenceID, "Another Property", params.Address, params.Timezone)

		// Check for unique violation error
		assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error, got: %v", err)
	})
}
