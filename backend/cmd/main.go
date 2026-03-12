package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"ollerod-pms/internal/env"
	"ollerod-pms/internal/seeders"

	appCfg "ollerod-pms/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/swaggo/http-swagger" // http-swagger middleware
)

// @title			Yop PMS Backend API
// @version		1.0
// @description	This is the backend API documentation for Yop PMS.
// @contact.name	API Support
// @contact.url	http://www.yop-pms.com/support
// @contact.email	support@yop-pms.com
// @host			localhost:8080
// @BasePath		/v1
// @schemes		http
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Database
	conn, err := pgxpool.New(ctx, cfg.db.dsn)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	logger.Info("connected to database", "dsn", cfg.db.dsn)

	// Seed if -seed flag
	shouldSeed := flag.Bool("seed", false, "seed the database with initial data")
	flag.Parse()
	if *shouldSeed {
		slog.Info("seeding database with initial data...")
		seeder := seeders.NewSeeder(conn)
		if err := seeder.SeedPlannerData(ctx); err != nil {
			slog.Error("failed to seed database", "error", err)
			os.Exit(1)
		}
		slog.Info("database seeding completed")
		return
	}

	api := application{
		config: cfg,
		db:     conn,
		logger: logger,
	}
	if err := api.run(api.mount()); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
