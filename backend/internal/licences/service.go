package licences

import (
	"context"
	"errors"
	"fmt"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type svc struct {
	repo repo.Queries
	db   *pgx.Conn
}

type Service interface {
	ListLicences(ctx context.Context) ([]repo.Licence, error)
	GetLicenceById(ctx context.Context, licenceID uuid.UUID) (repo.Licence, error)
	CreateLicence(ctx context.Context, params repo.CreateLicenceParams) (repo.Licence, error)
}

func NewService(r repo.Queries, db *pgx.Conn) Service {
	return &svc{
		repo: r,
		db:   db,
	}
}

func (s *svc) ListLicences(ctx context.Context) ([]repo.Licence, error) {
	licences, err := s.repo.ListLicences(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing licences: %w", err)
	}

	return licences, nil
}

func (s *svc) GetLicenceById(ctx context.Context, licenceID uuid.UUID) (repo.Licence, error) {
	licence, err := s.repo.GetLicenceByID(ctx, pgtype.UUID{Bytes: licenceID, Valid: true})

	if err != nil {
		return repo.Licence{}, fmt.Errorf("error getting licence by ID: %w", err)
	}

	return licence, nil
}

func (s *svc) CreateLicence(ctx context.Context, params repo.CreateLicenceParams) (repo.Licence, error) {
	// Validate parameters
	if err := validateCreateLicenceParams(params); err != nil {
		return repo.Licence{}, fmt.Errorf("invalid create licence parameters: %w", err)
	}

	// Start transaction for creating licence
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Licence{}, fmt.Errorf("error starting transaction: %w", err)
	}
	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	licence, err := qtx.CreateLicence(ctx, params)

	if err != nil {
		return repo.Licence{}, fmt.Errorf("error creating licence: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.Licence{}, fmt.Errorf("error committing transaction: %w", err)
	}
	
	return licence, nil
	
}

