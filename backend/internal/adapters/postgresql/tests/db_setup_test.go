package db_tests

import (
	"context"
	"database/sql"
	"log"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
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
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	// Point this to your actual migrations folder
	if err := goose.Up(db, "../migrations"); err != nil {
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
