package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // Allow Svelte
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", app.HealthHandler)

	// Swagger docs
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/v1", func(r chi.Router) {
		// In the next PR (Porting), we will add domain routers here:
		// r.Mount("/bookings", booking.NewHandler(app.store).Routes())
		// r.Mount("/rooms", room.NewHandler(app.store).Routes())
	})

	return r
}

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message" example:"Server is running"`
	Version string `json:"version" example:"1.0.0"`
}

// HealthHandler returns the server status
// @Summary      Get Server Health
// @Description  Check if the server, database, and redis are alive
// @Tags         System
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /healthz [get]
func (app *application) HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:  "ok",
		Message: "Yop API is alive and kicking",
		Version: "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Using a standard JSON encoder here for the skeleton
	json.NewEncoder(w).Encode(resp)
}
