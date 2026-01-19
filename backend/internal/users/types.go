package users

import (
	"github.com/google/uuid"
)

type Role string

type createUserParams struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      Role `json:"role"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsActive  bool   `json:"is_active"`
}

type GetUsersParams struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string      `json:"email"`
	Role   Role      `json:"role"`
}

type updateUserParams struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string      `json:"username"`
	Email     string      `json:"email"`
	Password  string      `json:"password"`
	Role      string      `json:"role"`
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
	IsActive  bool        `json:"is_active"`
}
