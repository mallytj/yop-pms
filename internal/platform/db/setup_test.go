package db_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *pgxpool.Pool
var testUserDB *pgxpool.Pool

func TestMain(m *testing.M) {
	code := runTests(m)

	os.Exit(code)
}

func runTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	// Now this WILL run when runTests returns
	defer cancel()

	cleanup, err := setupGlobalTestDB(ctx)
	if err != nil {
		// Use log.Printf instead of Panicf here to allow proper cleanup if needed,
		// or just return a failure code.
		log.Printf("failed to setup test environment: %v", err)
		return 1
	}

	// Ensure cleanup is also deferred if setup was successful
	defer cleanup()

	return m.Run()
}

func setupGlobalTestDB(ctx context.Context) (func(), error) {
	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("pms_test"),
		postgres.WithUsername("admin"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, err
	}

	adminConnStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")

	db, _ := sql.Open("pgx", adminConnStr)
	if err := goose.Up(db, "../../../migrations"); err != nil {
		return nil, err
	}
	db.Close()

	testDB, err = pgxpool.New(ctx, adminConnStr)
	if err != nil {
		return nil, err
	}

	testUserDB, err = setupRestrictedPool(ctx, adminConnStr)
	if err != nil {
		return nil, err
	}

	return func() {
		testDB.Close()
		testUserDB.Close()
		_ = pgContainer.Terminate(context.Background())
	}, nil
}

func setupRestrictedPool(ctx context.Context, adminConnStr string) (*pgxpool.Pool, error) {
	// We grant standard DML permissions but NO superuser/bypassrls
	setupSQL := []string{
		`DROP ROLE IF EXISTS app_user;`,
		`CREATE ROLE app_user WITH LOGIN PASSWORD 'password';`,
		`GRANT CONNECT ON DATABASE pms_test TO app_user;`,
		`GRANT USAGE ON SCHEMA public TO app_user;`,
		`GRANT USAGE ON SCHEMA inventory TO app_user;`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA inventory TO app_user;`,
		`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;`,
		`ALTER DEFAULT PRIVILEGES IN SCHEMA inventory GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;`,
	}

	for _, q := range setupSQL {
		if _, err := testDB.Exec(ctx, q); err != nil {
			return nil, fmt.Errorf("failed to setup app_user permissions: %w", err)
		}
	}

	// Standard Postgres connection string: postgres://user:pass@host:port/db
	// We'll use a simple string replacement or url.Parse for production grade
	userConnStr := strings.Replace(adminConnStr, "admin:password", "app_user:password", 1)

	return pgxpool.New(ctx, userConnStr)
}
