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

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexxcode1/yop-pms/internal/platform/cache"
	"github.com/lexxcode1/yop-pms/internal/platform/config"
	"github.com/lexxcode1/yop-pms/internal/platform/events"
	"github.com/lexxcode1/yop-pms/internal/platform/logging"
	yopOtel "github.com/lexxcode1/yop-pms/internal/platform/otel"
	"github.com/lexxcode1/yop-pms/internal/platform/worker"
	"github.com/redis/go-redis/v9"
)

type application struct {
	config *config.Config
	db     *pgxpool.Pool
	rdb    *redis.Client
	logger *slog.Logger
	cache  *cache.Client
}

// @title			Yop PMS Backend API
// @version			1.0
// @description		This is the backend API documentation for Yop PMS.
// @host			localhost:8080
// @BasePath		/v1
func main() {
	cfg := config.MustLoad()

	logger := logging.NewLogger(cfg.Environment)
	slog.SetDefault(logger)

	if err := run(cfg, logger); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, logger *slog.Logger) error {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create pgx pool with OTel tracer
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("unable to parse database URL", "error", err)
		return err
	}

	// Attach OTel pgx tracer
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
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

	// Setup OpenTelemetry
	otelShutdown, err := yopOtel.Setup(ctx, yopOtel.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.ServiceVersion,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		Environment:    cfg.Environment,
	})
	if err != nil {
		logger.Error("otel setup failed", "error", err)
		return err
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			logger.Error("otel shutdown failed", "error", err)
		}
	}()

	appCache := cache.New(rdb, "yop:", logger)

	// Events listener — dedicated connection outside the pool (LISTEN blocks the connection).
	// On reconnect, flush the entire cache to avoid serving stale data from the disconnect window.
	eventListener := events.New(cfg.DatabaseURL, logger, func() {
		if err := appCache.Invalidate(context.Background(), "yop:*"); err != nil {
			logger.Error("failed to flush cache on event listener reconnect", "error", err)
		}
	})

	eventListener.On("reservation_changes", cache.NewReservationChangeHandler(appCache, logger))
	eventListener.Start()
	defer eventListener.Stop()

	// Outbox worker — polls internal.outbox_events and dispatches registered handlers.
	// Route handlers enqueue events by inserting rows via SQLC; the worker processes them async.
	outboxWorker := worker.New(dbPool, logger, worker.Config{
		PollInterval: 5 * time.Second,
		BatchSize:    10,
		MaxRetries:   3,
	})
	// TODO: register domain handlers here as they are implemented, e.g.:
	//   outboxWorker.Register(worker.EventConfirmationEmail, smtp.HandleConfirmation(smtpClient))
	outboxWorker.Start()
	defer outboxWorker.Stop()

	app := &application{
		config: cfg,
		db:     dbPool,
		rdb:    rdb,
		logger: logger,
		cache:  appCache,
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
