package licences

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/types"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	hf "ollerod-pms/internal/helpers"
	mw "ollerod-pms/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	connStr, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for the database to be ready
	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("database not ready after 30 seconds: %v", err)
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

// TestLicenceFlow tests the complete licence CRUD flow including creating, retrieving,
// updating, and deleting licences. It also tests various edge cases and error scenarios
// such as duplicate keys, invalid formats, and non-existent licences.
// Most tests run in parallel for better performance and are isolated from each other.
func TestLicenceFlow(t *testing.T) {
	ctx := context.Background()
	svc := NewService(*testQueries, testDB)
	h := NewHandler(svc)
	r := chi.NewRouter()

	r.Route("/licences", func(r chi.Router) {
		r.Post("/", h.CreateLicence)
	})

	// Routes that require licenceID in URL
	r.Route("/licences/{licenceID}", func(r chi.Router) {
		r.Use(mw.LicenceCtx) // Middleware to extract licenceID from URL and add to context
		r.Get("/", h.GetLicenceById)
		r.Put("/", h.UpdateLicence)
		r.Delete("/", h.DeleteLicence)
		r.Get("/users", h.GetUsersByID)
	})

	t.Run("Create Licence - Success", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Define valid parameters
		params := repo.CreateLicenceParams{
			LicenceKey:       "ABC-1234",
			OrganisationName: "The Grand London",
			ContactEmail:     "admin@grandlondon.com",
			LicenceNotes:     hf.ToPgText(hf.Ptr("Standard Licence")),
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, "/licences", params, r)

		// Assert response status code is 201 Created
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Parse response body into Licence struct
		var created repo.Licence
		err := json.Unmarshal(rr.Body.Bytes(), &created)

		// Assert no error and correct LicenceKey
		require.NoError(t, err)

		// Assert that the created licence has the expected LicenceKey
		assert.Equal(t, "ABC-1234", created.LicenceKey)

		// Verify in DB
		dbLicence, err := testQueries.GetLicenceByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "The Grand London", dbLicence.OrganisationName)
	})

	t.Run("Create Licence - Invalid Format (Regex Fail)", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Define invalid parameters
		params := repo.CreateLicenceParams{
			LicenceKey:       "invalid-key", // Should fail XXX-YYYY format
			OrganisationName: "Hotel",
			ContactEmail:     "test@test.com",
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, "/licences", params, r)

		// Assert response status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Licence - Key Already Exists", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a licence with a specific key
		hf.CreateTestLicence(t, "DUP-0001", testQueries)

		// Now, attempt to create another with the same key
		params := repo.CreateLicenceParams{
			LicenceKey:       "DUP-0001", // Duplicate key
			OrganisationName: "Another Organisation",
			ContactEmail:     "another@example.com",
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, "/licences", params, r)

		// Assert response status code is 409 Conflict due to duplicate key
		assert.Equal(t, http.StatusConflict, rr.Code)
	})

	t.Run("Get Licence By ID", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Setup: Create a seed licence directly via Repo
		lic := hf.CreateTestLicence(t, "GET-9999", testQueries)

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, "/licences/"+lic.ID.String(), nil, r)

		// Assert that we get a 200 OK
		assert.Equal(t, http.StatusOK, rr.Code)

		// Unmarshal into Licence struct
		var found repo.Licence
		err := json.Unmarshal(rr.Body.Bytes(), &found)
		require.NoError(t, err)

		// Verify that the found licence matches the created one
		assert.Equal(t, lic.LicenceKey, found.LicenceKey)
		assert.Equal(t, lic.ID, found.ID)
	})

	t.Run("Get Licence By ID - Not Found", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Use a random UUID that does not exist
		nonExistentID := uuid.New()

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, "/licences/"+nonExistentID.String(), nil, r)

		// Assert that we get a 404 Not Found
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Get Users By Licence ID", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Create a licence
		lic := hf.CreateTestLicence(t, "GET-6969", testQueries)

		// Parse licence ID to uuid.UUID
		licenceID := uuid.MustParse(lic.ID.String())

		// Create a user associated with that licence
		hf.CreateTestUser(t, types.CreateUserParams{
			LicenceID: licenceID,
			Username:  "testuser-getusers",
			Email:     "testuser-getusers@hotel.com",
			Password:  "hashedpassword",
			FirstName: "Test",
			LastName:  "User",
			Role:      "user",
			IsActive:  true,
		}, testQueries)

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, "/licences/"+licenceID.String()+"/users", nil, r)

		// Assert that we get a 200 OK and the correct user data
		assert.Equal(t, http.StatusOK, rr.Code)

		// Unmarshal into slice of Users structs
		var users []repo.User
		err := json.Unmarshal(rr.Body.Bytes(), &users)
		require.NoError(t, err)

		// Verify at least one user is returned
		assert.GreaterOrEqual(t, len(users), 1)

		// Find our specific user
		foundUser := false
		for _, user := range users {
			if user.Username == "testuser-getusers" {
				foundUser = true
				assert.Equal(t, "testuser-getusers@hotel.com", user.Email)
				break
			}
		}
		assert.True(t, foundUser, "Created user should be in the list")
	})

	t.Run("Get Users By Licence ID - No Users", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Create a licence with no users
		lic := hf.CreateTestLicence(t, "GET-0000", testQueries)

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, "/licences/"+uuid.UUID(lic.ID.Bytes).String()+"/users", nil, r)

		// Assert that we get a 200 OK
		assert.Equal(t, http.StatusOK, rr.Code)

		// Unmarshal into slice of Users structs
		var users []repo.User
		err := json.Unmarshal(rr.Body.Bytes(), &users)
		require.NoError(t, err)

		// Verify we have zero users returned
		assert.Len(t, users, 0)
	})

	t.Run("Update Licence", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Setup
		lic := hf.CreateTestLicence(t, "UPD-2222", testQueries)

		// Define update parameters
		updateParams := repo.UpdateLicenceParams{
			OrganisationName: "New Improved Name",
			ContactEmail:     "new@email.com",
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), updateParams, r)

		// Assert response status code is 200 OK
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify in DB
		updated, err := testQueries.GetLicenceByID(ctx, lic.ID)
		require.NoError(t, err)

		// Assert that the organisation name was updated
		assert.Equal(t, "New Improved Name", updated.OrganisationName)

		// Assert that the contact email was updated
		assert.Equal(t, "new@email.com", updated.ContactEmail)
	})

	t.Run("Update Licence - Not Found", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Define update parameters for a non-existent licence
		updateParams := repo.UpdateLicenceParams{
			OrganisationName: "Non Existent",
			ContactEmail:     "nonexistent@email.com",
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, "/licences/"+uuid.New().String(), updateParams, r)

		// Assert response status code is 404 Not Found
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Update Licence - Invalid Params", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Create a licence
		lic := hf.CreateTestLicence(t, "UPD-3333", testQueries)

		// Invalid update params
		updateParams := repo.UpdateLicenceParams{
			OrganisationName: "",            // Invalid: empty name
			ContactEmail:     "alsoinvalid", // Invalid email format
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), updateParams, r)

		// Assert response status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Check in DB that no changes were made
		dbLicence, err := testQueries.GetLicenceByID(ctx, lic.ID)
		require.NoError(t, err)
		assert.Equal(t, lic.OrganisationName, dbLicence.OrganisationName)
		assert.Equal(t, lic.ContactEmail, dbLicence.ContactEmail)
	})

	t.Run("Update Licence - No Fields", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Create a licence
		lic := hf.CreateTestLicence(t, "UPD-4444", testQueries)

		// Empty update params
		updateParams := repo.UpdateLicenceParams{}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), updateParams, r)

		// Assert response status code is 200 OK (no changes made)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Delete Licence", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Create a licence
		lic := hf.CreateTestLicence(t, "DEL-2222", testQueries)

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodDelete, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), nil, r)

		// Assert response status code is 204 No Content
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify Deletion in DB
		_, err := testQueries.GetLicenceByID(ctx, lic.ID)
		assert.Error(t, err)
	})

	t.Run("Delete Licence - Not Found", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Use a random UUID that does not exist
		nonExistentID := uuid.New()

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodDelete, "/licences/"+nonExistentID.String(), nil, r)

		// Assert response status code is 404 Not Found
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
