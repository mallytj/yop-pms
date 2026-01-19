package main

import (
	"log"
	"net/http"
	"time"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/licences"
	"ollerod-pms/internal/users"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()
	// A good base middleware stack
	r.Use(middleware.RequestID) // important for rate limiting
	r.Use(middleware.RealIP)    // important for rate limiting and analytics and tracing
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)    // recover from crashes
	r.Use(middleware.StripSlashes) // remove trailing slashes from routes

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all good"))
	})

	/*
	  _   _
	 | | | |___  ___ _ __ ___
	 | | | / __|/ _ \ '__/ __|
	 | |_| \__ \  __/ |  \__ \
	  \___/|___/\___|_|  |___/
	*/
	userService := users.NewService(*repo.New(app.db), app.db)
	userHandler := users.NewHandler(userService)
	r.Route("/users", func(r chi.Router) {
		// Create a new user (using form data)
		r.Post("/", userHandler.CreateUser)

		// Get all users
		r.Get("/", userHandler.ListUsers)

		// Get user's licence by userID
		r.Get("/{userID}/licence", userHandler.GetLicence)

		// Get a single user by ID
		r.Get("/{userID}", userHandler.GetUserById)

		// Update an existing user by ID (using form data)
		r.Put("/{userID}", userHandler.UpdateUser)

		// Delete a user by ID
		r.Delete("/{userID}", userHandler.DeleteUser)

	})

	/*
	  _     _
	 | |   (_) ___ ___ _ __  ___  ___  ___
	 | |   | |/ __/ _ \ '_ \/ __|/ _ \/ __|
	 | |___| | (_|  __/ | | \__ \  __/\__ \
	 |_____|_|\___\___|_| |_|___/\___||___/
	*/
	licenceService := licences.NewService(*repo.New(app.db), app.db)
	licenceHandler := licences.NewHandler(licenceService)
	r.Route("/licences", func(r chi.Router) {
		// Create a new licence (using form data)
		r.Post("/", licenceHandler.CreateLicence)

		// Get all licences
		r.Get("/", licenceHandler.ListLicences)

		// Get a single licence by ID
		r.Get("/{licenceID}", licenceHandler.GetLicenceById)

		// Get all users by licence ID
		r.Get("/{licenceID}/users", licenceHandler.GetUsersByID)

		// Update an existing licence by ID (using form data)
		r.Put("/{licenceID}", licenceHandler.UpdateLicence)

		// Delete a licence by ID
		r.Delete("/{licenceID}", licenceHandler.DeleteLicence)
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
