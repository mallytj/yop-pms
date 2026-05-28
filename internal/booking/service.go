package booking

// Service implements the reservation domain business logic.
// Handlers call Service methods; Service calls SQLC-generated store.Queries
// via ExecuteTx for transactional consistency.
//
// Methods are split across files by domain area:
//   service_create.go   — CreateReservation, createReservationInTx, ConfirmReservation
//   service_read.go     — GetReservation, ListReservations, UpdateMetadata
//   service_helpers.go  — converter helpers and shared utilities
//   service_stubs.go    — Phase 7 stubs (not yet implemented)
//   availability.go     — CheckAvailability, InvalidateAvailabilityCache, conflictCheck
//   state_machine.go   — ValidateReservationTransition

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/lexxcode1/yop-pms/internal/store"
)

// Service implements the reservation domain business logic.
// Handlers call Service methods; Service calls SQLC-generated store.Queries
// via ExecuteTx for transactional consistency.
type Service struct {
	pool *pgxpool.Pool
	q    *store.Queries
	rdb  *redis.Client
	log  *slog.Logger
}

// NewService creates a new booking Service.
func NewService(pool *pgxpool.Pool, q *store.Queries, rdb *redis.Client, log *slog.Logger) *Service {
	return &Service{
		pool: pool,
		q:    q,
		rdb:  rdb,
		log:  log,
	}
}
