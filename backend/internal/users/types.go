package users

import (
	"github.com/google/uuid"
)

type Role string

// CreateUserParams holds parameters for creating a new user.
// Can't just use repo.CreateUserParams because the password needs to be plain text here.
type createUserParams struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	LicenceID uuid.UUID `json:"licence_id"`
	Role      Role      `json:"role"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	IsActive  bool      `json:"is_active"`
}

// GetUsersParams holds parameters for retrieving users based on filters.
type GetUsersParams struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   Role      `json:"role"`
}

// UpdateUserParams holds parameters for updating an existing user.
// All fields except UserID are optional for updates.
// Can't just use repo.UpdateUserParams because of the password needing to be plain text here.
type updateUserParams struct {
	UserID    uuid.UUID  `json:"user_id"`    // Required
	Username  *string    `json:"username"`   // Optional
	Email     *string    `json:"email"`      // Optional
	Password  *string    `json:"password"`   // Optional
	LicenceID *uuid.UUID `json:"licence_id"` // Optional
	Role      *string    `json:"role"`       // Optional
	FirstName *string    `json:"first_name"` // Optional
	LastName  *string    `json:"last_name"`  // Optional
	IsActive  *bool      `json:"is_active"`  // Optional
}
