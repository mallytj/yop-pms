package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/licences"
	"ollerod-pms/internal/properties"
	"ollerod-pms/internal/users"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	mw "ollerod-pms/internal/middleware"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.RequestID) // important for rate limiting
	r.Use(middleware.RealIP)    // important for rate limiting and analytics and tracing
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)    // recover from crashes
	r.Use(middleware.StripSlashes) // remove trailing slashes from routes

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
		r.Use(mw.UserCtx) // Middleware to extract userID from URL and add to context

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
		r.Use(mw.LicenceCtx) // Middleware to extract licenceID from URL and add to context

		// Create a new licence (using form data)
		r.Post("/", licenceHandler.CreateLicence)

		// Get all licences
		r.Get("/", licenceHandler.ListLicences)

		// Get a single licence by ID
		r.Get("/{licenceID}", licenceHandler.GetLicenceById)

		// Update an existing licence by ID (using form data)
		r.Put("/{licenceID}", licenceHandler.UpdateLicence)

		// Delete a licence by ID
		r.Delete("/{licenceID}", licenceHandler.DeleteLicence)

		// Users by licence ID
		r.Route("/{licenceID}/users", func(r chi.Router) {
			r.Use(mw.UserCtx) // Middleware to extract userID from URL and add to context
			// Get all users by licence ID
			r.Get("/", licenceHandler.GetUsersByID)
		})
	})

	/*
	  ____                            _   _
	 |  _ \ _ __ ___  _ __   ___ _ __| |_(_) ___  ___
	 | |_) | '__/ _ \| '_ \ / _ \ '__| __| |/ _ \/ __|
	 |  __/| | | (_) | |_) |  __/ |  | |_| |  __/\__ \
	 |_|   |_|  \___/| .__/ \___|_|   \__|_|\___||___/
	                 |_|
	*/
	propertyService := properties.NewService(*repo.New(app.db), app.db)
	propertyHandler := properties.NewHandler(propertyService)
	r.Route("/properties", func(r chi.Router) {
		// Create a new property
		r.Post("/", propertyHandler.CreateProperty)

		// Get all properties
		r.Get("/", propertyHandler.ListProperties)

		r.Route("/{propertyID}", func(r chi.Router) {
			r.Use(mw.PropertyCtx) // Middleware to extract propertyID from URL and add to context

			// Get a single property by ID
			r.Get("/{propertyID}", propertyHandler.GetPropertyById)

			// Update an existing property by ID
			r.Put("/{propertyID}", propertyHandler.UpdateProperty)

			// Delete a property by ID
			r.Delete("/{propertyID}", propertyHandler.DeleteProperty)

			// Get properties licence by propertyID
			r.Get("/{propertyID}/licence", propertyHandler.GetLicence)

			// Get properties users by propertyID
			r.Get("/{propertyID}/users", propertyHandler.GetUsers)

			// Get properties room types by propertyID
			// r.Get("/{propertyID}/roomtypes", propertyHandler.GetRoomTypes)

			// Get properties amenities by propertyID
			// r.Get("/{propertyID}/amenities", propertyHandler.GetAmenities)

			// Get property reservations by propertyID
			// r.Get("/{propertyID}/reservations", propertyHandler.GetReservations)

			// Get property rooms by propertyID
			// r.Get("/{propertyID}/rooms", propertyHandler.GetRooms)

			// Get property rate plans by propertyID
			// r.Get("/{propertyID}/rateplans", propertyHandler.GetRatePlans)

			// Get property guests by propertyID
			// r.Get("/{propertyID}/guests", propertyHandler.GetGuests)

			// Get property daily availability by propertyID
			// Returns availability matrix for the next 365 days
			// r.Get("/{propertyID}/availability", propertyHandler.GetDailyAvailability)
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

type application struct {
	config config
	// logger
	db *pgxpool.Pool
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string
}
