package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexxcode1/ollerod-pms/backend/internal/models"
	"github.com/lexxcode1/ollerod-pms/backend/internal/service"
	"github.com/lexxcode1/ollerod-pms/backend/internal/store/postgres"
)

type Server struct {
	bookingService *service.BookingService
}

func main() {
	// Database connection string
	dbURL := getEnv("DATABASE_URL", "postgres://pms_user:pms_password@localhost:5432/pms_db?sslmode=disable")

	// Create database connection pool
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Test database connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	log.Println("Successfully connected to database")

	// Initialize repositories
	bookingRepo := postgres.NewBookingRepository(pool)
	roomRepo := postgres.NewRoomRepository(pool)

	// Initialize services
	bookingService := service.NewBookingService(bookingRepo, roomRepo)

	// Create server
	server := &Server{
		bookingService: bookingService,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/api/bookings", server.handleBookings)
	mux.HandleFunc("/api/bookings/", server.handleBookingByID)

	// Create HTTP server
	port := getEnv("PORT", "8080")
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s\n", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	log.Println("Server exited")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleBookings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createBooking(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBookingByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	idStr := r.URL.Path[len("/api/bookings/"):]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getBooking(w, r, id)
	case http.MethodPatch:
		s.updateBookingStatus(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GuestID  string `json:"guest_id"`
		RoomID   string `json:"room_id"`
		CheckIn  string `json:"check_in"`
		CheckOut string `json:"check_out"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	guestID, err := uuid.Parse(req.GuestID)
	if err != nil {
		http.Error(w, "Invalid guest ID", http.StatusBadRequest)
		return
	}

	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	checkIn, err := time.Parse(time.RFC3339, req.CheckIn)
	if err != nil {
		http.Error(w, "Invalid check-in date", http.StatusBadRequest)
		return
	}

	checkOut, err := time.Parse(time.RFC3339, req.CheckOut)
	if err != nil {
		http.Error(w, "Invalid check-out date", http.StatusBadRequest)
		return
	}

	booking := &models.Booking{
		GuestID:  guestID,
		RoomID:   roomID,
		CheckIn:  checkIn,
		CheckOut: checkOut,
	}

	if err := s.bookingService.CreateBooking(r.Context(), booking); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(booking)
}

func (s *Server) getBooking(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	booking, err := s.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(booking)
}

func (s *Server) updateBookingStatus(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Action string `json:"action"` // "confirm" or "cancel"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var err error
	switch req.Action {
	case "confirm":
		err = s.bookingService.ConfirmBooking(r.Context(), id)
	case "cancel":
		err = s.bookingService.CancelBooking(r.Context(), id)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
