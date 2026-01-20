package types

import "github.com/google/uuid"

// Role represents the role of a user in the system.
// Setting type like this to allow for easier changes in the future if needed.
type Role string

// CreateUserParams holds parameters for creating a new user.
// Can't just use repo.CreateUserParams because the password needs to be plain text here.
type CreateUserParams struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	LicenceID uuid.UUID `json:"licence_id"`
	Role      Role      `json:"role"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	IsActive  bool      `json:"is_active"`
}

// contextKey is a custom type for context keys to avoid collisions.
// It helps prevent conflicts when storing and retrieving values from context.
type ContextKey string
