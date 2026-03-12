package integration_tests

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/cache"
)

var (
	testDB        *pgxpool.Pool
	testQueries   *repo.Queries
	testRedis     *redis.Client
	testMiniRedis *miniredis.Miniredis
	testCache     *cache.CacheInvalidator
	testLogger    *slog.Logger
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// --- Postgres Container ---
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
		log.Fatalf("failed to start postgres container: %s", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate postgres container: %s", err)
		}
	}()

	connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")

	// Run migrations
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	if err := goose.Up(db, "../adapters/postgresql/migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}
	db.Close()

	testDB, err = pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatal(err)
	}
	testQueries = repo.New(testDB)

	// --- Miniredis (in-process Redis) ---
	testMiniRedis, err = miniredis.Run()
	if err != nil {
		log.Fatalf("failed to start miniredis: %s", err)
	}
	defer testMiniRedis.Close()

	testRedis = redis.NewClient(&redis.Options{Addr: testMiniRedis.Addr()})
	testCache = cache.NewCacheInvalidator(testRedis, testLogger)

	code := m.Run()

	testDB.Close()
	testRedis.Close()
	os.Exit(code)
}
