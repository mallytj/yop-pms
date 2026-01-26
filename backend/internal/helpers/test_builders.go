package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/types"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// BuildAndServeHttpRequest is a helper function to build and serve an HTTP request.
// method: HTTP method (GET, POST, etc.)
// url: Request URL
// body: Request body (can be nil)
// r: chi.Mux router to serve the request
// Returns the ResponseRecorder
// Example: BuildAndServeHttpRequest("POST", "/users", params, r) => *httptest.ResponseRecorder
func BuildAndServeHttpRequest(method string, url string, body interface{}, r *chi.Mux) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewReader(jsonBody)
	} else {
		reqBody = bytes.NewReader([]byte{})
	}
	req := httptest.NewRequest(method, url, reqBody)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	return rr
}

// CreateTestLicence is a helper function to create a test licence with the given licence key.
// Returns the created licence.
// licenceKey: Must be in the format "XXX-YYYY" where X is uppercase letter and Y is digit.
// testQueries: Database queries interface for executing SQL commands.
// Example: CreateTestLicence(t, "TEST-1234") = repo.Licence{...}
func CreateTestLicence(t *testing.T, licenceKey string, testQueries *repo.Queries) repo.Licence {
	ctx := context.Background()
	lic, err := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
		LicenceKey:       licenceKey,
		OrganisationName: "Test Organisation",
		ContactEmail:     "test@example.com",
	})
	require.NoError(t, err)
	return lic
}

// CreateTestUser is a helper function to create a test user with the given parameters.
// Returns the created user.
// params: Parameters required to create the user.
// testQueries: Database queries interface for executing SQL commands.
// Example: CreateTestUser(t, params, testQueries) = repo.User{...}
func CreateTestUser(t *testing.T, params types.CreateUserParams, testQueries *repo.Queries) repo.User {
	ctx := context.Background()

	// Go service route to create first user directly
	user, err := testQueries.CreateUser(ctx, repo.CreateUserParams{
		LicenceID:    ToPgUUID(&params.LicenceID),
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: params.Password,
		FirstName:    params.FirstName,
		LastName:     params.LastName,
		Role:         string(params.Role),
		IsActive:     ToPgBool(&params.IsActive),
	})

	// Ensure no error occurred during first user creation
	require.NoError(t, err, fmt.Sprintf("failed to create test user: %v", err))

	return user
}

// CreateTestProperty is a helper function to create a test property with the given parameters.
// Returns the created property.
// params: Parameters required to create the property.
// testQueries: Database queries interface for executing SQL commands.
// Example: CreateTestProperty(t, params, testQueries) = repo.Property{...}
func CreateTestProperty(t *testing.T, params repo.CreatePropertyParams, testQueries *repo.Queries) repo.Property {
	// Create context
	ctx := context.Background()

	// Create property in the database
	property, err := testQueries.CreateProperty(ctx, params)

	// Ensure no error occurred during property creation
	require.NoError(t, err, fmt.Sprintf("failed to create test property: %v", err))

	// Return the created property
	return property
}

// CreateTestPropertyAmenity is a helper function to create a test property amenity with the given parameters.
// Returns the created property amenity.
// params: Parameters required to create the property amenity.
// testQueries: Database queries interface for executing SQL commands.
// Example: CreateTestPropertyAmenity(t, params, testQueries) = repo.PropertyAmenity{...}
func CreateTestPropertyAmenity(t *testing.T, params repo.CreatePropertyAmenityParams, licKey string, testQueries *repo.Queries) repo.PropertyAmenity {
	// Create context
	ctx := context.Background()

	// Create licence in the database
	lic, err := testQueries.GetLicenceByKey(ctx, licKey)
	licId := &lic.ID
	if err != nil {
		lic = CreateTestLicence(t, licKey, testQueries)
		licId = &lic.ID
	}

	// Ensure the property ID is set to a valid property

	if !params.PropertyID.Valid || params.PropertyID.Bytes == uuid.Nil {
		property := CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: *licId,
			Name:      "Test Property",
			Address:   "123 Test St, Test City",
			Timezone:  "Test/Europe",
		}, testQueries)

		params.PropertyID = property.ID
	}

	// Create property amenity in the database
	propertyAmenity, err := testQueries.CreatePropertyAmenity(ctx, params)

	// Ensure no error occurred during property amenity creation
	require.NoError(t, err, fmt.Sprintf("failed to create test property amenity: %v", err))

	// Return the created property amenity
	return propertyAmenity
}
