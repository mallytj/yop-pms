package users

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/helpers"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDB      *pgxpool.Pool
	testQueries *repo.Queries
)

// TestMain sets up the test environment, including starting a PostgreSQL container,
// running migrations, and initializing the test database connection.
// This function is executed before any tests are run.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1. Start Container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("pms_test"),
		postgres.WithUsername("admin"),
		postgres.WithPassword("password"),

		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432").WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()

	connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")

	// 2. Run Migrations
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	// Point this to your actual migrations folder
	if err := goose.Up(db, "../adapters/postgresql/migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}
	db.Close()

	// 3. Setup global test connection
	testDB, err = pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatal(err)
	}
	testQueries = repo.New(testDB)

	code := m.Run()

	testDB.Close()
	pgContainer.Terminate(ctx)
	os.Exit(code)
}

// createTestLicence is a helper function to create a test licence with the given licence key.
// Returns the created licence.
// licenceKey: Must be in the format "XXX-YYYY" where X is uppercase letter and Y is digit.
// Example: createTestLicence(t, "TEST-1234") = repo.Licence{...}
func createTestLicence(t *testing.T, licenceKey string) repo.Licence {
	ctx := context.Background()
	lic, err := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
		LicenceKey:       licenceKey,
		OrganisationName: "Test Organisation",
		ContactEmail:     "test@example.com",
	})
	require.NoError(t, err)
	return lic
}

// buildAndServeHttpRequest is a helper function to build and serve an HTTP request.
// method: HTTP method (GET, POST, etc.)
// url: Request URL
// body: Request body (can be nil)
// r: chi.Mux router to serve the request
// Returns the ResponseRecorder
// Example: buildAndServeHttpRequest("POST", "/users", params, r) => *httptest.ResponseRecorder
func buildAndServeHttpRequest(method string, url string, body interface{}, r *chi.Mux) *httptest.ResponseRecorder {
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

// createTestUser is a helper function to create a test user with the given parameters.
// Returns the created user.
// params: Parameters required to create the user.
// Example: createTestUser(t, params) = repo.User{...}
func createTestUser(t *testing.T, params createUserParams) repo.User {
	ctx := context.Background()

	// Go service route to create first user directly
	user, err := testQueries.CreateUser(ctx, repo.CreateUserParams{
		LicenceID:    helpers.ToPgUUID(&params.LicenceID),
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: params.Password,
		FirstName:    params.FirstName,
		LastName:     params.LastName,
		Role:         string(params.Role),
		IsActive:     helpers.ToPgBool(&params.IsActive),
	})

	// Ensure no error occurred during first user creation
	require.NoError(t, err, fmt.Sprintf("failed to create test user: %v", err))

	return user
}

// TestUserFlow tests the complete user flow including creating, retrieving, listing, and updating users.
// It also tests various edge cases and error scenarios.
// This test assumes that the database is clean before running.
func TestUserFlow(t *testing.T) {
	// Setup test environment
	ctx := context.Background()
	svc := NewService(*testQueries, testDB)
	h := NewHandler(svc)

	t.Run("Create User", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a test licence
		testLicence := createTestLicence(t, "CRE-0001")

		// Then, create the user
		params := createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "testuser",
			Email:     "testuser@example.com",
			Password:  "hashedpassword",
			FirstName: "Test",
			LastName:  "User",
			Role:      "user",
			IsActive:  true,
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Post("/users", h.CreateUser)

		// Build HTTP request
		rr := buildAndServeHttpRequest(http.MethodPost, "/users", params, r)

		// Assert the request was successful
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Get response body, unmarshal into User struct
		var createdUser repo.User
		err := json.Unmarshal(rr.Body.Bytes(), &createdUser)

		// Ensure no error occurred during unmarshalling
		require.NoError(t, err)

		// Validate fields of created user to ensure they match input params
		assert.Equal(t, params.Username, createdUser.Username)
		assert.Equal(t, params.Email, createdUser.Email)
		assert.Equal(t, params.FirstName, createdUser.FirstName)
		assert.Equal(t, params.LastName, createdUser.LastName)
		assert.Equal(t, string(params.Role), createdUser.Role)
		assert.Equal(t, params.LicenceID, uuid.MustParse(createdUser.LicenceID.String()))
		assert.Equal(t, params.IsActive, createdUser.IsActive.Bool)
		assert.NotEmpty(t, createdUser.ID)                            // Ensure ID is set
		assert.NotEmpty(t, createdUser.CreatedAt)                     // Ensure CreatedAt is set
		assert.NotEmpty(t, createdUser.UpdatedAt)                     // Ensure UpdatedAt is set
		assert.Equal(t, createdUser.CreatedAt, createdUser.UpdatedAt) // On creation, these should be equal
	})

	t.Run("Create User - Duplicate Email", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a test licence
		testLicence := createTestLicence(t, "CRE-0002")

		// Then, create the first user with a specific email
		createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "userone",
			Email:     "userone@example.com",
			Password:  "hashedpassword",
			FirstName: "User",
			LastName:  "One",
			Role:      "user",
			IsActive:  true,
		})

		// Set up params for second user with the same email as the first user
		userTwoParams := createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "usertwo",
			Email:     "userone@example.com",
			Password:  "hashedpassword",
			FirstName: "User",
			LastName:  "Two",
			Role:      "user",
			IsActive:  true,
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Post("/users", h.CreateUser)

		// Build HTTP request
		rr := buildAndServeHttpRequest(http.MethodPost, "/users", userTwoParams, r)

		// Assert that the response status code indicates a conflict due to duplicate email
		assert.Equal(t, http.StatusConflict, rr.Code)
	})

	t.Run("Create User - Invalid Role", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a test licence
		testLicence := createTestLicence(t, "CRE-0003")

		// Then, build params with an invalid role
		params := createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "invalidroleuser",
			Email:     "invalidrole@example.com",
			Password:  "hashedpassword",
			FirstName: "Invalid",
			LastName:  "Role",
			Role:      "invalid",
			IsActive:  true,
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Post("/users", h.CreateUser)

		// Build HTTP request and serve
		rr := buildAndServeHttpRequest(http.MethodPost, "/users", params, r)

		// Assert that the response status code indicates a bad request due to invalid email format
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create User - Non-existent Licence", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, build params with a non-existent licence ID
		params := createUserParams{
			LicenceID: uuid.New(),
			Username:  "nonexistentuser",
			Email:     "nonexistent@example.com",
			Password:  "hashedpassword",
			FirstName: "Nonexistent",
			LastName:  "User",
			Role:      "user",
			IsActive:  true,
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Post("/users", h.CreateUser)

		// Build HTTP request and serve
		rr := buildAndServeHttpRequest(http.MethodPost, "/users", params, r)

		// Assert that the response status code indicates a bad request due to non-existent licence
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create User - Missing Fields", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a test licence
		testLicence := createTestLicence(t, "CRE-0004")

		// Then, build params with missing required fields
		params := createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "missinguser",
			Email:     "", // Required field left empty
			Password:  "hashedpassword",
			FirstName: "", // Optional field left empty
			LastName:  "", // Optional field left empty
			Role:      "user",
			IsActive:  true,
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Post("/users", h.CreateUser)

		// Build HTTP request and serve
		rr := buildAndServeHttpRequest(http.MethodPost, "/users", params, r)

		// Assert that the response status code indicates a bad request due to missing required fields
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create User - Invalid Email Format", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a test licence
		testLicence := createTestLicence(t, "CRE-0005")

		// Then, build params with an invalid email format
		params := createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "invalidemailuser",
			Email:     "invalid-email-format",
			Password:  "hashedpassword",
			FirstName: "Invalid",
			LastName:  "Email",
			Role:      "user",
			IsActive:  true,
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Post("/users", h.CreateUser)

		// Build HTTP request and serve
		rr := buildAndServeHttpRequest(http.MethodPost, "/users", params, r)

		// Assert that the response status code indicates a bad request due to invalid email format
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Get User By ID", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution
		// First, create a test licence
		testLicence := createTestLicence(t, "CRE-0006")

		// Then, build params to create the user
		createdUser := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "getbyiduser",
			Email:     "getbyiduser@example.com",
			Password:  "hashedpassword",
			FirstName: "Get",
			LastName:  "ByID",
			Role:      "user",
			IsActive:  true,
		})

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Get("/users/{userID}", h.GetUserById)

		// Build HTTP request
		rr := buildAndServeHttpRequest(http.MethodGet, "/users/"+createdUser.ID.String(), nil, r)

		// Validate response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Unmarshal response body into User struct
		var retrievedUser repo.User
		err := json.Unmarshal(rr.Body.Bytes(), &retrievedUser)

		// Ensure no error occurred during unmarshalling
		require.NoError(t, err)

		// Assert retrieved user matches created user
		assert.Equal(t, createdUser.ID, retrievedUser.ID)
		assert.Equal(t, createdUser.Username, retrievedUser.Username)
		assert.Equal(t, createdUser.Email, retrievedUser.Email)
		assert.Equal(t, createdUser.FirstName, retrievedUser.FirstName)
		assert.Equal(t, createdUser.LastName, retrievedUser.LastName)
		assert.Equal(t, createdUser.Role, retrievedUser.Role)
		assert.Equal(t, createdUser.LicenceID, retrievedUser.LicenceID)
		assert.Equal(t, createdUser.IsActive, retrievedUser.IsActive)
		assert.NotEmpty(t, retrievedUser.CreatedAt)
		assert.NotEmpty(t, retrievedUser.UpdatedAt)
	})

	t.Run("List Users", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a test licence
		testLicence := createTestLicence(t, "LIS-0001")

		// Create two test users
		user1 := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "listuser1",
			Email:     "listuser1@example.com",
			Password:  "hashedpassword",
			FirstName: "List",
			LastName:  "User1",
			Role:      "user",
			IsActive:  true,
		})

		createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "listuser2",
			Email:     "listuser2@example.com",
			Password:  "hashedpassword",
			FirstName: "List",
			LastName:  "User2",
			Role:      "user",
			IsActive:  true,
		})

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Get("/users", h.ListUsers)

		// Build HTTP request and serve
		rr := buildAndServeHttpRequest(http.MethodGet, "/users", nil, r)

		// Assert the request was successful and response code is 200 OK
		assert.Equal(t, http.StatusOK, rr.Code)

		// Unmarshal response body into slice of User structs
		var users []repo.User
		err := json.Unmarshal(rr.Body.Bytes(), &users)
		require.NoError(t, err)

		// Assert that the created users are in the list
		foundUser1 := false
		for _, u := range users {
			if u.ID == user1.ID {
				foundUser1 = true
				break
			}
		}
		assert.True(t, foundUser1, "Created user1 should be in the list of users")

		// Assert at least two users are returned
		assert.GreaterOrEqual(t, len(users), 2)
	})

	t.Run("Update User", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution
		// First, create a test licence
		testLicence := createTestLicence(t, "UPD-0001")

		// Then, create a user to update
		createdUser := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "updateuser",
			Email:     "updateuser@example.com",
			Password:  "hashedpassword",
			FirstName: "Update",
			LastName:  "User",
			Role:      "user",
			IsActive:  true,
		})

		// Create a new licence to update to
		newLicence := createTestLicence(t, "UPD-9999")

		// Now attempt to update the user with all fields changed
		updateParams := updateUserParams{
			UserID:    uuid.MustParse(createdUser.ID.String()),
			Username:  helpers.Ptr("updateduser"),
			Email:     helpers.Ptr("updateduser@example.com"),
			Password:  helpers.Ptr("newhashedpassword"),
			FirstName: helpers.Ptr("Updated"),
			LastName:  helpers.Ptr("UserUpdated"),
			Role:      helpers.Ptr("manager"),
			LicenceID: helpers.Ptr(uuid.MustParse(newLicence.ID.String())),
			IsActive:  helpers.Ptr(false),
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Put("/users/{userID}", h.UpdateUser)

		rr := buildAndServeHttpRequest(http.MethodPut, "/users/"+createdUser.ID.String(), updateParams, r)

		// Validate response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Unmarshal response body into User struct
		var updatedUser repo.User
		err := json.Unmarshal(rr.Body.Bytes(), &updatedUser)
		require.NoError(t, err)

		// Validate fields of updated user to ensure they match update params
		assert.Equal(t, createdUser.ID, updatedUser.ID)
		assert.Equal(t, *updateParams.Username, updatedUser.Username)
		assert.Equal(t, *updateParams.Email, updatedUser.Email)
		assert.Equal(t, *updateParams.FirstName, updatedUser.FirstName)
		assert.Equal(t, *updateParams.LastName, updatedUser.LastName)
		assert.Equal(t, *updateParams.Role, updatedUser.Role)
		assert.Equal(t, newLicence.ID, updatedUser.LicenceID)
		assert.Equal(t, pgtype.Bool{Bool: false, Valid: true}, updatedUser.IsActive)
		assert.NotEmpty(t, updatedUser.CreatedAt)
		assert.NotEmpty(t, updatedUser.UpdatedAt)
	})

	t.Run("Update User - Non-existent User", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution
		// First, build update params with a non-existent user ID
		fakeUUID := uuid.New()
		updateParms := updateUserParams{
			UserID:   fakeUUID,
			IsActive: helpers.Ptr(false),
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Put("/users/{userID}", h.UpdateUser)

		// Build and serve HTTP request
		rr := buildAndServeHttpRequest(http.MethodPut, "/users/"+fakeUUID.String(), updateParms, r)

		// Validate response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Update User - Duplicate Email", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create two users
		testLicence := createTestLicence(t, "UPD-0002")
		userOne := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "dupemailuser1",
			Email:     "dupemailuser1@example.com",
			Password:  "hashedpassword",
			FirstName: "Dupemail",
			LastName:  "User1",
			Role:      "user",
			IsActive:  true,
		})

		userTwo := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "dupemailuser2",
			Email:     "dupemailuser2@example.com",
			Password:  "hashedpassword",
			FirstName: "Dupemail",
			LastName:  "User2",
			Role:      "user",
			IsActive:  true,
		})

		// Now build update params to change userTwo's email to userOne's email
		updateParams := updateUserParams{
			UserID: uuid.MustParse(userTwo.ID.String()),
			Email:  &userOne.Email,
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Put("/users/{userID}", h.UpdateUser)

		// Build and serve HTTP request
		rr := buildAndServeHttpRequest(http.MethodPut, "/users/"+userTwo.ID.String(), updateParams, r)

		// Assert that the response status code indicates a conflict due to duplicate email
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Update User - Invalid Role", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a test licence
		testLicence := createTestLicence(t, "UPD-0003")

		// Then, create a user to update
		createdUser := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "invalidroleupdateuser",
			Email:     "invalidroleupdateuser@example.com",
			Password:  "hashedpassword",
			FirstName: "InvalidRole",
			LastName:  "UpdateUser",
			Role:      "user",
			IsActive:  true,
		})

		// Now build update params with an invalid role
		updateParams := updateUserParams{
			UserID: uuid.MustParse(createdUser.ID.String()),
			Role:   helpers.Ptr("invalid"),
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Put("/users/{userID}", h.UpdateUser)

		rr := buildAndServeHttpRequest(http.MethodPut, "/users/"+createdUser.ID.String(), updateParams, r)

		// Validate response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Update User - Missing UserID", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// Now build update params without a userID
		updateParams := updateUserParams{
			Username: helpers.Ptr("newusername"),
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Put("/users/{userID}", h.UpdateUser)

		// Make the request
		rr := buildAndServeHttpRequest(http.MethodPut, "/users/", updateParams, r)

		// Assert that the response status code indicates not found due to missing userID in URL
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Update User - Invalid Email Format", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a licence
		testLicence := createTestLicence(t, "UPD-0005")

		// Then, create a user to update
		createdUser := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "invalidemailupdateuser",
			Email:     "invalid@emailupdate.com",
			Password:  "hashedpassword",
			FirstName: "Invalid",
			LastName:  "Email",
			Role:      "user",
			IsActive:  true,
		})

		// Now build update params with an invalid email format
		updateParams := updateUserParams{
			UserID: uuid.MustParse(createdUser.ID.String()),
			Email:  helpers.Ptr("invalid-email-format"),
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Put("/users/{userID}", h.UpdateUser)

		// Build and serve HTTP request
		rr := buildAndServeHttpRequest(http.MethodPut, "/users/"+createdUser.ID.String(), updateParams, r)

		// Assert that the response status code indicates a bad request due to invalid email format
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Update User - Invalid Licence", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a licence
		testLicence := createTestLicence(t, "UPD-0006")

		// Then, create a user to update
		createdUser := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "inactiveupdateuser",
			Email:     "inactiveupdateuser@example.com",
			Password:  "hashedpassword",
			FirstName: "Inactive",
			LastName:  "UpdateUser",
			Role:      "user",
			IsActive:  false,
		})

		// Now build update params with a non-existent licence ID
		updateParams := updateUserParams{
			UserID:    uuid.MustParse(createdUser.ID.String()),
			LicenceID: helpers.Ptr(uuid.New()),
		}

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Put("/users/{userID}", h.UpdateUser)

		// Build and serve HTTP request
		rr := buildAndServeHttpRequest(http.MethodPut, "/users/"+createdUser.ID.String(), updateParams, r)

		// Assert that the response status code indicates a bad request due to invalid licence
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Get User's Licence", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a licence
		testLicence := createTestLicence(t, "GET-0007")

		// Then, create a user associated with that licence
		createdUser := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "getlicenceuser",
			Email:     "getlicenceuser@example.com",
			Password:  "hashedpassword",
			FirstName: "Get",
			LastName:  "LicenceUser",
			Role:      "user",
		})

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Get("/users/{userID}/licence", h.GetLicence)

		// Build and serve HTTP request
		rr := buildAndServeHttpRequest(http.MethodGet, "/users/"+createdUser.ID.String()+"/licence", nil, r)

		// Assert the request was successful and response code is 200 OK
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Get User's Licence - Non-existent User", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// Generate a fake user ID (random UUID)
		fakeID := uuid.New().String()

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Get("/users/{userID}/licence", h.GetLicence)

		// Build and serve HTTP request
		rr := buildAndServeHttpRequest(http.MethodGet, "/users/"+fakeID+"/licence", nil, r)

		// Assert the response status code (404) indicates the user was not found
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Delete User", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// First, create a test licence
		testLicence := createTestLicence(t, "DEL-0001")

		// Then, create a user to delete
		createdUser := createTestUser(t, createUserParams{
			LicenceID: uuid.MustParse(testLicence.ID.String()),
			Username:  "deleteuser",
			Email:     "deleteuser@example.com",
			Password:  "hashedpassword",
			FirstName: "Delete",
			LastName:  "User",
			Role:      "user",
			IsActive:  true,
		})

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Delete("/users/{userID}", h.DeleteUser)

		// Now attempt to delete the user
		rr := buildAndServeHttpRequest(http.MethodDelete, "/users/"+createdUser.ID.String(), nil, r)

		// Assert the response status code indicates successful deletion (204 No Content)
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify user is deleted
		_, err := svc.GetUserById(ctx, uuid.MustParse(createdUser.ID.String()))

		// Assert that an error is returned indicating the user was not found
		assert.Error(t, err)
	})

	t.Run("Delete User - Non-existent User", func(t *testing.T) {
		t.Parallel() // Run test in parallel to speed up execution

		// Generate a fake user ID (random UUID)
		fakeID := uuid.New().String()

		// Create router
		r := chi.NewRouter()

		// Define route and handler
		r.Delete("/users/{userID}", h.DeleteUser)

		rr := buildAndServeHttpRequest(http.MethodDelete, "/users/"+fakeID, nil, r)

		// Assert the response status code indicates the user was not found (404 Not Found)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
