package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	// Adds a request ID into context of each request
	// May potentially be deprecated in lieu of
	// OpenTelemetry
	r.Use(middleware.RequestID)

	// Stores IP address for DDoS protection
	r.Use(middleware.RealIP)

	// Logs start and end of each request
	r.Use(middleware.Logger)

	// Recovers from panics, returns 500 when possible
	r.Use(middleware.Recoverer)

	// Strips trailing slashes e.g. /healthz/ => /healthz
	r.Use(middleware.StripSlashes)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // Allow Svelte
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", app.HealthHandler)

	// API Docs
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/v1", func(_ chi.Router) {
		// In future PR (Reservations), we will add domain routers here:
		// r.Mount("/bookings", booking.NewHandler(app.store).Routes())
		// r.Mount("/rooms", room.NewHandler(app.store).Routes())
	})

	return r
}

type HealthResponse struct {
	Status   string                   `json:"status" example:"ok"`
	Message  string                   `json:"message" example:"Server is running"`
	Version  string                   `json:"version" example:"1.0.0"`
	Services map[string]ServiceHealth `json:"services"`
}

type ServiceHealth struct {
	Status  string `json:"status" example:"ok"`
	Latency string `json:"latency" example:"5ms"`
	Error   string `json:"error,omitempty"`
}

// HealthHandler returns the server and all dependencies health status
// @Summary      Get Server Health
// @Description  Check if the server, database, and redis are alive
// @Tags         System
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      503  {object}  HealthResponse
// @Router       /healthz [get]
func (app *application) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	services := make(map[string]ServiceHealth)

	// Check Postgres
	postgresHealth := app.checkPostgres(ctx)
	services["postgres"] = postgresHealth

	// Check Redis
	redisHealth := app.checkRedis(ctx)
	services["redis"] = redisHealth

	// Determine overall status
	overallStatus := "ok"
	statusCode := http.StatusOK

	for _, service := range services {
		if service.Status != "ok" {
			overallStatus = "degraded"
			statusCode = http.StatusServiceUnavailable
			break
		}
	}

	resp := HealthResponse{
		Status:   overallStatus,
		Message:  "Yop API health check",
		Version:  "0.1.0",
		Services: services,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		app.logger.Error("failed to encode health response", "error", err)
	}
}

// checkPostgres checks if the database is accessible
func (app *application) checkPostgres(ctx context.Context) ServiceHealth {
	start := time.Now()

	err := app.db.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		app.logger.Error("postgres health check failed", "error", err)
		return ServiceHealth{
			Status: "down",
			Error:  err.Error(),
		}
	}

	return ServiceHealth{
		Status:  "ok",
		Latency: latency.String(),
	}
}

// checkRedis checks if redis is accessible
func (app *application) checkRedis(ctx context.Context) ServiceHealth {
	start := time.Now()

	err := app.rdb.Ping(ctx).Err()
	latency := time.Since(start)

	if err != nil {
		app.logger.Error("redis health check failed", "error", err)
		return ServiceHealth{
			Status: "down",
			Error:  err.Error(),
		}
	}

	return ServiceHealth{
		Status:  "ok",
		Latency: latency.String(),
	}
}
