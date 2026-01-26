package users

import (
	"context"
	"errors"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/helpers"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrLicenceNotFound     = errors.New("licence not found")
	ErrDuplicatedField     = errors.New("user with the given username or email already exists")
	ErrListingUsers        = errors.New("listing users failed")
	ErrGettingUser         = errors.New("getting user failed")
	ErrInvalidCreateParams = errors.New("invalid create user parameters")
	ErrStartingTx          = errors.New("starting transaction failed")
	ErrHashingPw           = errors.New("hashing password failed")
	ErrCreatingUser        = errors.New("creating user failed")
	ErrCommittingTx        = errors.New("committing transaction failed")
	ErrInvalidUpdateParams = errors.New("invalid update user parameters")
	ErrUpdatingUser        = errors.New("updating user failed")
	ErrDeletingUser        = errors.New("deleting user failed")
	ErrGettingLicence      = errors.New("getting licence failed")
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
	ListUsers(ctx context.Context) ([]repo.User, error)
	CreateUser(ctx context.Context, params CreateUserParams) (repo.User, error)
	GetUserById(ctx context.Context, userID uuid.UUID) (repo.User, error)
	UpdateUser(ctx context.Context, params updateUserParams) (repo.User, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	GetLicence(ctx context.Context, userID uuid.UUID) (repo.Licence, error)
}

func NewService(r repo.Queries, db *pgxpool.Pool) Service {
	return &svc{
		repo: r,
		db:   db,
	}
}

// ListUsers retrieves all users from the database. (CRUD - Read)
// Returns a slice of User objects or an error.
func (s *svc) ListUsers(ctx context.Context) ([]repo.User, error) {
	// Call the repository method to list users
	users, err := s.repo.ListUsers(ctx)

	// Handle any errors that occurred during the repository call
	if err != nil {
		// If no rows are found, return an empty slice
		// Shouldn't be an error anyway, but just in case
		if errors.Is(err, pgx.ErrNoRows) {
			return []repo.User{}, nil
		}

		// For other errors, wrap and return the error with context
		return nil, err
	}

	// If no users are found, return an empty slice
	if len(users) == 0 {
		return []repo.User{}, nil
	}

	// Return the list of users
	return users, nil
}

// GetUserById retrieves a user by their ID from the database. (CRUD - Read)
// Returns a User object or an error if the user is not found.
func (s *svc) GetUserById(ctx context.Context, userID uuid.UUID) (repo.User, error) {
	// Call the repository method to get the user by ID
	user, err := s.repo.GetUserByID(ctx, helpers.ToPgUUID(&userID))

	if err != nil {
		// If no rows are found, return a user not found error
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.User{}, ErrUserNotFound
		}

		// For other errors, wrap and return the error with context
		return repo.User{}, err
	}

	// Return the retrieved user
	return user, nil
}

// CreateUser creates a new user in the database. (CRUD - Create)
// Returns the created User object with its assigned ID or an error.
func (s *svc) CreateUser(ctx context.Context, params CreateUserParams) (repo.User, error) {
	// Validate parameters, return error if invalid
	if err := validateCreateUserParams(&params); err != nil {
		return repo.User{}, err
	}

	// Start transaction for creating user
	tx, err := s.db.Begin(ctx)

	if err != nil {
		return repo.User{}, err
	}
	// Ensure the transaction is rolled back if not committed
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Hash the password before storing
	encryptedPassword, err := hashPassword(params.Password)

	// Handle password hashing error
	if err != nil {
		return repo.User{}, err
	}

	// Create the user in the database within the transaction
	user, err := qtx.CreateUser(ctx, repo.CreateUserParams{
		Username:     params.Username,
		FirstName:    params.FirstName,
		LastName:     params.LastName,
		LicenceID:    helpers.ToPgUUID(&params.LicenceID),
		IsActive:     helpers.ToPgBool(&params.IsActive),
		Email:        params.Email,
		PasswordHash: encryptedPassword,
		Role:         string(params.Role),
	})

	// Initialize pgErr for error type assertion
	var pgErr *pgconn.PgError

	if err != nil {
		// Check for specific PostgreSQL errors
		if errors.As(err, &pgErr) {
			if pgErr.Code == uniqueViolationCode {
				return repo.User{}, ErrDuplicatedField
			}
			if pgErr.Code == foreignKeyViolationCode {
				return repo.User{}, ErrLicenceNotFound
			}
		}

		// For other errors, wrap and return the error with context
		return repo.User{}, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return repo.User{}, err
	}

	// Return the created user
	return user, nil
}

// UpdateUser updates an existing user in the database. (CRUD - Update)
// Returns the updated User object or an error if the user is not found.
func (s *svc) UpdateUser(ctx context.Context, params updateUserParams) (repo.User, error) {
	// Extract user ID from context
	userID := helpers.GetUserID(ctx)

	if params == (updateUserParams{}) {
		return repo.User{}, nil // No fields to update
	}

	// Validate parameters
	if err := validateUpdateUserParams(&params); err != nil {
		return repo.User{}, err
	}

	// Start transaction for updating user
	tx, err := s.db.Begin(ctx)

	// Ensure the transaction is rolled back if not committed
	if err != nil {
		return repo.User{}, err
	}
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Handle optional password update
	var encryptedPassword string
	if params.Password != nil {
		encryptedPassword, err = hashPassword(*params.Password)
		if err != nil {
			return repo.User{}, err
		}
	}

	// Prepare arguments for updating the user
	args := repo.UpdateUserParams{
		ID:           pgtype.UUID{Bytes: userID, Valid: true},
		FirstName:    helpers.ToPgText(params.FirstName),
		LastName:     helpers.ToPgText(params.LastName),
		Username:     helpers.ToPgText(params.Username),
		Email:        helpers.ToPgText(params.Email),
		Role:         helpers.ToPgText(params.Role),
		LicenceID:    helpers.ToPgUUID(params.LicenceID),
		IsActive:     helpers.ToPgBool(params.IsActive),
		PasswordHash: helpers.ToPgText(&encryptedPassword),
	}

	// Perform the update operation
	user, err := qtx.UpdateUser(ctx, args)

	if err != nil {
		// If no rows are found, return a user not found error
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.User{}, ErrUserNotFound
		}

		// Check for specific PostgreSQL errors
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == uniqueViolationCode {
				return repo.User{}, ErrDuplicatedField
			}

			if pgErr.Code == foreignKeyViolationCode {
				return repo.User{}, ErrLicenceNotFound
			}
		}
		// For other errors, wrap and return the error with context
		return repo.User{}, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return repo.User{}, err
	}

	// Return the updated user
	return user, nil
}

// DeleteUser deletes a user by their ID from the database. (CRUD - Delete)
// Returns an error if the user is not found.
func (s *svc) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	// Start transaction for deleting user
	tx, err := s.db.Begin(ctx)

	// Ensure the transaction is rolled back if not committed
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Use transaction for repo operations
	qtx := s.repo.WithTx(tx)

	// Perform the delete operation
	res, err := qtx.DeleteUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})

	if err != nil {
		return err
	}

	// Check if any rows were affected (i.e., if the user existed)
	if res.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Return nil if deletion was successful
	return nil
}

// GetLicence retrieves a user's licence by their user ID. (CRUD - Read)
// Returns a Licence object or an error if the licence is not found.
func (s *svc) GetLicence(ctx context.Context, userID uuid.UUID) (repo.Licence, error) {
	// Call the repository method to get the licence by user ID
	licence, err := s.repo.GetLicenceByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})

	// Handle any errors that occurred during the repository call
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.Licence{}, ErrLicenceNotFound
		}
		return repo.Licence{}, err
	}

	return licence, nil
}

// TODO implement this method if needed
// func (s *svc) GetUsers(ctx context.Context, params GetUsersParams) ([]repo.User, error) {

// 	users, err := s.repo.GetUsers(ctx, repo.GetUsersParams{
// 		Column1: params.UserID,
// 		Column3: params.Email,
// 		Column6: params.Role,
// 	})

// 	if err != nil {
// 		return nil, fmt.Errorf("error getting users: %w", err)
// 	}

// 	if len(users) == 0 {
// 		return nil, ErrUserNotFound
// 	}

// 	return users, nil
// }
