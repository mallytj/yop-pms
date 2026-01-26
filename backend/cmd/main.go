package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"ollerod-pms/internal/env"

	appCfg "ollerod-pms/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	envCfg := *appCfg.NewConfig()
	ctx := context.Background()
	fmt.Println("Starting Ollerod PMS Backend...", envCfg.DBName, envCfg.DBHost, envCfg.DBPort, envCfg.DBUser, envCfg.DBPassword)

	cfg := config{
		addr: ":8080",
		db: dbConfig{
			dsn: env.GetEnv("DATABASE_DSN", "host="+envCfg.DBHost+" port="+envCfg.DBPort+" user="+envCfg.DBUser+" password="+envCfg.DBPassword+" dbname="+envCfg.DBName+" sslmode=disable"),
		},
	}

	// Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Database
	conn, err := pgxpool.New(ctx, cfg.db.dsn)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

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
