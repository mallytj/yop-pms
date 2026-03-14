package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexxcode1/yop-pms/internal/platform/config"
	"github.com/redis/go-redis/v9"
)

type application struct {
	config *config.Config
	db     *pgxpool.Pool
	rdb    *redis.Client
	logger *slog.Logger
}

// @title			Yop PMS Backend API
// @version			1.0
// @description		This is the backend API documentation for Yop PMS.
// @host			localhost:8080
// @BasePath		/v1
func main() {
	cfg := config.MustLoad()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	if err := run(cfg, logger); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, logger *slog.Logger) error {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("unable to connect to database", "error", err)
		return err
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		logger.Error("postgres connection failed", "error", err)
		return err
	}
	logger.Info("postgres connected")

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("redis connection failed", "error", err)
		return err
	}
	logger.Info("redis connected")

	if cfg.Environment == "dev" {
		if err := rdb.FlushDB(ctx).Err(); err != nil {
			logger.Error("failed to flush redis", "error", err)
			return err
		}
		logger.Info("redis cache cleared for development")
	}

	app := &application{
		config: cfg,
		db:     dbPool,
		rdb:    rdb,
		logger: logger,
	}

	srv := &http.Server{
		Addr:         ":" + app.config.Port,
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	errChan := make(chan error, 1)
	go func() {
		logger.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen and serve failed", "error", err)
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		logger.Info("shutting down gracefully...")
	}

	// Give active requests 5 seconds to finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		return fmt.Errorf("server shutdown failed: %v", err)
	}

	logger.Info("server stopped")

	return nil
}
