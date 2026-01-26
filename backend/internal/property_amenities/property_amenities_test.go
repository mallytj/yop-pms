package property_amenities

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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"

	hf "ollerod-pms/internal/helpers"
	mw "ollerod-pms/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDB      *pgxpool.Pool
	testQueries *repo.Queries
)

const (
	pathPrefix = "/property_amenities"
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

func TestPropertyAmenityFlow(t *testing.T) {
	ctx := context.Background()
	svc := NewService(*testQueries, testDB)
	h := NewHandler(svc)
	r := chi.NewRouter()

	r.Route(pathPrefix, func(r chi.Router) {
		r.Use(middleware.StripSlashes)
		r.Post("/", h.CreatePropertyAmenity)
		r.Get("/", h.ListPropertyAmenities)

		// Routes that require propertyAmenityID in URL
		r.Route("/{propertyAmenityID}", func(r chi.Router) {
			// Middleware to extract propertyAmenityID from URL and add to context
			r.Use(mw.PropertyAmenityCtx)
			r.Use(middleware.StripSlashes)

			r.Get("/", h.GetPropertyAmenityById)
			r.Put("/", h.UpdatePropertyAmenity)
			r.Delete("/", h.DeletePropertyAmenity)
			r.Get("/property", h.GetProperty)
			r.Get("/licence", h.GetLicence)
		})
	})

	t.Run("Create Property Amenity", func(t *testing.T) {
		// First, create a licence
		lic := hf.CreateTestLicence(t, "PAM-9999", testQueries)

		// Then, create a property
		property := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: lic.ID,
			Name:      "Test Property",
			Address:   "123 Test St, Test City",
		}, testQueries)

		// Now, create a property amenity
		payload := repo.CreatePropertyAmenityParams{
			PropertyID:  property.ID,
			Name:        "Free WiFi",
			Description: hf.ToPgText(hf.Ptr("High-speed wireless internet access")),
			ShortCode:   "WIFI",
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, pathPrefix, payload, r)

		// Check the response
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Check the property amenity in the database
		var createdAmenity repo.PropertyAmenity
		err := testDB.QueryRow(ctx, "SELECT property_id, name, description, short_code, is_active FROM property_amenities WHERE property_id=$1 AND name=$2",
			payload.PropertyID, payload.Name).Scan(
			&createdAmenity.PropertyID,
			&createdAmenity.Name,
			&createdAmenity.Description,
			&createdAmenity.ShortCode,
			&createdAmenity.IsActive,
		)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, payload.PropertyID, createdAmenity.PropertyID)
		assert.Equal(t, payload.Name, createdAmenity.Name)
		assert.Equal(t, payload.Description.String, createdAmenity.Description.String)
		assert.Equal(t, payload.ShortCode, createdAmenity.ShortCode)
		assert.Equal(t, payload.IsActive.Bool, createdAmenity.IsActive.Bool)
	})

	t.Run("Create Property Amenity - Invalid Name", func(t *testing.T) {
		// Attempt to create a test property
		lic := hf.CreateTestLicence(t, "PAM-3333", testQueries)
		property := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: lic.ID,
			Name:      "Test Property for Invalid Name",
			Address:   "123 Test St, Test City",
		}, testQueries)

		// Attempt to create a property amenity with an invalid name
		payload := repo.CreatePropertyAmenityParams{
			PropertyID:  property.ID,
			Name:        "A", // Invalid name (too short)
			Description: hf.ToPgText(hf.Ptr("Description")),
			ShortCode:   "SC",
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, pathPrefix, payload, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Property Amenity - Invalid ShortCode", func(t *testing.T) {
		// Attempt to create a test property
		lic := hf.CreateTestLicence(t, "PAM-4444", testQueries)
		property := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: lic.ID,
			Name:      "Test Property for Invalid ShortCode",
			Address:   "123 Test St, Test City",
		}, testQueries)

		// Attempt to create a property amenity with an invalid shortcode
		payload := repo.CreatePropertyAmenityParams{
			PropertyID:  property.ID,
			Name:        "Valid Name",
			Description: hf.ToPgText(hf.Ptr("Description")),
			ShortCode:   "S", // Invalid shortcode (too short)
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, pathPrefix, payload, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Property Amenity - Invalid Description", func(t *testing.T) {
		// Create test property
		lic := hf.CreateTestLicence(t, "PAM-5555", testQueries)
		property := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: lic.ID,
			Name:      "Test Property for Invalid Description",
			Address:   "123 Test St, Test City",
		}, testQueries)

		// Attempt to create a property amenity with an invalid description
		longDescription := ""
		for i := 0; i < 600; i++ {
			longDescription += "a"
		}

		payload := repo.CreatePropertyAmenityParams{
			PropertyID:  property.ID,
			Name:        "Valid Name",
			Description: hf.ToPgText(hf.Ptr(longDescription)), // Invalid description (too long)
			ShortCode:   "SCODE",
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, pathPrefix, payload, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Property Amenity - Related Entity Not Found", func(t *testing.T) {
		// Attempt to create a property amenity with a non-existent property ID
		fakeUUID := uuid.New()
		payload := repo.CreatePropertyAmenityParams{
			PropertyID:  hf.ToPgUUID(&fakeUUID), // Non-existent property ID
			Name:        "Valid Name",
			Description: hf.ToPgText(hf.Ptr("Valid Description")),
			ShortCode:   "SCODE",
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, pathPrefix, payload, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Property Amenity - Duplicate Entry", func(t *testing.T) {
		// First, create a licence
		lic := hf.CreateTestLicence(t, "PAM-8888", testQueries)

		// Then, create a property
		property := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: lic.ID,
			Name:      "Duplicate Test Property",
			Address:   "456 Test Ave, Test City",
		}, testQueries)

		// Create an initial property amenity
		hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			PropertyID:  property.ID,
			Name:        "Swimming Pool",
			Description: hf.ToPgText(hf.Ptr("Outdoor pool")),
			ShortCode:   "POOL",
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}, "PAM-8888", testQueries)

		// Attempt to create a duplicate property amenity
		duplicatePayload := repo.CreatePropertyAmenityParams{
			PropertyID:  property.ID,
			Name:        "Swimming Pool",
			Description: hf.ToPgText(hf.Ptr("Another pool")),
			ShortCode:   "POOL", // Same name as initial
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, pathPrefix, duplicatePayload, r)

		// Check the response
		assert.Equal(t, http.StatusConflict, rr.Code)
	})

	t.Run("Create Property Amenity - Same Shortcode, Different Licence", func(t *testing.T) {
		// First, create a licence
		lic1 := hf.CreateTestLicence(t, "PAM-7777", testQueries)
		lic2 := hf.CreateTestLicence(t, "PAM-7778", testQueries)

		// Then, create two properties under different licences
		property1 := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: lic1.ID,
			Name:      "Property One",
			Address:   "789 Test Blvd, Test City",
		}, testQueries)

		property2 := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: lic2.ID,
			Name:      "Property Two",
			Address:   "101 Test Rd, Test City",
		}, testQueries)

		// Create an initial property amenity for the first property
		hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			PropertyID:  property1.ID,
			Name:        "Gym",
			Description: hf.ToPgText(hf.Ptr("Fitness center")),
			ShortCode:   "GYM",
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}, "PAM-7777", testQueries)

		// Now, create a property amenity with the same shortcode for the second property
		payload := repo.CreatePropertyAmenityParams{
			PropertyID:  property2.ID,
			Name:        "Wellness Center",
			Description: hf.ToPgText(hf.Ptr("Spa and wellness services")),
			ShortCode:   "GYM", // Same shortcode as the first amenity
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPost, pathPrefix, payload, r)

		// Check the response
		assert.Equal(t, http.StatusCreated, rr.Code)
	})

	t.Run("List Property Amenities", func(t *testing.T) {
		// Create a property amenity to ensure there's at least one entry
		testAmenityOne := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "test",
			Name:        "Test",
			Description: hf.ToPgText(hf.Ptr("Test Description")),
		}, "PAM-0001", testQueries)

		// Create another property amenity
		testAmenityTwo := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "demo",
			Name:        "Demo",
			Description: hf.ToPgText(hf.Ptr("Demo Description")),
		}, "PAM-0001", testQueries)

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix, nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response body
		var amenities []repo.PropertyAmenity
		json.Unmarshal(rr.Body.Bytes(), &amenities)

		// Assertions
		assert.GreaterOrEqual(t, len(amenities), 2)

		// Check that the created amenities are in the list
		foundAmenityOne := false
		foundAmenityTwo := false

		for _, amenity := range amenities {
			if amenity.ID == testAmenityOne.ID {
				foundAmenityOne = true
			}
			if amenity.ID == testAmenityTwo.ID {
				foundAmenityTwo = true
			}
		}

		// Assertions
		assert.True(t, foundAmenityOne, "Test Amenity One should be in the list")
		assert.True(t, foundAmenityTwo, "Test Amenity Two should be in the list")
	})

	t.Run("List Property Amenities - No Entries", func(t *testing.T) {
		// Clear all property amenities from the database
		_, err := testDB.Exec(ctx, "DELETE FROM property_amenities")
		assert.NoError(t, err)

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix, nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response body
		var amenities []repo.PropertyAmenity
		json.Unmarshal(rr.Body.Bytes(), &amenities)

		// Assertions
		assert.Equal(t, 0, len(amenities), "Amenities list should be empty")
	})

	t.Run("Get Property Amenity By ID", func(t *testing.T) {
		// Create a test property amenity
		testAmenity := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "fetch",
			Name:        "Fetch Test",
			Description: hf.ToPgText(hf.Ptr("Fetch Description")),
		}, "PAM-0002", testQueries)

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+testAmenity.ID.String(), nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response body
		var fetchedAmenity repo.PropertyAmenity
		err := json.Unmarshal(rr.Body.Bytes(), &fetchedAmenity)
		assert.NoError(t, err)

		// Assertions
		assert.Equal(t, testAmenity.ID, fetchedAmenity.ID)
		assert.Equal(t, testAmenity.Name, fetchedAmenity.Name)
		assert.Equal(t, testAmenity.ShortCode, fetchedAmenity.ShortCode)
		assert.Equal(t, testAmenity.Description.String, fetchedAmenity.Description.String)
	})

	t.Run("Get Property Amenity By ID - Not Found", func(t *testing.T) {
		// Generate a random UUID that does not exist
		nonExistentID := uuid.New()

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+nonExistentID.String(), nil, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Get Property Amenity By ID - Invalid UUID", func(t *testing.T) {
		// Use an invalid UUID string
		invalidID := "invalid-uuid"

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+invalidID, nil, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Update Property Amenity", func(t *testing.T) {
		// Create a test property amenity
		testAmenity := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "update",
			Name:        "Update Test",
			Description: hf.ToPgText(hf.Ptr("Update Description")),
		}, "PAM-0003", testQueries)

		// Prepare update payload
		updatePayload := repo.UpdatePropertyAmenityParams{
			ID:          testAmenity.ID,
			Name:        hf.ToPgText(hf.Ptr("Updated Name")),
			Description: hf.ToPgText(hf.Ptr("Updated Description")),
			ShortCode:   hf.ToPgText(hf.Ptr("UPD")),
			IsActive:    hf.ToPgBool(hf.Ptr(false)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, pathPrefix+"/"+testAmenity.ID.String(), updatePayload, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response body
		var updatedAmenity repo.PropertyAmenity
		err := json.Unmarshal(rr.Body.Bytes(), &updatedAmenity)
		assert.NoError(t, err)

		// Assertions
		assert.Equal(t, updatePayload.Name.String, updatedAmenity.Name)
		assert.Equal(t, updatePayload.Description.String, updatedAmenity.Description.String)
		assert.Equal(t, updatePayload.ShortCode.String, updatedAmenity.ShortCode)
		assert.Equal(t, updatePayload.IsActive.Bool, updatedAmenity.IsActive.Bool)

		// Verify the update in the database
		var dbAmenity repo.PropertyAmenity
		err = testDB.QueryRow(ctx, "SELECT name, description, short_code, is_active FROM property_amenities WHERE id=$1",
			testAmenity.ID).Scan(
			&dbAmenity.Name,
			&dbAmenity.Description,
			&dbAmenity.ShortCode,
			&dbAmenity.IsActive,
		)
		assert.NoError(t, err)
		assert.Equal(t, updatePayload.Name.String, dbAmenity.Name)
		assert.Equal(t, updatePayload.Description.String, dbAmenity.Description.String)
		assert.Equal(t, updatePayload.ShortCode.String, dbAmenity.ShortCode)
		assert.Equal(t, updatePayload.IsActive.Bool, dbAmenity.IsActive.Bool)
	})

	t.Run("Update Property Amenity - No Fields to Update", func(t *testing.T) {
		// Create a test property amenity
		testAmenity := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "nofield",
			Name:        "No Field Test",
			Description: hf.ToPgText(hf.Ptr("No Field Description")),
		}, "PAM-0004", testQueries)

		// Prepare empty update payload
		updatePayload := repo.UpdatePropertyAmenityParams{
			ID: testAmenity.ID,
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, pathPrefix+"/"+testAmenity.ID.String(), updatePayload, r)
 
		// Check the response returns ok as no fields to update is now handled gracefully
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Update Property Amenity - Not Found", func(t *testing.T) {
		// Generate a random UUID that does not exist
		nonExistentID := uuid.New()

		// Prepare update payload
		updatePayload := repo.UpdatePropertyAmenityParams{
			ID:          hf.ToPgUUID(&nonExistentID),
			Name:        hf.ToPgText(hf.Ptr("Updated Name")),
			Description: hf.ToPgText(hf.Ptr("Updated Description")),
			ShortCode:   hf.ToPgText(hf.Ptr("UPD")),
			IsActive:    hf.ToPgBool(hf.Ptr(false)),
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, pathPrefix+"/"+nonExistentID.String(), updatePayload, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Update Property Amenity - Invalid ShortCode", func(t *testing.T) {
		// Create a test property amenity
		testAmenity := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "invshort",
			Name:        "Invalid Shortcode Test",
			Description: hf.ToPgText(hf.Ptr("Invalid Shortcode Description")),
		}, "PAM-0005", testQueries)

		// Prepare update payload with invalid shortcode
		updatePayload := repo.UpdatePropertyAmenityParams{
			ID:        testAmenity.ID,
			ShortCode: hf.ToPgText(hf.Ptr("A")), // Invalid shortcode (too short)
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, pathPrefix+"/"+testAmenity.ID.String(), updatePayload, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Update Property Amenity - Duplicate Entry", func(t *testing.T) {
		// Create a property as they must be the same
		lic := hf.CreateTestLicence(t, "PAM-6666", testQueries)
		property := hf.CreateTestProperty(t, repo.CreatePropertyParams{
			LicenceID: lic.ID,
			Name:      "Duplicate Update Test Property",
			Address:   "999 Test Ln, Test City",
		}, testQueries)

		// Create an initial property amenity
		hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			PropertyID:  property.ID,
			Name:        "Sauna",
			Description: hf.ToPgText(hf.Ptr("Relaxing sauna")),
			ShortCode:   "SAUN",
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}, "PAM-6666", testQueries)

		// Create another property amenity to be updated
		amenityToUpdate := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			PropertyID:  property.ID,
			Name:        "Steam Room",
			Description: hf.ToPgText(hf.Ptr("Hot steam room")),
			ShortCode:   "STEAM",
			IsActive:    hf.ToPgBool(hf.Ptr(true)),
		}, "PAM-6666", testQueries)

		// Prepare update payload to duplicate the existing amenity's name and shortcode
		updatePayload := repo.UpdatePropertyAmenityParams{
			ID:        amenityToUpdate.ID,
			Name:      hf.ToPgText(hf.Ptr("Sauna")), // Duplicate name
			ShortCode: hf.ToPgText(hf.Ptr("SAUN")),  // Duplicate shortcode
		}

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodPut, pathPrefix+"/"+amenityToUpdate.ID.String(), updatePayload, r)

		// Check the response
		assert.Equal(t, http.StatusConflict, rr.Code)
	})

	t.Run("Delete Property Amenity", func(t *testing.T) {
		// Create a test property amenity
		testAmenity := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "deltest",
			Name:        "Delete Test",
			Description: hf.ToPgText(hf.Ptr("Delete Description")),
		}, "PAM-0006", testQueries)

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodDelete, pathPrefix+"/"+testAmenity.ID.String(), nil, r)

		// Check the response
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify the amenity has been deleted from the database
		var count int
		err := testDB.QueryRow(ctx, "SELECT COUNT(*) FROM property_amenities WHERE id=$1", testAmenity.ID).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Delete Property Amenity - Not Found", func(t *testing.T) {
		// Generate a random UUID that does not exist
		nonExistentID := uuid.New()

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodDelete, pathPrefix+"/"+nonExistentID.String(), nil, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Delete Property Amenity - Invalid UUID", func(t *testing.T) {
		// Use an invalid UUID string
		invalidID := "invalid-uuid"

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodDelete, pathPrefix+"/"+invalidID, nil, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Get Licence for Property Amenity", func(t *testing.T) {
		// Create a test property amenity
		testAmenity := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "lictest",
			Name:        "Licence Test",
			Description: hf.ToPgText(hf.Ptr("Licence Description")),
		}, "PAM-0007", testQueries)

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+testAmenity.ID.String()+"/licence", nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response body
		var licence repo.Licence
		err := json.Unmarshal(rr.Body.Bytes(), &licence)
		assert.NoError(t, err)

		// Assertions
		assert.Equal(t, "PAM-0007", licence.LicenceKey)
	})

	t.Run("Get Licence for Property Amenity - Not Found", func(t *testing.T) {
		// Generate a random UUID that does not exist
		nonExistentID := uuid.New()

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+nonExistentID.String()+"/licence", nil, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Get Licence for Property Amenity - Invalid UUID", func(t *testing.T) {
		// Use an invalid UUID string
		invalidID := "invalid-uuid"

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+invalidID+"/licence", nil, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Get Property for Property Amenity", func(t *testing.T) {
		// Create a test property amenity
		testAmenity := hf.CreateTestPropertyAmenity(t, repo.CreatePropertyAmenityParams{
			ShortCode:   "proptest",
			Name:        "Property Test",
			Description: hf.ToPgText(hf.Ptr("Property Description")),
		}, "PAM-0008", testQueries)

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+testAmenity.ID.String()+"/property", nil, r)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response body
		var property repo.Property
		err := json.Unmarshal(rr.Body.Bytes(), &property)
		assert.NoError(t, err)

		// Assertions
		assert.Equal(t, testAmenity.PropertyID, property.ID)
	})

	t.Run("Get Property for Property Amenity - Not Found", func(t *testing.T) {
		// Generate a random UUID that does not exist
		nonExistentID := uuid.New()

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+nonExistentID.String()+"/property", nil, r)

		// Check the response
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Get Property for Property Amenity - Invalid UUID", func(t *testing.T) {
		// Use an invalid UUID string
		invalidID := "invalid-uuid"

		// Build and send the HTTP request
		rr := hf.BuildAndServeHttpRequest(http.MethodGet, pathPrefix+"/"+invalidID+"/property", nil, r)

		// Check the response
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
