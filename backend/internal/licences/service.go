package licences

import (
	"context"
	"errors"
	"fmt"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	"ollerod-pms/internal/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
	ErrDuplicatedField               = errors.New("duplicated field error")
)

// PostgreSQL error codes
const (
	uniqueViolationCode     = "23505"
	foreignKeyViolationCode = "23503"
)

type svc struct {
	repo repo.Queries
	db   *pgxpool.Pool
}

type Service interface {
	ListLicences(ctx context.Context) ([]repo.Licence, error)
	GetLicenceById(ctx context.Context) (repo.Licence, error)
	GetUsersByID(ctx context.Context) ([]repo.User, error)
	CreateLicence(ctx context.Context, params repo.CreateLicenceParams) (repo.Licence, error)
	UpdateLicence(ctx context.Context, params repo.UpdateLicenceParams) (repo.Licence, error)
	DeleteLicence(ctx context.Context) error
}

func NewService(r repo.Queries, db *pgxpool.Pool) Service {
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

	// Perform the create operation
	licence, err := qtx.CreateLicence(ctx, params)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Handle unique violation error for licence key
			if pgErr.Code == uniqueViolationCode {
				return repo.Licence{}, ErrDuplicatedField
			}
		}
		return repo.Licence{}, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return repo.Licence{}, err
	}

	// Return the created licence
	return licence, nil
}

// ListLicences retrieves all licences from the database. (CRUD - Read)
func (s *svc) ListLicences(ctx context.Context) ([]repo.Licence, error) {
	// Perform the list operation
	licences, err := s.repo.ListLicences(ctx)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return licences, nil
}

// GetLicenceById retrieves a licence by its ID from the database. (CRUD - Read)
func (s *svc) GetLicenceById(ctx context.Context) (repo.Licence, error) {
	// Extract licence ID from context
	licenceID := helpers.GetLicenceID(ctx)

	// Perform the get operation
	licence, err := s.repo.GetLicenceByID(ctx, helpers.ToPgUUID(&licenceID))

	if err != nil {
		// Handle not found error
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.Licence{}, ErrLicenceNotFound
		}
		return repo.Licence{}, err
	}

	// Return the retrieved licence
	return licence, nil
}

// GetUsersByID retrieves all users associated with a given licence ID. (CRUD - Read)
func (s *svc) GetUsersByID(ctx context.Context) ([]repo.User, error) {
	// Extract licence ID from context
	licenceID := helpers.GetLicenceID(ctx)

	// Perform the get users by licence ID operation
	users, err := s.repo.GetUsersByLicenceID(ctx, helpers.ToPgUUID(&licenceID))
	if err != nil {
		// Handle not found error
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		// For other errors, return the error
		return nil, err
	}

	// Return the list of users
	return users, nil
}

// UpdateLicence updates an existing licence in the database. (CRUD - Update)
func (s *svc) UpdateLicence(ctx context.Context, params repo.UpdateLicenceParams) (repo.Licence, error) {
	// Extract licence ID from context
	licenceID := helpers.GetLicenceID(ctx)

	// Validate parameters
	if err := validateUpdateLicenceParams(params); err != nil {
		return repo.Licence{}, err
	}

	// Check if there is anything to update
	if params == (repo.UpdateLicenceParams{}) {
		return repo.Licence{}, nil // No fields to update
	}

	// Start transaction for updating licence
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Licence{}, err
	}

	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Set the ID for the update operation
	params.ID = helpers.ToPgUUID(&licenceID)

	// Perform the update
	licence, err := qtx.UpdateLicence(ctx, params)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.Licence{}, ErrLicenceNotFound
		}
		return repo.Licence{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.Licence{}, err
	}

	return licence, nil
}

// DeleteLicence deletes a licence from the database. (CRUD - Delete)
func (s *svc) DeleteLicence(ctx context.Context) error {
	// Extract licence ID from context
	licenceID := helpers.GetLicenceID(ctx)

	// Start transaction for deleting licence
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Perform the delete operation
	res, err := qtx.DeleteLicence(ctx, helpers.ToPgUUID(&licenceID))
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return ErrLicenceNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
