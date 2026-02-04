//go:build ignore
package db_tests

import (
	"context"
	hf "ollerod-pms/internal/helpers"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)


// TestDbUsers contains tests related to the auth.users table and its constraints.
func TestDbUsers(t *testing.T) {
	// TC-USER-01 in db_migration_test.go
	ctx := context.Background()

	t.Run("TC-USER-02 - The user's licence must exist and be active", func(t *testing.T) {
		t.Parallel()

		// Create a test licence
		licenceID := GenerateTestLicence(t, ctx, true).ID

		// Hash a test password
		hashedPassword, err := hf.HashPassword("test")
		assert.NoError(t, err, "Failed to hash password: %v", err)

		// Fill user params
		params := CreateTestUser{
			LicenceID:    licenceID.String(),
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: hashedPassword,
			FirstName:    "Test",
			LastName:     "User",
			Role:         "staff",
			IsActive:     true,
		}

		// Create query to insert user
		insertUserQuery := `
			INSERT INTO auth.users (licence_id, username, email, password_hash, first_name, last_name, role, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		// Sub-test: Licence must exist
		t.Run("Non-existent licence", func(t *testing.T) {
			t.Parallel()

			// Use a non-existent licence ID
			nonExistentLicenceID := "00000000-0000-0000-0000-000000000000"

			// Attempt to create a user with a non-existent licence
			_, err := testDB.Exec(ctx, insertUserQuery, nonExistentLicenceID, params.Username, params.Email, params.PasswordHash, params.FirstName, params.LastName, params.Role, params.IsActive)

			assert.True(t, hf.CheckErrorCode(err, hf.RaiseExceptionCode), "Expected foreign key violation error for non-existent licence, got: %v", err)
		})

		t.Run("Inactive licence", func(t *testing.T) {
			t.Parallel()

			// Create an inactive test licence
			inactiveLicenceID := GenerateTestLicence(t, ctx, false).ID

			// Attempt to create a user with an inactive licence
			_, err := testDB.Exec(ctx, insertUserQuery, inactiveLicenceID.String(), params.Username, params.Email, params.PasswordHash, params.FirstName, params.LastName, params.Role, params.IsActive)

			assert.True(t, hf.CheckErrorCode(err, hf.RaiseExceptionCode), "Expected check violation error for inactive licence, got: %v", err)
		})

		t.Run("Valid active licence", func(t *testing.T) {
			t.Parallel()

			// Attempt to create a user with a valid active licence
			_, err := testDB.Exec(ctx, insertUserQuery, params.LicenceID, params.Username, params.Email, params.PasswordHash, params.FirstName, params.LastName, params.Role, params.IsActive)

			assert.NoError(t, err, "Expected no error when creating user with valid active licence, got: %v", err)
		})
	})

	t.Run("Required Fields", func(t *testing.T) {
		t.Parallel()

		// Create a test licence
		licenceID := GenerateTestLicence(t, ctx, true).ID

		// Hash a test password
		hashedPassword, err := hf.HashPassword("test")
		assert.NoError(t, err, "Failed to hash password: %v", err)

		// Base user params
		params := CreateTestUser{
			LicenceID:    licenceID.String(),
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: hashedPassword,
			FirstName:    "Test",
			LastName:     "User",
			Role:         "staff",
			IsActive:     true,
		}
		// Create query to insert user
		insertString := `
			INSERT INTO auth.users (licence_id, username, email, password_hash, first_name, last_name, role, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		t.Run("TC-USER-03 - Username is required", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx,
				insertString,
				params.LicenceID, nil, params.Email, params.PasswordHash, params.FirstName, params.LastName, params.Role, params.IsActive,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error for missing username, got: %v", err)
		})

		t.Run("TC-USER-07 - Email is required", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx,
				insertString,
				params.LicenceID, params.Username, nil, params.PasswordHash, params.FirstName, params.LastName, params.Role, params.IsActive,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error for missing email, got: %v", err)
		})

		t.Run("TC-USER-08 - Password hash is required", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx,
				insertString,
				params.LicenceID, params.Username, params.Email, nil, params.FirstName, params.LastName, params.Role, params.IsActive,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error for missing password hash, got: %v", err)
		})

		t.Run("TC-USER-09 - The role is required", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx,
				insertString,
				params.LicenceID, params.Username, params.Email, params.PasswordHash, params.FirstName, params.LastName, nil, params.IsActive,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error for missing role, got: %v", err)
		})

		t.Run("TC-USER-11 - First name is required", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx,
				insertString,
				params.LicenceID, params.Username, params.Email, params.PasswordHash, nil, params.LastName, params.Role, params.IsActive,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error for missing first name, got: %v", err)
		})

		t.Run("TC-USER-14 - Last name is required", func(t *testing.T) {
			t.Parallel()

			_, err := testDB.Exec(ctx,
				insertString,
				params.LicenceID, params.Username, params.Email, params.PasswordHash, params.FirstName, nil, params.Role, params.IsActive,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.NotNullViolationCode), "Expected not null violation error for missing last name, got: %v", err)
		})
	})

	t.Run("Unique Fields", func(t *testing.T) {
		licenceID := GenerateTestLicence(t, ctx, true).ID
		hashedPassword, err := hf.HashPassword("123")
		assert.NoError(t, err, "Failed to hash password: %v", err)

		t.Parallel()
		userParams := CreateTestUser{
			LicenceID:    licenceID.String(),
			Username:     "uniqueuser",
			Email:        "uniqueuser@example.com",
			PasswordHash: hashedPassword,
			FirstName:    "Unique",
			LastName:     "User",
			Role:         "staff",
			IsActive:     true,
		}

		// Create query to insert user
		insertUserQuery := `
				INSERT INTO auth.users (licence_id,username, email, password_hash, first_name, last_name, role, is_active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`

		// Insert the first user
		_, err = testDB.Exec(ctx, insertUserQuery,
			userParams.LicenceID,
			userParams.Username,
			userParams.Email,
			userParams.PasswordHash,
			userParams.FirstName,
			userParams.LastName,
			userParams.Role,
			userParams.IsActive,
		)
		assert.NoError(t, err, "Failed to insert first user: %v", err)

		t.Run("TC-USER-04 - Username must be unique", func(t *testing.T) {
			t.Parallel()

			// Attempt to insert a second user with the same username
			_, err = testDB.Exec(ctx, insertUserQuery,
				userParams.LicenceID,
				userParams.Username, // Same username
				"anotheruser@example.com",
				userParams.PasswordHash,
				userParams.FirstName,
				userParams.LastName,
				userParams.Role,
				true,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error for duplicate username, got: %v", err)
		})

		t.Run("TC-USER-26 - Email must be unique", func(t *testing.T) {
			t.Parallel()
			// Attempt to insert a second user with the same email
			_, err = testDB.Exec(ctx, insertUserQuery,
				userParams.LicenceID,
				"anotheruser",
				userParams.Email, // Same email
				userParams.PasswordHash,
				userParams.FirstName,
				userParams.LastName,
				userParams.Role,
				true,
			)
			assert.True(t, hf.CheckErrorCode(err, hf.UniqueViolationCode), "Expected unique violation error for duplicate email, got: %v", err)
		})

	})

	t.Run("Alphanumerical Checks", func(t *testing.T) {
		t.Parallel()
		// Create a test licence
		licenceID := GenerateTestLicence(t, ctx, true).ID

		// Hash a test password
		hashedPassword, err := hf.HashPassword("test")
		assert.NoError(t, err, "Failed to hash password: %v", err)

		params := CreateTestUser{
			LicenceID:    licenceID.String(),
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: hashedPassword,
			FirstName:    "Test",
			LastName:     "User",
			Role:         "staff",
			IsActive:     true,
		}

		// Create query to insert user
		insertUserQuery := `
				INSERT INTO auth.users (licence_id,username, email, password_hash, first_name, last_name, role, is_active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`

		t.Run("TC-USER-05 The username must be alphanumerical (or _)", func(t *testing.T) {
			t.Parallel()

			testCases := []FieldTestCase{
				{"validUser_123", true},
				{"user.name", false},
				{"user-name", false},
				{"user name", false},
				{"user@name", false},
				{"user!", false},
			}

			for _, test := range testCases {
				t.Run("Username: "+test.example+"'", func(t *testing.T) {
					_, err := testDB.Exec(ctx, insertUserQuery,
						params.LicenceID,
						test.example, // To ensure uniqueness
						test.example+params.Email,
						params.PasswordHash,
						params.FirstName,
						params.LastName,
						params.Role,
						params.IsActive,
					)

					if test.result {
						assert.NoError(t, err, "Expected no error for valid username '%s', got: %v", test.example, err)
					} else {
						assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for invalid username '%s', got: %v", test.example, err)
					}
				})

			}
		})

		t.Run("First & Last Name Checks", func(t *testing.T) {
			t.Parallel()

			tests := []FieldTestCase{
				{"John", true},
				{"Mary-Anne", true},
				{"O'Connor", true},
				{"Jean-Luc", true},
				{"Anna Maria", false},
				{"John3", false},
				{"@lice", false},
			}

			for i, test := range tests {
				t.Run("TC-USER-13 - First name validation for '"+test.example+"'", func(t *testing.T) {
					_, err := testDB.Exec(ctx, insertUserQuery,
						params.LicenceID,
						params.Username+strings.Repeat("x", i+1), // To ensure uniqueness
						test.example+params.Email,
						params.PasswordHash,
						test.example,
						params.LastName,
						params.Role,
						params.IsActive,
					)

					if test.result {
						assert.NoError(t, err, "Expected no error for valid first name '%s', got: %v", test.example, err)
					} else {
						assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for invalid first name '%s', got: %v", test.example, err)
					}
				})

				t.Run("TC-USER-16 - Last name validation for '"+test.example+"'", func(t *testing.T) {
					_, err := testDB.Exec(ctx, insertUserQuery,
						params.LicenceID,
						params.Username+strings.Repeat("y", i+1), // To ensure uniqueness
						params.Email+test.example,
						params.PasswordHash,
						params.FirstName,
						test.example,
						params.Role,
						params.IsActive,
					)

					if test.result {
						assert.NoError(t, err, "Expected no error for valid last name '%s', got: %v", test.example, err)
					} else {
						assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for invalid last name '%s', got: %v", test.example, err)
					}
				})
			}
		})
	})

	t.Run("Character Limits", func(t *testing.T) {
		t.Parallel()
		// Create a test licence
		licenceID := GenerateTestLicence(t, ctx, true).ID

		// Hash a test password
		hashedPassword, err := hf.HashPassword("test")
		assert.NoError(t, err, "Failed to hash password: %v", err)

		// Base user params
		params := CreateTestUser{
			LicenceID:    licenceID.String(),
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: hashedPassword,
			FirstName:    "Test",
			LastName:     "User",
			Role:         "staff",
			IsActive:     true,
		}

		// Define constraint tests
		tests := []hf.ConstraintTest{
			{
				Name:        "TC-USER-06 - The username must not exceed 20 characters",
				Field:       "username",
				Value:       strings.Repeat("a", 21),
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  1, // Username is the 2nd field (index 1)
			},
			{
				Name:        "TC-USER-12 - First name must not exceed 50 characters",
				Field:       "first_name",
				Value:       strings.Repeat("a", 51),
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  4, // First name is the 5th field (index 4)
			},
			{
				Name:        "TC-USER-15 - Last name must not exceed 50 characters",
				Field:       "last_name",
				Value:       strings.Repeat("a", 51),
				ExpectedErr: hf.CheckViolationCode,
				FieldIndex:  5, // Last name is the 6th field (index 5)
			},
		}

		// Create query to insert user
		insertUserQuery := `
		INSERT INTO auth.users (licence_id,username, email, password_hash, first_name, last_name, role, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		hf.RunConstraintTests(t, ctx, testDB, insertUserQuery, hf.StructToSlice(params), tests)
	})

	t.Run("TC-USER-10 - The role must be one of the defined enum values", func(t *testing.T) {
		t.Parallel()

		// Create a test licence
		licenceID := GenerateTestLicence(t, ctx, true).ID

		// Hash a test password
		hashedPassword, err := hf.HashPassword("test")
		assert.NoError(t, err, "Failed to hash password: %v", err)

		params := CreateTestUser{
			LicenceID:    licenceID.String(),
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: hashedPassword,
			FirstName:    "Test",
			LastName:     "User",
			Role:         "invalid_role",
			IsActive:     true,
		}

		// Create query to insert user
		insertUserQuery := `
				INSERT INTO auth.users (licence_id,username, email, password_hash, first_name, last_name, role, is_active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`

		_, err = testDB.Exec(ctx, insertUserQuery,
			params.LicenceID,
			params.Username,
			params.Email,
			params.PasswordHash,
			params.FirstName,
			params.LastName,
			params.Role,
			params.IsActive,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.InvalidTextRepresentationCode), "Expected invalid text representation error for invalid role, got: %v", err)
	})

	t.Run("TC-USER-19 - A user's password must not be stored in plain text", func(t *testing.T) {
		t.Parallel()

		// Create a test licence
		licenceID := GenerateTestLicence(t, ctx, true).ID

		// Plain text password
		plainPassword := "plaintextpassword"

		params := CreateTestUser{
			LicenceID:    licenceID.String(),
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: plainPassword,
			FirstName:    "Test",
			LastName:     "User",
			Role:         "staff",
			IsActive:     true,
		}
		// Create query to insert user
		insertUserQuery := `
				INSERT INTO auth.users (licence_id,username, email, password_hash, first_name, last_name, role, is_active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`

		_, err := testDB.Exec(ctx, insertUserQuery,
			params.LicenceID,
			params.Username,
			params.Email,
			params.PasswordHash,
			params.FirstName,
			params.LastName,
			params.Role,
			params.IsActive,
		)
		assert.True(t, hf.CheckErrorCode(err, hf.CheckViolationCode), "Expected check violation error for plain text password, got: %v", err)
	})
}
