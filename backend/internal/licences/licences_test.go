package licences

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDB      *pgx.Conn
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
	testDB, err = pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatal(err)
	}
	testQueries = repo.New(testDB)

	code := m.Run()

	testDB.Close(ctx)
	pgContainer.Terminate(ctx)
	os.Exit(code)
}

func TestLicenceFlow(t *testing.T) {
	ctx := context.Background()
	svc := NewService(*testQueries, testDB)
	h := NewHandler(svc)

	t.Run("Create Licence - Success", func(t *testing.T) {
		params := repo.CreateLicenceParams{
			LicenceKey:       "ABC-1234",
			OrganisationName: "The Grand London",
			ContactEmail:     "admin@grandlondon.com",
			LicenceNotes:     pgtype.Text{String: "Standard Licence", Valid: true},
		}

		body, _ := json.Marshal(params)
		req := httptest.NewRequest(http.MethodPost, "/licences", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		h.CreateLicence(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var created repo.Licence
		err := json.Unmarshal(rr.Body.Bytes(), &created)
		require.NoError(t, err)
		assert.Equal(t, "ABC-1234", created.LicenceKey)
	})

	t.Run("Create Licence - Invalid Format (Regex Fail)", func(t *testing.T) {
		params := repo.CreateLicenceParams{
			LicenceKey:       "invalid-key", // Should fail XXX-YYYY
			OrganisationName: "Hotel",
			ContactEmail:     "test@test.com",
		}

		body, _ := json.Marshal(params)
		req := httptest.NewRequest(http.MethodPost, "/licences", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		h.CreateLicence(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Create Licence - Key Already Exists", func(t *testing.T) {
		// First, create a licence with a specific key
		_, err := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
			LicenceKey:       "DUP-0001",
			OrganisationName: "Duplicate Hotel",
			ContactEmail:     "duplicate@hotel.com",
		})
		require.NoError(t, err)

		// Now, attempt to create another with the same key
		params := repo.CreateLicenceParams{
			LicenceKey:       "DUP-0001", // Duplicate key
			OrganisationName: "Another Hotel",
			ContactEmail:     "another@hotel.com",
		}

		body, _ := json.Marshal(params)
		req := httptest.NewRequest(http.MethodPost, "/licences", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		h.CreateLicence(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		// assert.Contains(t, rr.Body.String(), "duplicate key value violates unique constraint")
	})

	t.Run("Get Licence By ID", func(t *testing.T) {
		// Setup: Create a seed licence directly via Repo
		lic, err := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
			LicenceKey:       "GET-9999",
			OrganisationName: "Getter Hotel",
			ContactEmail:     "get@hotel.com",
		})
		require.NoError(t, err)

		// Create Chi router to handle URL params
		r := chi.NewRouter()
		r.Get("/licences/{licenceID}", h.GetLicenceById)

		req := httptest.NewRequest(http.MethodGet, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var found repo.Licence
		json.Unmarshal(rr.Body.Bytes(), &found)
		assert.Equal(t, lic.LicenceKey, found.LicenceKey)
	})

	t.Run("Get Licence By ID - Not Found", func(t *testing.T) {
		// Create Chi router to handle URL params
		r := chi.NewRouter()
		r.Get("/licences/{licenceID}", h.GetLicenceById)

		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/licences/"+nonExistentID.String(), nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("Get Users By Licence ID", func(t *testing.T) {
		// Setup: Create a seed licence and user directly via Repo
		lic, err := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
			LicenceKey:       "USER-1234",
			OrganisationName: "User Hotel",
			ContactEmail:     "user@hotel.com",
		})

		require.NoError(t, err)

		_, err = testQueries.CreateUser(ctx, repo.CreateUserParams{
			LicenceID:    lic.ID,
			Username:     "testuser",
			Email:        "testuser@hotel.com",
			PasswordHash: "hashedpassword",
			FirstName:    "Test",
			LastName:     "User",
			Role:         "user",
			IsActive:     pgtype.Bool{Bool: true, Valid: true},
		})

		require.NoError(t, err)

		// Create Chi router to handle URL params
		r := chi.NewRouter()
		r.Get("/licences/{licenceID}/users", h.GetUsersByID)

		req := httptest.NewRequest(http.MethodGet, "/licences/"+uuid.UUID(lic.ID.Bytes).String()+"/users", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		// Assert that we get a 200 OK and the correct user data
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse response
		var users []repo.User
		json.Unmarshal(rr.Body.Bytes(), &users)

		// Verify we got the expected user
		assert.Len(t, users, 1)
		assert.Equal(t, "testuser", users[0].Username)
	})

	t.Run("Update Licence", func(t *testing.T) {
		// Setup
		lic, _ := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
			LicenceKey:       "UPD-1111",
			OrganisationName: "Old Name",
			ContactEmail:     "old@email.com",
		})

		updateParams := repo.UpdateLicenceParams{
			OrganisationName: "New Improved Name",
			ContactEmail:     "new@email.com",
		}

		body, _ := json.Marshal(updateParams)

		r := chi.NewRouter()
		r.Put("/licences/{licenceID}", h.UpdateLicence)

		req := httptest.NewRequest(http.MethodPut, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), bytes.NewReader(body))
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify in DB
		updated, _ := testQueries.GetLicenceByID(ctx, lic.ID)
		assert.Equal(t, "New Improved Name", updated.OrganisationName)
	})

	t.Run("Update Licence - Not Found", func(t *testing.T) {
		updateParams := repo.UpdateLicenceParams{
			OrganisationName: "Non Existent",
			ContactEmail:     "nonexistent@email.com",
		}

		body, _ := json.Marshal(updateParams)

		r := chi.NewRouter()
		r.Put("/licences/{licenceID}", h.UpdateLicence)

		req := httptest.NewRequest(http.MethodPut, "/licences/"+uuid.New().String(), bytes.NewReader(body))
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("Update Licence - Invalid Params", func(t *testing.T) {
		// Setup
		lic, _ := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
			LicenceKey:       "UPD-3333",
			OrganisationName: "Valid Name",
			ContactEmail:     "valid@email.com",
		})

		updateParams := repo.UpdateLicenceParams{
			OrganisationName: "",            // Invalid: empty name
			ContactEmail:     "alsoinvalid", // Invalid email format
		}

		body, _ := json.Marshal(updateParams)

		r := chi.NewRouter()
		r.Put("/licences/{licenceID}", h.UpdateLicence)

		req := httptest.NewRequest(http.MethodPut, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), bytes.NewReader(body))
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Update Licence - No Fields", func(t *testing.T) {
		// Setup
		lic, _ := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
			LicenceKey:       "UPD-4444",
			OrganisationName: "Another Valid Name",
			ContactEmail:     "anothervalid@email.com",
		})

		updateParams := repo.UpdateLicenceParams{}

		body, _ := json.Marshal(updateParams)

		r := chi.NewRouter()

		r.Put("/licences/{licenceID}", h.UpdateLicence)

		req := httptest.NewRequest(http.MethodPut, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), bytes.NewReader(body))
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Delete Licence", func(t *testing.T) {
		// Setup
		lic, _ := testQueries.CreateLicence(ctx, repo.CreateLicenceParams{
			LicenceKey:       "DEL-2222",
			OrganisationName: "To Be Deleted",
			ContactEmail:     "delete@email.com",
		})

		r := chi.NewRouter()
		r.Delete("/licences/{licenceID}", h.DeleteLicence)

		req := httptest.NewRequest(http.MethodDelete, "/licences/"+uuid.UUID(lic.ID.Bytes).String(), nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify Deletion
		_, err := testQueries.GetLicenceByID(ctx, lic.ID)
		assert.Error(t, err)
	})

	t.Run("Delete Licence - Not Found", func(t *testing.T) {
		r := chi.NewRouter()
		r.Delete("/licences/{licenceID}", h.DeleteLicence)

		nonExistentID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/licences/"+nonExistentID.String(), nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
