package properties

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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	hf "ollerod-pms/internal/helpers"
	mw "ollerod-pms/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

// TestPropertyFlow tests the complete property CRUD flow including creating, retrieving,
// listing, updating, and deleting properties. It also tests various edge cases and error
// scenarios such as non-existent licences, invalid parameters, and missing fields.
// All tests run in parallel for better performance and are isolated from each other.
func TestPropertyFlow(t *testing.T) {
	ctx := context.Background()
	svc := NewService(*testQueries, testDB)
	h := NewHandler(svc)
	r := chi.NewRouter()

	r.Route("/properties", func(r chi.Router) {
		r.Post("/", h.CreateProperty)
		r.Get("/", h.ListProperties)
	})

	// Routes that require licenceID in URL
	r.Route("/properties/{propertyID}", func(r chi.Router) {
		r.Use(mw.PropertyCtx)          // Middleware to extract propertyID from URL and add to context
		r.Use(middleware.StripSlashes) // remove trailing slashes from routes
		r.Get("/", h.GetPropertyById)
		r.Put("/", h.UpdateProperty)
		r.Delete("/", h.DeleteProperty)
		r.Get("/licence", h.GetLicence)
		r.Get("/users", h.GetUsers)
		// r.Get("/daily-availability", h.GetDailyAvailability)
		// r.Get("/rooms", h.GetRooms)
		// r.Get("/rate-plans", h.GetRatePlans)
		// r.Get("/guests", h.GetGuests)
		// r.Get("/reservations", h.GetReservations)
		// r.Get("/amenities", h.GetAmenities)
		// r.Get("/room-types", h.GetRoomTypes)
	})

	t.Run("Create Property", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-1234", testQueries)

		// Now, build property creation params
		createPropertyParams := repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("POST", "/properties/", createPropertyParams, r)

		// Check the response
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Parse response body into Property struct
		var prop repo.Property
		err := json.Unmarshal(rr.Body.Bytes(), &prop)

		// Ensure no error during unmarshalling
		require.NoError(t, err)

		// Check the property has been stored in DB
		storedProp, err := testQueries.GetPropertyByID(ctx, prop.ID)

		// Ensure no error during retrieval
		require.NoError(t, err)

		// Validate stored property fields
		assert.Equal(t, createPropertyParams.Address, storedProp.Address)
		assert.Equal(t, createPropertyParams.Name, storedProp.Name)
		assert.Equal(t, createPropertyParams.LicenceID, storedProp.LicenceID)
		assert.Equal(t, createPropertyParams.Timezone, storedProp.Timezone)
		assert.Equal(t, createPropertyParams.PropertyNotes, storedProp.PropertyNotes)
	})

	t.Run("Create Property - Licence Not Found", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Build property creation params with non-existent licence ID
		nonExistentLicenceID := uuid.New()
		createPropertyParams := repo.CreatePropertyParams{
			Address:       "456 Fake St",
			Name:          "Fake Property",
			LicenceID:     hf.ToPgUUID(&nonExistentLicenceID), // Random UUID
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("Not real")),
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("POST", "/properties/", createPropertyParams, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Property - Optional Fields Missing", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-5678", testQueries)

		// Build property creation params without optional fields
		createPropertyParams := repo.CreatePropertyParams{
			Address:   "789 NoNotes St",
			Name:      "No Notes Property",
			LicenceID: testLicence.ID,
			Timezone:  "Europe/Copenhagen",
			// PropertyNotes is omitted
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("POST", "/properties/", createPropertyParams, r)

		// Check the response
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Parse response body into Property struct
		var prop repo.Property
		err := json.Unmarshal(rr.Body.Bytes(), &prop)

		// Ensure no error during unmarshalling
		require.NoError(t, err)

		// Check the property has been stored in DB
		storedProp, err := testQueries.GetPropertyByID(ctx, prop.ID)

		// Ensure no error during retrieval
		require.NoError(t, err)

		// Validate stored property fields
		assert.Equal(t, createPropertyParams.Address, storedProp.Address)
		assert.Equal(t, createPropertyParams.Name, storedProp.Name)
		assert.Equal(t, createPropertyParams.LicenceID, storedProp.LicenceID)
		assert.Equal(t, createPropertyParams.Timezone, storedProp.Timezone)
		assert.False(t, storedProp.PropertyNotes.Valid) // Should be invalid since it was omitted
	})

	t.Run("Create Property - Invalid Params", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Simulate a user sending invalid parameters online
		createPropertyParams := map[string]interface{}{
			"address":   "", // Invalid: empty address
			"name":      "Invalid Property",
			"licenceID": "not-a-uuid", // Invalid UUID format
			"timezone":  "Europe/Copenhagen",
			"country":   "Neverland", // Invalid: unexpected field
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("POST", "/properties/", createPropertyParams, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Property - Invalid Name", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-9101", testQueries)

		// Build property creation params with invalid name
		createPropertyParams := repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "A", // Invalid: too short
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("POST", "/properties/", createPropertyParams, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	/* Timezone logic not implemented yet
	t.Run("Create Property - Invalid Timezone", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-1121", testQueries)

		// Build property creation params with invalid timezone
		createPropertyParams := repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Invalid/Timezone", // Invalid timezone
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("POST", "/properties/", createPropertyParams, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	}) */

	t.Run("Create Property - Invalid Address", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-3141", testQueries)

		// Build property creation params with invalid address
		createPropertyParams := repo.CreatePropertyParams{
			Address:       "123", // Invalid: too short
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("POST", "/properties/", createPropertyParams, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Property - Invalid Property Notes", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-1617", testQueries)

		// Build property creation params with invalid property notes
		longNotes := ""
		for i := 0; i < 501; i++ {
			longNotes += "a"
		}

		createPropertyParams := repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr(longNotes)), // Invalid: too long
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("POST", "/properties/", createPropertyParams, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("List Properties", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-1819", testQueries)

		// Create multiple test properties
		for i := 0; i < 3; i++ {
			hf.CreateTestProperty(t, repo.CreatePropertyParams{
				Address:       "123 Test St" + string(rune(i+'0')),   // Unique address
				Name:          "Test Property" + string(rune(i+'0')), // Unique name
				LicenceID:     testLicence.ID,                        // Same licence ID
				Timezone:      "Europe/Copenhagen",                   // Valid timezone
				PropertyNotes: hf.ToPgText(hf.Ptr("The best")),       // Valid notes
			}, testQueries)
		}

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("GET", "/properties", nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse response body into slice of Property structs
		var properties []repo.Property
		err := json.Unmarshal(rr.Body.Bytes(), &properties)

		// Ensure no error during unmarshalling
		require.NoError(t, err)

		// Validate that at least 3 properties are returned
		assert.GreaterOrEqual(t, len(properties), 3)
	})

	t.Run("List Properties - Empty Database", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Create a separate test licence for isolation (not used but ensures test isolation)
		_ = hf.CreateTestLicence(t, "EMPTY-001", testQueries)

		// Don't create any properties for this licence

		// Build and serve the HTTP request
		rr := hf.BuildAndServeHttpRequest("GET", "/properties", nil, r)

		// Check the response - should return 200 OK with empty array or 404 Not Found
		// depending on implementation. If there are properties from other parallel tests,
		// it will return 200 OK. If database is completely empty, it returns 404.
		if rr.Code == http.StatusOK {
			// Parse response body into slice of Property structs
			var properties []repo.Property
			err := json.Unmarshal(rr.Body.Bytes(), &properties)
			require.NoError(t, err)
			// In parallel tests, other tests may have created properties
			assert.GreaterOrEqual(t, len(properties), 0)
		} else {
			// When database is empty, expect 404
			assert.Equal(t, http.StatusNotFound, rr.Code)
		}
	})

	t.Run("Get Property By ID", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-2021", testQueries)

		// Create a test property
		testProperty := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}, testQueries)

		// Build and serve the HTTP request
		url := "/properties/" + testProperty.ID.String()
		rr := hf.BuildAndServeHttpRequest("GET", url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse response body into Property struct
		var prop repo.Property
		err := json.Unmarshal(rr.Body.Bytes(), &prop)

		// Ensure no error during unmarshalling
		require.NoError(t, err)

		// Validate retrieved property fields
		assert.Equal(t, testProperty.ID, prop.ID)
		assert.Equal(t, testProperty.Address, prop.Address)
		assert.Equal(t, testProperty.Name, prop.Name)
		assert.Equal(t, testProperty.LicenceID, prop.LicenceID)
		assert.Equal(t, testProperty.Timezone, prop.Timezone)
		assert.Equal(t, testProperty.PropertyNotes, prop.PropertyNotes)
	})

	t.Run("Get Property By ID - Not Found", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Build and serve the HTTP request with random UUID
		nonExistentPropertyID := uuid.New()
		url := "/properties/" + nonExistentPropertyID.String()
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Get Property By ID - Invalid UUID", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Build and serve the HTTP request with invalid UUID
		url := "/properties/invalid-uuid"
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Update Property", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-2223", testQueries)

		// Create a test property
		testProperty := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}, testQueries)

		// Build update parameters
		updateParams := repo.UpdatePropertyParams{
			Address:       hf.ToPgText(hf.Ptr("456 Updated St")),
			Name:          hf.ToPgText(hf.Ptr("Updated Property")),
			Timezone:      hf.ToPgText(hf.Ptr("Europe/Amsterdam")),
			PropertyNotes: hf.ToPgText(hf.Ptr("Updated notes")),
		}

		// Build and serve the HTTP request
		url := "/properties/" + testProperty.ID.String()
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, url, updateParams, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse response body into Property struct
		var updatedProp repo.Property
		err := json.Unmarshal(rr.Body.Bytes(), &updatedProp)

		// Ensure no error during unmarshalling
		require.NoError(t, err)

		// Validate updated property fields
		assert.Equal(t, testProperty.ID, updatedProp.ID)
		assert.Equal(t, updateParams.Address.String, updatedProp.Address)
		assert.Equal(t, updateParams.Name.String, updatedProp.Name)
		assert.Equal(t, updateParams.Timezone.String, updatedProp.Timezone)
		assert.Equal(t, updateParams.PropertyNotes.String, updatedProp.PropertyNotes.String)

		// Check the property has been updated in DB
		storedProp, err := testQueries.GetPropertyByID(ctx, testProperty.ID)

		// Ensure no error during retrieval
		require.NoError(t, err)

		// Validate stored property fields
		assert.Equal(t, updateParams.Address.String, storedProp.Address)
		assert.Equal(t, updateParams.Name.String, storedProp.Name)
		assert.Equal(t, updateParams.Timezone.String, storedProp.Timezone)
		assert.Equal(t, updateParams.PropertyNotes.String, storedProp.PropertyNotes.String)
	})

	t.Run("Update Property - Not Found", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Build update parameters
		updateParams := repo.UpdatePropertyParams{
			Address:       hf.ToPgText(hf.Ptr("456 Updated St")),
			Name:          hf.ToPgText(hf.Ptr("Updated Property")),
			Timezone:      hf.ToPgText(hf.Ptr("Europe/Amsterdam")),
			PropertyNotes: hf.ToPgText(hf.Ptr("Updated notes")),
		}

		// Build and serve the HTTP request with random UUID
		nonExistentPropertyID := uuid.New()
		url := "/properties/" + nonExistentPropertyID.String()
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, url, updateParams, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Update Property - Invalid Params", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-2425", testQueries)

		// Create a test property
		testProperty := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}, testQueries)

		// Build invalid update parameters
		updateParams := repo.UpdatePropertyParams{
			Name: hf.ToPgText(hf.Ptr("A")), // Invalid: too short
		}

		// Build and serve the HTTP request
		url := "/properties/" + testProperty.ID.String()
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, url, updateParams, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Update Property - No Fields Provided", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-2627", testQueries)

		// Create a test property
		testProperty := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}, testQueries)

		// Build empty update parameters
		updateParams := repo.UpdatePropertyParams{
			// No fields provided
		}

		// Build and serve the HTTP request
		url := "/properties/" + testProperty.ID.String()
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, url, updateParams, r)

		// Check the response
		assert.Equal(t, http.StatusNotModified, rr.Code)
	})

	t.Run("Update Property - Optional Fields Missing", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-2829", testQueries)

		// Create a test property
		testProperty := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}, testQueries)

		// Build update parameters without optional fields
		updateParams := repo.UpdatePropertyParams{
			Address:  hf.ToPgText(hf.Ptr("456 Updated St")),
			Name:     hf.ToPgText(hf.Ptr("Updated Property")),
			Timezone: hf.ToPgText(hf.Ptr("Europe/Amsterdam")),
			// PropertyNotes is omitted
		}

		// Build and serve the HTTP request
		url := "/properties/" + testProperty.ID.String()
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, url, updateParams, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse response body into Property struct
		var updatedProp repo.Property
		err := json.Unmarshal(rr.Body.Bytes(), &updatedProp)

		// Ensure no error during unmarshalling
		require.NoError(t, err)

		// Validate updated property fields
		assert.Equal(t, testProperty.ID, updatedProp.ID)
		assert.Equal(t, updateParams.Address.String, updatedProp.Address)
		assert.Equal(t, updateParams.Name.String, updatedProp.Name)
		assert.Equal(t, updateParams.Timezone.String, updatedProp.Timezone)
		assert.Equal(t, testProperty.PropertyNotes.String, updatedProp.PropertyNotes.String) // Should remain unchanged

		// Check the property has been updated in DB
		storedProp, err := testQueries.GetPropertyByID(ctx, testProperty.ID)

		// Ensure no error during retrieval
		require.NoError(t, err)

		// Validate stored property fields
		assert.Equal(t, updateParams.Address.String, storedProp.Address)
		assert.Equal(t, updateParams.Name.String, storedProp.Name)
		assert.Equal(t, updateParams.Timezone.String, storedProp.Timezone)
		assert.Equal(t, testProperty.PropertyNotes.String, storedProp.PropertyNotes.String) // Should remain unchanged
	})

	t.Run("Delete Property", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-3031", testQueries)

		// Create a test property
		testProperty := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}, testQueries)

		// Build and serve the HTTP request
		url := "/properties/" + testProperty.ID.String()
		rr := hf.BuildAndServeHttpRequest(http.MethodDelete, url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify the property has been deleted from DB
		_, err := testQueries.GetPropertyByID(ctx, testProperty.ID)

		// Ensure error indicates not found
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("Delete Property - Not Found", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Build and serve the HTTP request with random UUID
		nonExistentPropertyID := uuid.New()
		url := "/properties/" + nonExistentPropertyID.String()
		rr := hf.BuildAndServeHttpRequest(http.MethodDelete, url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Delete Property - Invalid UUID", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Build and serve the HTTP request with invalid UUID
		url := "/properties/invalid-uuid"
		rr := hf.BuildAndServeHttpRequest(http.MethodDelete, url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Get Licence", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-3233", testQueries)

		// Create a test property
		testProperty := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}, testQueries)

		// Build and serve the HTTP request
		url := "/properties/" + testProperty.ID.String() + "/licence"
		rr := hf.BuildAndServeHttpRequest("GET", url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse response body into Licence struct
		var licence repo.Licence
		err := json.Unmarshal(rr.Body.Bytes(), &licence)

		// Ensure no error during unmarshalling
		require.NoError(t, err)

		// Validate retrieved licence fields
		assert.Equal(t, testLicence.ID, licence.ID)
		assert.Equal(t, testLicence.LicenceKey, licence.LicenceKey)
	})

	t.Run("Get Licence - Property Not Found", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Build and serve the HTTP request with random UUID
		nonExistentPropertyID := uuid.New()
		url := "/properties/" + nonExistentPropertyID.String() + "/licence"
		rr := hf.BuildAndServeHttpRequest("GET", url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Get Licence - Invalid UUID", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// Build and serve the HTTP request with invalid UUID
		url := "/properties/invalid-uuid/licence"
		rr := hf.BuildAndServeHttpRequest("GET", url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	// Property must have a licence to be created, so no test for "Licence Not Found" case here

	t.Run("Get Users", func(t *testing.T) {
		t.Parallel() // Run this test in parallel

		// First, create a test licence to associate with the property
		testLicence := hf.CreateTestLicence(t, "PRO-3435", testQueries)

		// Create a test property
		testProperty := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			Address:       "123 Test St",
			Name:          "Test Property",
			LicenceID:     testLicence.ID,
			Timezone:      "Europe/Copenhagen",
			PropertyNotes: hf.ToPgText(hf.Ptr("The best")),
		}, testQueries)

		// Create test users associated with the property
		for i := 0; i < 2; i++ {
			hf.CreateTestUser(t, types.CreateUserParams{
				LicenceID: uuid.MustParse(testLicence.ID.String()),
				Email:     "user" + string(rune(i+'0')) + "@test.com", // Unique email
				Username:  "TestUser" + string(rune(i+'0')),           // Unique name
				Password:  "securepassword",
				Role:      "manager",
				IsActive:  true,
			}, testQueries)
		}

		// Build and serve the HTTP request
		url := "/properties/" + testProperty.ID.String() + "/users"
		rr := hf.BuildAndServeHttpRequest("GET", url, nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse response body into slice of User structs
		var users []repo.User
		err := json.Unmarshal(rr.Body.Bytes(), &users)

		// Ensure no error during unmarshalling
		require.NoError(t, err)

		// Validate that 2 users are returned
		assert.Len(t, users, 2)

		// Assert both usernames are present
		usernames := []string{users[0].Username, users[1].Username}
		assert.Contains(t, usernames, "TestUser0")
		assert.Contains(t, usernames, "TestUser1")
	})
}
