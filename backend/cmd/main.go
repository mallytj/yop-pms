package main

import (
	"context"
	"log/slog"
	"os"

	"ollerod-pms/internal/env"

	"github.com/jackc/pgx/v5"
)

var (
	dbName     = env.GetEnv("DB_NAME", "hotel_pms")
	dbHost     = env.GetEnv("DB_HOST", "localhost")
	dbPort     = env.GetEnv("DB_PORT", "5433")
	dbUser     = env.GetEnv("DB_USER", "nil")
	dbPassword = env.GetEnv("DB_PASSWORD", "nil")
)

func main() {
	ctx := context.Background()

	cfg := config{
		addr: ":8080",
		db: dbConfig{
			dsn: env.GetEnv("DATABASE_DSN", "host="+dbHost+" port="+dbPort+" user="+dbUser+" password="+dbPassword+" dbname="+dbName+" sslmode=disable"),
		},
	}

	// Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Database
	conn, err := pgx.Connect(ctx, cfg.db.dsn)
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)

	logger.Info("connected to database", "dsn", cfg.db.dsn)

	api := application{
		config: cfg,
		db:     conn,
	}
	if err := api.run(api.mount()); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
