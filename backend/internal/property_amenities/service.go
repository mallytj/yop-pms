package property_amenities

import (
	"context"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	hf "ollerod-pms/internal/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	CreatePropertyAmenity(ctx context.Context, params repo.CreatePropertyAmenityParams) (repo.PropertyAmenity, error)
	ListPropertyAmenities(ctx context.Context) ([]repo.PropertyAmenity, error)
	GetPropertyAmenityById(ctx context.Context) (repo.PropertyAmenity, error)
	UpdatePropertyAmenity(ctx context.Context, params repo.UpdatePropertyAmenityParams) (repo.PropertyAmenity, error)
	DeletePropertyAmenity(ctx context.Context) error
	GetProperty(ctx context.Context) (repo.Property, error)
	GetLicence(ctx context.Context) (repo.Licence, error)
}

type svc struct {
	repo repo.Queries
	db   *pgxpool.Pool
}

func NewService(r repo.Queries, db *pgxpool.Pool) Service {
	return &svc{
		repo: r,
		db:   db,
	}
}

// CreatePropertyAmenity creates a new property amenity in the database. (CRUD - Create)
func (s *svc) CreatePropertyAmenity(ctx context.Context, params repo.CreatePropertyAmenityParams) (repo.PropertyAmenity, error) {
	// Validate parameters
	if err := validateCreatePropertyAmenityParams(params); err != nil {
		return repo.PropertyAmenity{}, err
	}

	// Start a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.PropertyAmenity{}, err
	}
	// Ensure the transaction is rolled back in case of an error
	defer tx.Rollback(ctx)

	// Create a repository instance with the transaction
	qtx := s.repo.WithTx(tx)

	// Perform the create operation
	amenity, err := qtx.CreatePropertyAmenity(ctx, params)
	if err != nil {
		// If propertyID does not exist, return related entity not found error
		if hf.CheckErrorCode(err, hf.ForeignKeyViolationCode) {
			return repo.PropertyAmenity{}, hf.ErrRelatedEntityNotFound
		}

		// If unique constraint is violated, return duplicated field error
		// for example, if the amenity name must be unique
		if hf.CheckErrorCode(err, hf.UniqueViolationCode) {
			return repo.PropertyAmenity{}, hf.ErrDuplicatedField
		}

		return repo.PropertyAmenity{}, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return repo.PropertyAmenity{}, err
	}

	// Return the created property amenity
	return amenity, nil

}

// ListPropertyAmenities retrieves all property amenities from the database. (CRUD - Read)
func (s *svc) ListPropertyAmenities(ctx context.Context) ([]repo.PropertyAmenity, error) {
	// Retrieve the list of property amenities from the repository
	amenities, err := s.repo.ListPropertyAmenities(ctx)
	if err != nil {
		// If no rows are found, return an empty slice
		if err == pgx.ErrNoRows {
			return []repo.PropertyAmenity{}, ErrPropertyAmenityNotFound
		}
		return nil, err
	}

	// If no amenities are found, return an empty slice with specific error for handler to interpret
	if len(amenities) == 0 {
		return []repo.PropertyAmenity{}, ErrPropertyAmenityNotFound
	}

	// Return the list of property amenities
	return amenities, nil
}

// GetPropertyAmenityById retrieves a property amenity by its ID from the database. (CRUD - Read)
func (s *svc) GetPropertyAmenityById(ctx context.Context) (repo.PropertyAmenity, error) {
	// Extract property amenity ID from context
	propertyAmenityID := hf.GetPropertyAmenityID(ctx)

	// Perform the get operation
	amenity, err := s.repo.GetPropertyAmenityByID(ctx, hf.ToPgUUID(&propertyAmenityID))

	if err != nil {
		// Handle not found error
		if err == pgx.ErrNoRows {
			return repo.PropertyAmenity{}, ErrPropertyAmenityNotFound
		}
		return repo.PropertyAmenity{}, err
	}

	// Return the retrieved property amenity
	return amenity, nil
}

// UpdatePropertyAmenity updates an existing property amenity in the database. (CRUD - Update)
func (s *svc) UpdatePropertyAmenity(ctx context.Context, params repo.UpdatePropertyAmenityParams) (repo.PropertyAmenity, error) {
	// Get the property amenity ID from context
	propertyAmenityID := hf.GetPropertyAmenityID(ctx)

	// Start a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.PropertyAmenity{}, err
	}
	// Ensure the transaction is rolled back in case of an error
	defer tx.Rollback(ctx)

	// Create a repository instance with the transaction
	qtx := s.repo.WithTx(tx)

	// Set the ID in the update parameters
	// to identify which property amenity to update
	params.ID = hf.ToPgUUID(&propertyAmenityID)

	// Perform the update operation
	amenity, err := qtx.UpdatePropertyAmenity(ctx, params)
	if err != nil {
		// Handle not found error
		if err == pgx.ErrNoRows {
			return repo.PropertyAmenity{}, ErrPropertyAmenityNotFound
		}

		// Handle unique constraint violation
		if hf.CheckErrorCode(err, hf.UniqueViolationCode) {
			return repo.PropertyAmenity{}, hf.ErrDuplicatedField
		}

		// Handle foreign key violation
		if hf.CheckErrorCode(err, hf.ForeignKeyViolationCode) {
			return repo.PropertyAmenity{}, hf.ErrRelatedEntityNotFound
		}

		return repo.PropertyAmenity{}, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return repo.PropertyAmenity{}, err
	}

	// Return the updated property amenity
	return amenity, nil
}

// DeletePropertyAmenity deletes a property amenity from the database. (CRUD - Delete)
func (s *svc) DeletePropertyAmenity(ctx context.Context) error {
	// Get the property amenity ID from context
	propertyAmenityID := hf.GetPropertyAmenityID(ctx)

	// Start a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	// Ensure the transaction is rolled back in case of an error
	defer tx.Rollback(ctx)

	// Create a repository instance with the transaction
	qtx := s.repo.WithTx(tx)

	// Perform the delete operation
	res, err := qtx.DeletePropertyAmenity(ctx, hf.ToPgUUID(&propertyAmenityID))
	if err != nil {
		// Handle not found error
		if err == pgx.ErrNoRows {
			return ErrPropertyAmenityNotFound
		}
		return err
	}

	// If no rows were affected, the property amenity was not found
	if res.RowsAffected() == 0 {
		return ErrPropertyAmenityNotFound
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Return nil on successful deletion
	return nil
}

// GetProperty retrieves the property associated with a given property amenity ID.
func (s *svc) GetProperty(ctx context.Context) (repo.Property, error) {
	// Get the property amenity ID from context
	propertyAmenityID := hf.GetPropertyAmenityID(ctx)

	// Retrieve the property associated with the property amenity
	property, err := s.repo.GetPropertyByPropertyAmenityID(ctx, hf.ToPgUUID(&propertyAmenityID))
	if err != nil {
		// Handle not found error
		if err == pgx.ErrNoRows {
			return repo.Property{}, hf.ErrRelatedEntityNotFound
		}
		return repo.Property{}, err
	}

	// Return the retrieved property
	return property, nil
}

// GetLicence retrieves the licence associated with a given property amenity ID.
func (s *svc) GetLicence(ctx context.Context) (repo.Licence, error) {
	// Get the property amenity ID from context
	propertyAmenityID := hf.GetPropertyAmenityID(ctx)

	// Retrieve the licence associated with the property amenity
	licence, err := s.repo.GetLicenceByPropertyAmenityID(ctx, hf.ToPgUUID(&propertyAmenityID))
	if err != nil {
		// Handle not found error
		if err == pgx.ErrNoRows {
			return repo.Licence{}, hf.ErrRelatedEntityNotFound
		}
		return repo.Licence{}, err
	}

	// Return the retrieved licence
	return licence, nil
}
