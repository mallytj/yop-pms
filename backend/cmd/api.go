package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	_ "ollerod-pms/cmd/docs"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/cache"
	appCfg "ollerod-pms/internal/config"
	"ollerod-pms/internal/events"
	"ollerod-pms/internal/handlers"
	"ollerod-pms/internal/service"
	"time"

	mw "ollerod-pms/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

type application struct {
	config config
	logger *slog.Logger
	db     *pgxpool.Pool
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()
	cfg := appCfg.NewConfig()

	// Initialise redis client for cache invalidation
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	// Just for dev
	cache.ClearAllCache(redisClient)

	ctx := context.Background()
	// Clear all cache on startup (optional, but ensures we don't serve stale data after a restart)
	iter := redisClient.Scan(ctx, 0, "planner:*", 100).Iterator()
	deleted := 0

	for iter.Next(ctx) {
		redisClient.Del(ctx, iter.Val())
		deleted++
	}

	if err := iter.Err(); err != nil {
		log.Fatal(err)
	}

	// Initialize cache invalidator and subscribe to reservation change events
	cacheInvalidator := cache.NewCacheInvalidator(redisClient, app.logger)

	listener := events.NewEventListener(app.config.db.dsn, app.logger)

	if err := listener.On("reservation_changes", cacheInvalidator.OnReservationChange); err != nil {
		log.Fatalf("failed to subscribe to reservation_change events: %v", err)
	}

	if err := listener.Start(); err != nil {
		log.Fatalf("failed to start event listener: %v", err)
	}
	// defer listener.Stop()

	// Global middlewares
	r.Use(middleware.RequestID) // important for rate limiting
	r.Use(middleware.RealIP)    // important for rate limiting and analytics and tracing
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)    // recover from crashes
	r.Use(middleware.StripSlashes) // remove trailing slashes from routes

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // Allow Svelte
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Swagger documentation endpoint
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// +=============================+
	// |                             |
	// |  __   ___     _   ___ ___   |
	// |  \ \ / / |   /_\ | _ \_ _|  |
	// |   \ V /| |  / _ \|  _/| |   |
	// |    \_/ |_| /_/ \_\_| |___|  |
	// |                             |
	// +=============================+
	r.Route("/v1", func(r chi.Router) {

		r.Use(mw.EnforcePropertyContext)

		plannerService := service.NewPlannerService(*repo.New(app.db), app.db)
		plannerHandler := handlers.NewPlannerHandler(plannerService, cacheInvalidator, app.logger)
		r.Get("/planner", plannerHandler.GetPlannerData)

		pricingService := service.NewPricingService(*repo.New(app.db), app.db)
		pricingHandler := handlers.NewPricingHandler(pricingService, cacheInvalidator, app.logger)
		r.Get("/rate-map", pricingHandler.GetRateMap)

		ratePlanService := service.NewRatePlanService(*repo.New(app.db), app.db)
		ratePlanHandler := handlers.NewRatePlanHandler(ratePlanService, cacheInvalidator, app.logger)
		r.Get("/rate-plans", ratePlanHandler.GetRatePlans)

		reservationService := service.NewReservationService(*repo.New(app.db), app.db)
		reservationHandler := handlers.NewReservationHandler(reservationService)
		r.Route("/reservation_item", func(r chi.Router) {
			r.Route("/{reservationItemID}", func(r chi.Router) {
				r.Use(mw.ReservationItemCtx)
				r.Put("/", func(w http.ResponseWriter, r *http.Request) {
					reservationHandler.UpdateReservationItem(w, r)
				})
			})
		})
	})
	return r
}

func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	log.Printf("server has started at addr %s", app.config.addr)
	fmt.Printf(`
	 ▗▄▄▖▗▄▄▄▖▗▄▄▖ ▗▖  ▗▖▗▄▄▄▖▗▄▄▖      ▗▄▄▖▗▄▄▄▖▗▄▖ ▗▄▄▖▗▄▄▄▖▗▄▄▄▖▗▄▄▄ 
	▐▌   ▐▌   ▐▌ ▐▌▐▌  ▐▌▐▌   ▐▌ ▐▌    ▐▌     █ ▐▌ ▐▌▐▌ ▐▌ █  ▐▌   ▐▌  █
	 ▝▀▚▖▐▛▀▀▘▐▛▀▚▖▐▌  ▐▌▐▛▀▀▘▐▛▀▚▖     ▝▀▚▖  █ ▐▛▀▜▌▐▛▀▚▖ █  ▐▛▀▀▘▐▌  █
	▗▄▄▞▘▐▙▄▄▖▐▌ ▐▌ ▝▚▞▘ ▐▙▄▄▖▐▌ ▐▌    ▗▄▄▞▘  █ ▐▌ ▐▌▐▌ ▐▌ █  ▐▙▄▄▖▐▙▄▄▀
	`)

	return srv.ListenAndServe()
}
