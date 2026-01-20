package properties

import (
	"context"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	hf "ollerod-pms/internal/helpers"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type svc struct {
	repo repo.Queries
	db   *pgxpool.Pool
}

type Service interface {
	// Define service methods here
}

func NewService(r repo.Queries, db *pgxpool.Pool) Service {
	return &svc{
		repo: r,
		db:   db,
	}
}

func (s *svc) CreateProperty(ctx context.Context, params repo.CreatePropertyParams) (repo.Property, error) {
	// Validate parameters
	if err := validateCreatePropertyParams(params); err != nil {
		return repo.Property{}, err
	}

	// Start a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Property{}, err
	}
	// Ensure the transaction is rolled back in case of an error
	defer tx.Rollback(ctx)

	// Create a repository instance with the transaction
	qtx := s.repo.WithTx(tx)

	// Perform the create operation
	property, err := qtx.CreateProperty(ctx, params)
	if err != nil {
		return repo.Property{}, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return repo.Property{}, err
	}

	// Return the created property
	return property, nil
}

func (s *svc) UpdateProperty(ctx context.Context, propertyID uuid.UUID, params repo.UpdatePropertyParams) (repo.Property, error) {
	// Validate parameters
	if err := validateUpdatePropertyParams(params); err != nil {
		return repo.Property{}, err
	}

	// Start transaction for updating property
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Property{}, err
	}
	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Set the ID as a pgtype.UUID for the update operation
	params.ID = hf.ToPgUUID(&propertyID)

	// Perform the update operation
	property, err := qtx.UpdateProperty(ctx, params)
	if err != nil {
		// Check if the error is due to no rows being found
		if err == pgx.ErrNoRows {
			return repo.Property{}, ErrPropertyNotFound
		}

		return repo.Property{}, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return repo.Property{}, err
	}

	// Return the updated property
	return property, nil
}

func (s *svc) DeleteProperty(ctx context.Context, propertyID uuid.UUID) error {
	// Start transaction for deleting property
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Perform the delete operation
	res, err := qtx.DeleteProperty(ctx, hf.ToPgUUID(&propertyID))
	if err != nil {
		return err
	}

	// Check if any rows were affected
	if res.RowsAffected() == 0 {
		return ErrPropertyNotFound
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *svc) ListProperties(ctx context.Context, licenceID uuid.UUID) ([]repo.Property, error) {
	// Get the list of properties from the repository
	properties, err := s.repo.ListProperties(ctx, hf.ToPgUUID(&licenceID))
	if err != nil {
		return nil, err
	}

	// Return the list of properties
	return properties, nil
}

func (s *svc) GetPropertyById(ctx context.Context, propertyID uuid.UUID) (repo.Property, error) {
	// Get the property from the repository
	property, err := s.repo.GetPropertyByID(ctx, hf.ToPgUUID(&propertyID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return repo.Property{}, ErrPropertyNotFound
		}
		return repo.Property{}, err
	}

	// Return the property
	return property, nil
}

func (s *svc) GetDailyAvailability(ctx context.Context, propertyID uuid.UUID) ([][]int32, error) {
	// Implementation will come here
	return nil, nil
}

func (s *svc) GetRooms(ctx context.Context, propertyID uuid.UUID) ([]repo.Room, error) {
	return nil, nil
}

func (s *svc) GetRatePlans(ctx context.Context, propertyID uuid.UUID) ([]repo.RatePlan, error) {
	return nil, nil
}

func (s *svc) GetGuests(ctx context.Context, propertyID uuid.UUID) ([]repo.Guest, error) {
	// Implementation here
	return nil, nil
}

func (s *svc) GetReservations(ctx context.Context, propertyID uuid.UUID) ([]repo.Reservation, error) {
	// Implementation here
	return nil, nil
}

func (s *svc) GetAmenities(ctx context.Context, propertyID uuid.UUID) ([]repo.PropertyAmenity, error) {
	return nil, nil
}

func (s *svc) GetRoomTypes(ctx context.Context, propertyID uuid.UUID) ([]repo.RoomType, error) {
	return nil, nil
}

func (s *svc) GetLicence(ctx context.Context, propertyID uuid.UUID) (repo.Licence, error) {
	// Get the licence associated with the property
	licence, err := s.repo.GetLicenceByPropertyID(ctx, hf.ToPgUUID(&propertyID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return repo.Licence{}, ErrLicenceNotFound
		}
		return repo.Licence{}, err
	}

	// Return the licence
	return licence, nil
}
