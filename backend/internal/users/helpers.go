package users

import (
	"errors"
	"ollerod-pms/internal/helpers"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	RoleSuperAdmin   Role = "superadmin"
	RoleManager      Role = "manager"
	RoleReceptionist Role = "receptionist"
	RoleHousekeeper  Role = "housekeeper"
	RoleAccountant   Role = "accountant"
	RoleGuest        Role = "guest"
)

var validRoles = []Role{RoleSuperAdmin, RoleManager, RoleReceptionist, RoleHousekeeper, RoleAccountant, RoleGuest}
var validRoleStrings = []string{string(RoleSuperAdmin), string(RoleManager), string(RoleReceptionist), string(RoleHousekeeper), string(RoleAccountant), string(RoleGuest)}

var (
	ErrInvalidUsername = errors.New("invalid username")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("Password must be at least 8 characters long")
	ErrInvalidRole     = errors.New("Role must be one of: " + strings.Join(validRoleStrings, ", "))
)

func validateEmail(email string) bool {
	return helpers.IsValidEmail(email)
}

func validateRole(role Role) bool {
	allowedRoles := make(map[Role]bool)

	// Populate the map with valid roles
	for _, r := range validRoles {
		allowedRoles[r] = true
	}

	return allowedRoles[role]
}

func validateCreateUserParams(params *createUserParams) error {
	if params.Username == "" || helpers.StringCharCount(params.Username) < 3 || helpers.StringCharCount(params.Username) > 20 {
		return ErrInvalidUsername
	}
	if !validateEmail(params.Email) {
		return ErrInvalidEmail
	}
	if params.Password == "" || helpers.StringCharCount(params.Password) < 8 {
		return ErrInvalidPassword
	}
	if !validateRole(params.Role) {
		return ErrInvalidRole
	}
	return nil
}


func validateUpdateUserParams(params *updateUserParams) error {
	if username := params.Username; username != "" {
		if helpers.StringCharCount(username) < 3 || helpers.StringCharCount(username) > 20 {
			return ErrInvalidUsername
		}
	}
	if email := params.Email; email != "" {
		if !validateEmail(email) {
			return ErrInvalidEmail
		}
	}
	if password := params.Password; password != "" {
		if helpers.StringCharCount(password) < 8 {
			return ErrInvalidPassword
		}
	}
	if role := params.Role; role != "" {
		if !validateRole(Role(role)) {
			return ErrInvalidRole
		}
	}
	return nil
}

func hashPassword(password string) (string, error) {
	// Placeholder for password hashing logic
	// In a real implementation, use a secure hashing algorithm like bcrypt
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hashedPassword := string(hashedBytes)

	return hashedPassword, nil
}
