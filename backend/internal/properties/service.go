package properties

import (
	"context"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	hf "ollerod-pms/internal/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type svc struct {
	repo repo.Queries
	db   *pgxpool.Pool
}

type Service interface {
	CreateProperty(ctx context.Context, params repo.CreatePropertyParams) (repo.Property, error)
	UpdateProperty(ctx context.Context, params repo.UpdatePropertyParams) (repo.Property, error)
	DeleteProperty(ctx context.Context) error
	ListProperties(ctx context.Context) ([]repo.Property, error)
	GetPropertyById(ctx context.Context) (repo.Property, error)
	GetDailyAvailability(ctx context.Context) ([][]int32, error)
	GetRooms(ctx context.Context) ([]repo.Room, error)
	GetRatePlans(ctx context.Context) ([]repo.RatePlan, error)
	GetGuests(ctx context.Context) ([]repo.Guest, error)
	GetReservations(ctx context.Context) ([]repo.Reservation, error)
	GetAmenities(ctx context.Context) ([]repo.PropertyAmenity, error)
	GetRoomTypes(ctx context.Context) ([]repo.RoomType, error)
	GetLicence(ctx context.Context) (repo.Licence, error)
	GetUsersByID(ctx context.Context) ([]repo.User, error)
}

func NewService(r repo.Queries, db *pgxpool.Pool) Service {
	return &svc{
		repo: r,
		db:   db,
	}
}

// CreateProperty creates a new property in the database. (CRUD - Create)
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
		if hf.CheckErrorCode(err, hf.UniqueViolationCode) {
			return repo.Property{}, hf.ErrDuplicatedField
		}

		if hf.CheckErrorCode(err, hf.ForeignKeyViolationCode) {
			return repo.Property{}, ErrLicenceNotFound
		}

		return repo.Property{}, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return repo.Property{}, err
	}

	// Return the created property
	return property, nil
}

// ListProperties retrieves all properties from the database. (CRUD - Read)
func (s *svc) ListProperties(ctx context.Context) ([]repo.Property, error) {
	// Retrieve the list of properties from the repository
	properties, err := s.repo.ListProperties(ctx)
	if err != nil {
		// If no rows are found, return an empty slice
		if err == pgx.ErrNoRows {
			return []repo.Property{}, ErrNoPropertiesFound
		}
		return nil, err
	}

	// If no properties are found, return an empty slice
	if len(properties) == 0 {
		return []repo.Property{}, nil
	}

	// Return the list of properties
	return properties, nil
}

// UpdateProperty updates an existing property in the database. (CRUD - Update)
func (s *svc) UpdateProperty(ctx context.Context, params repo.UpdatePropertyParams) (repo.Property, error) {
	// Get propertyID from context
	// If failed, it will panic - middleware should ensure it's there
	// so this is safe
	propertyID := hf.GetPropertyID(ctx)

	// Validate parameters
	if err := validateUpdatePropertyParams(params); err != nil {
		return repo.Property{}, err
	}

	// If no fields to update, return an error
	if params == (repo.UpdatePropertyParams{}) {
		return repo.Property{}, ErrNoFieldsToUpdate
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

		// Check for unique constraint violation
		if hf.CheckErrorCode(err, hf.UniqueViolationCode) {
			return repo.Property{}, hf.ErrDuplicatedField
		}

		// Check for foreign key constraint violation
		if hf.CheckErrorCode(err, hf.ForeignKeyViolationCode) {
			return repo.Property{}, ErrLicenceNotFound
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

// DeleteProperty deletes an existing property from the database. (CRUD - Delete)
func (s *svc) DeleteProperty(ctx context.Context) error {
	// Get propertyID from context
	propertyID := hf.GetPropertyID(ctx)

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

// GetPropertyById retrieves a property by its ID from the database. (CRUD - Read)
func (s *svc) GetPropertyById(ctx context.Context) (repo.Property, error) {
	// Get propertyID from context
	propertyID := hf.GetPropertyID(ctx)

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

func (s *svc) GetDailyAvailability(ctx context.Context) ([][]int32, error) {
	// Implementation will come here
	return nil, nil
}

func (s *svc) GetRooms(ctx context.Context) ([]repo.Room, error) {
	return nil, nil
}

func (s *svc) GetRatePlans(ctx context.Context) ([]repo.RatePlan, error) {
	return nil, nil
}

func (s *svc) GetGuests(ctx context.Context) ([]repo.Guest, error) {
	// Implementation here
	return nil, nil
}

func (s *svc) GetReservations(ctx context.Context) ([]repo.Reservation, error) {
	// Implementation here
	return nil, nil
}

func (s *svc) GetAmenities(ctx context.Context) ([]repo.PropertyAmenity, error) {
	return nil, nil
}

func (s *svc) GetRoomTypes(ctx context.Context) ([]repo.RoomType, error) {
	return nil, nil
}

// GetLicence retrieves the licence associated with the property. (CRUD - Read)
// Returns a Licence object or an error if the licence is not found.
func (s *svc) GetLicence(ctx context.Context) (repo.Licence, error) {
	// Get propertyID from context
	propertyID := hf.GetPropertyID(ctx)

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

// GetUsers retrieves all users associated with the properties given licence_id (CRUD - Read)
func (s *svc) GetUsersByID(ctx context.Context) ([]repo.User, error) {
	// Get propertyID from context
	propertyID := hf.GetPropertyID(ctx)

	// Get the users associated with the property's licence
	users, err := s.repo.GetUsersByPropertyID(ctx, hf.ToPgUUID(&propertyID))
	if err != nil {
		// Handle not found error
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		// For other errors, return the error
		return nil, err
	}

	// Return the list of users
	return users, nil
}
