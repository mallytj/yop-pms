package main

import (
	"log"
	"net/http"
	"time"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/users"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()
	// A good base middleware stack
	r.Use(middleware.RequestID) // important for rate limiting
	r.Use(middleware.RealIP)    // import for rate limiting and analytics and tracing
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer) // recover from crashes

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all good"))
	})

	userService := users.NewService(*repo.New(app.db), app.db)
	userHandler := users.NewHandler(userService)
	r.Route("/users", func(r chi.Router) {
		// Get all users
		r.Get("/", userHandler.ListUsers)

		// Get a single user by ID
		r.Get("/{userID}", userHandler.GetUserById)

		// Create a new user (using form data)
		r.Post("/", userHandler.CreateUser)

		// Update an existing user by ID (using form data)
		r.Put("/{userID}", userHandler.UpdateUser)

		// Delete a user by ID
		r.Delete("/{userID}", userHandler.DeleteUser)

		// Get user's licence by ID
		r.Get("/{userID}/licence", userHandler.GetLicence)

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

	return srv.ListenAndServe()
}

type application struct {
	config config
	// logger
	db *pgx.Conn
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string
}
