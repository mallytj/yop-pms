package users

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
	ListUsers(ctx context.Context) ([]repo.User, error)
	CreateUser(ctx context.Context, params createUserParams) (repo.User, error)
	GetUserById(ctx context.Context, userID uuid.UUID) (repo.User, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, params updateUserParams) (repo.User, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	GetLicence(ctx context.Context, userID uuid.UUID) (repo.Licence, error)
}

func NewService(r repo.Queries, db *pgx.Conn) Service {
	return &svc{
		repo: r,
		db:   db,
	}
}

func (s *svc) ListUsers(ctx context.Context) ([]repo.User, error) {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}

	return users, nil
}

func (s *svc) GetUserById(ctx context.Context, userID uuid.UUID) (repo.User, error) {
	user, err := s.repo.GetUserByID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.User{}, ErrUserNotFound
		}
		return repo.User{}, fmt.Errorf("error getting user by ID: %w", err)
	}

	return user, nil
}

func (s *svc) CreateUser(ctx context.Context, params createUserParams) (repo.User, error) {
	if err := validateCreateUserParams(&params); err != nil {
		return repo.User{}, fmt.Errorf("invalid create user parameters: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.User{}, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)

	encryptedPassword, err := hashPassword(params.Password)
	if err != nil {
		return repo.User{}, fmt.Errorf("error hashing password: %w", err)
	}

	user, err := qtx.CreateUser(ctx, repo.CreateUserParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: encryptedPassword,
		Role:         string(params.Role),
	})

	if err != nil {
		return repo.User{}, fmt.Errorf("error creating user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.User{}, fmt.Errorf("error committing transaction: %w", err)
	}

	return user, nil
}

func (s *svc) UpdateUser(ctx context.Context, userID uuid.UUID, params updateUserParams) (repo.User, error) {
	if err := validateUpdateUserParams(&params); err != nil {
		return repo.User{}, fmt.Errorf("invalid update user parameters: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.User{}, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)

	encryptedPassword, err := hashPassword(params.Password)
	if err != nil {
		return repo.User{}, fmt.Errorf("error hashing password: %w", err)
	}

	user, err := qtx.UpdateUser(ctx, repo.UpdateUserParams{
		ID:           pgtype.UUID{Bytes: userID, Valid: true},
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: encryptedPassword,
		Role:         params.Role,
	})

	if err != nil {
		return repo.User{}, fmt.Errorf("Error updating user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.User{}, fmt.Errorf("error committing transaction: %w", err)
	}

	return user, nil
}

func (s *svc) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)

	err = qtx.DeleteUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

func (s *svc) GetLicence(ctx context.Context, userID uuid.UUID) (repo.Licence, error) {
	licence, err := s.repo.GetLicenceByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return repo.Licence{}, fmt.Errorf("error listing licence: %w", err)
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
