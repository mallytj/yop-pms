package licences

import (
	"context"
	"errors"
	"fmt"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	"ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrUserNotFound                  = errors.New("user not found")
	ErrLicenceNotFound               = errors.New("licence not found")
	ErrUpdatingLicence               = errors.New("error updating licence in database")
	ErrCreatingLicenceFailed         = errors.New("creating licence failed")
	ErrInvalidCreateLicenceParams    = errors.New("invalid create licence parameters")
	ErrListingLicencesFailed         = errors.New("listing licences failed")
	ErrDeletingLicenceFailed         = errors.New("deleting licence failed")
	ErrGettingUsersByLicenceIDFailed = errors.New("getting users by licence ID failed")
)

type svc struct {
	repo repo.Queries
	db   *pgx.Conn
}

type Service interface {
	ListLicences(ctx context.Context) ([]repo.Licence, error)
	GetLicenceById(ctx context.Context, licenceID uuid.UUID) (repo.Licence, error)
	GetUsersByID(ctx context.Context, licenceID uuid.UUID) ([]repo.User, error)
	CreateLicence(ctx context.Context, params repo.CreateLicenceParams) (repo.Licence, error)
	UpdateLicence(ctx context.Context, licenceID uuid.UUID, params repo.UpdateLicenceParams) (repo.Licence, error)
	DeleteLicence(ctx context.Context, licenceID uuid.UUID) error
}

func NewService(r repo.Queries, db *pgx.Conn) Service {
	return &svc{
		repo: r,
		db:   db,
	}
}

// CreateLicence creates a new licence in the database. (CRUD - Create)
func (s *svc) CreateLicence(ctx context.Context, params repo.CreateLicenceParams) (repo.Licence, error) {
	// Validate parameters
	if err := validateCreateLicenceParams(params); err != nil {
		return repo.Licence{}, fmt.Errorf("%w: %v", ErrInvalidCreateLicenceParams, err)
	}

	// Start transaction for creating licence
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Licence{}, fmt.Errorf("%w, %v", helpers.ErrStartingTx, err)
	}
	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	licence, err := qtx.CreateLicence(ctx, params)

	if err != nil {
		return repo.Licence{}, fmt.Errorf("%w: %v", ErrCreatingLicenceFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.Licence{}, fmt.Errorf("%w: %v", helpers.ErrCommitingTx, err)
	}

	return licence, nil
}

// ListLicences retrieves all licences from the database. (CRUD - Read)
func (s *svc) ListLicences(ctx context.Context) ([]repo.Licence, error) {
	licences, err := s.repo.ListLicences(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrListingLicencesFailed, err)
	}

	return licences, nil
}

// GetLicenceById retrieves a licence by its ID from the database. (CRUD - Read)
func (s *svc) GetLicenceById(ctx context.Context, licenceID uuid.UUID) (repo.Licence, error) {
	licence, err := s.repo.GetLicenceByID(ctx, pgtype.UUID{Bytes: licenceID, Valid: true})

	if err != nil {
		return repo.Licence{}, fmt.Errorf("%w: %v", ErrLicenceNotFound, err)
	}

	return licence, nil
}

// GetUsersByID retrieves all users associated with a given licence ID. (CRUD - Read)
func (s *svc) GetUsersByID(ctx context.Context, licenceID uuid.UUID) ([]repo.User, error) {
	users, err := s.repo.GetUsersByLicenceID(ctx, pgtype.UUID{Bytes: licenceID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("%w: %v", ErrGettingUsersByLicenceIDFailed, err)
	}

	return users, nil
}

// UpdateLicence updates an existing licence in the database. (CRUD - Update)
func (s *svc) UpdateLicence(ctx context.Context, licenceID uuid.UUID, params repo.UpdateLicenceParams) (repo.Licence, error) {
	// Validate parameters
	if err := validateUpdateLicenceParams(params); err != nil {
		return repo.Licence{}, fmt.Errorf("invalid update licence parameters: %w", err)
	}

	if params == (repo.UpdateLicenceParams{}) {
		return repo.Licence{}, nil // No fields to update
	}

	// Start transaction for updating licence
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Licence{}, fmt.Errorf("%w: %v", helpers.ErrStartingTx, err)
	}

	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Set the ID for the update operation
	params.ID = pgtype.UUID{Bytes: licenceID, Valid: true}

	// Perform the update
	licence, err := qtx.UpdateLicence(ctx, params)

	if err != nil {
		return repo.Licence{}, fmt.Errorf("%w: %v", ErrUpdatingLicence, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.Licence{}, fmt.Errorf("%w: %v", helpers.ErrCommitingTx, err)
	}

	return licence, nil
}

// DeleteLicence deletes a licence from the database. (CRUD - Delete)
func (s *svc) DeleteLicence(ctx context.Context, licenceID uuid.UUID) error {
	// Start transaction for deleting licence
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", helpers.ErrStartingTx, err)
	}
	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Perform the delete operation
	res, err := qtx.DeleteLicence(ctx, pgtype.UUID{Bytes: licenceID, Valid: true})
	if err != nil {

		return fmt.Errorf("%w: %v", ErrDeletingLicenceFailed, err)
	}

	if res.RowsAffected() == 0 {
		return ErrLicenceNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: %v", helpers.ErrCommitingTx, err)
	}

	return nil
}
