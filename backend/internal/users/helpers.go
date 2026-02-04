package users

import (
	"errors"
	"ollerod-pms/internal/helpers"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleSuperAdmin   Role = "superadmin"
	RoleManager      Role = "manager"
	RoleReceptionist Role = "receptionist"
	RoleHousekeeper  Role = "housekeeper"
	RoleAccountant   Role = "accountant"
	RoleGuest        Role = "guest"
	RoleUser         Role = "user"
)

var validRoles = []Role{RoleSuperAdmin, RoleManager, RoleReceptionist, RoleHousekeeper, RoleAccountant, RoleGuest, RoleUser}
var validRoleStrings = []string{string(RoleSuperAdmin), string(RoleManager), string(RoleReceptionist), string(RoleHousekeeper), string(RoleAccountant), string(RoleGuest), string(RoleUser)}

var (
	ErrInvalidUsername   = errors.New("username must be between 3 and 50 characters and cannot contain spaces")
	ErrInvalidEmail      = errors.New("email must be a valid email address")
	ErrInvalidPassword   = errors.New("Password must be at least 8 characters long")
	ErrInvalidRole       = errors.New("Role must be one of: " + strings.Join(validRoleStrings, ", "))
	ErrUserIDRequired    = errors.New("user ID is required")
	ErrNilUpdateParams   = errors.New("update parameters cannot be nil")
	ErrLicenceIDNotFound = errors.New("licence ID not found")
	ErrInvalidFirstName  = errors.New("first name must be between 2 and 50 characters")
	ErrInvalidLastName   = errors.New("last name must be between 2 and 50 characters")
	RoleNotProvided      = errors.New("role not provided")
)

// validateUsername checks if the username length is between 3 and 50 characters and does not contain spaces.
// Example usage: validateUsername("john_doe") => true
// Example usage: validateUsername("ab") => false
// Example usage: validateUsername("thisusernameiswaytoolongtobevalidbecauseitexceedsthefiftycharacterlimit") => false
// Example usage: validateUsername("invalid username") => false
// Returns true if the username is valid, false otherwise.
func validateUsername(username string) bool {
	return helpers.StringCharCount(username) >= 3 && helpers.StringCharCount(username) <= 50 && !strings.Contains(username, " ")
}

func validatePassword(password string) bool {
	return helpers.StringCharCount(password) >= 8
}

// validateEmail checks if the provided email has a valid format.
// Example of valid email: user@example.com
// Example usage: validateEmail("user@example.com") => true
// Example usage: validateEmail("invalid-email") => false
// Returns true if the email is valid, false otherwise.
func validateEmail(email string) bool {
	return helpers.IsValidEmail(email)
}

// validateRole checks if the provided role is one of the allowed roles.
// Example usage: validateRole(RoleManager) => true
// Example usage: validateRole("invalidrole") => false
// Returns true if the role is valid, false otherwise.
func validateRole(role Role) bool {
	allowedRoles := make(map[Role]bool)

	// Populate the map with valid roles
	for _, r := range validRoles {
		allowedRoles[r] = true
	}

	return allowedRoles[role]
}

// validateFirstName checks if the first name length is between 2 and 50 characters.
// Example usage: validateFirstName("John") => true
// Example usage: validateFirstName("A") => false
// Returns true if the first name is valid, false otherwise.
func validateFirstName(firstName string) bool {
	return helpers.StringCharCount(firstName) >= 2 && helpers.StringCharCount(firstName) <= 50
}

// validateLastName checks if the last name length is between 2 and 50 characters.
// Example usage: validateLastName("Doe") => true
// Example usage: validateLastName("B") => false
// Returns true if the last name is valid, false otherwise.
func validateLastName(lastName string) bool {
	return helpers.StringCharCount(lastName) >= 2 && helpers.StringCharCount(lastName) <= 50
}

// validateCreateUserParams validates all parameters required to create a user.
// Returns an error if any parameter is invalid.
func validateCreateUserParams(params *CreateUserParams) error {
	// Check for nil params
	if params == nil {
		return errors.New("create user parameters cannot be nil")
	}

	// Username is required, so can not be empty
	if params.Username == "" || !validateUsername(params.Username) {
		return ErrInvalidUsername
	}

	// Email is required, so can not be empty
	if params.Email == "" || !validateEmail(params.Email) {
		return ErrInvalidEmail
	}

	// Password is required, so can not be empty
	if params.Password == "" || !validatePassword(params.Password) {
		return ErrInvalidPassword
	}

	// Role is required, but if not provided, default to "user"
	if params.Role == "" || !validateRole(params.Role) {
		return ErrInvalidRole
	}

	// LicenceID is required, so can not be nil
	// (uuid.Nil is the zero value for uuid.UUID)
	// Licence not found will be checked in service layer
	if params.LicenceID == uuid.Nil {
		return ErrLicenceIDNotFound
	}

	// First name is optional, but if provided, validate it
	if params.FirstName != "" && !validateFirstName(params.FirstName) {
		return ErrInvalidFirstName
	}

	// Last name is optional, but if provided, validate it
	if params.LastName != "" && !validateLastName(params.LastName) {
		return ErrInvalidLastName
	}

	// params.IsActive can only be true or false, so no need to validate its presence

	return nil
}

// validateUpdateUserParams validates all parameters required to update a user.
// Returns an error if any parameter is invalid.
// Note: All fields except UserID are optional for updates.
func validateUpdateUserParams(params *updateUserParams) error {
	// Check for nil params
	if params == nil {
		return ErrNilUpdateParams
	}

	// UserID is required for updates
	if params.UserID == uuid.Nil {
		return ErrUserIDRequired
	}

	// Username is optional, but if provided, validate it
	if username := params.Username; helpers.ParamIsProvided(username) && !validateUsername(*username) {
		return ErrInvalidUsername
	}

	// Email is optional, but if provided, validate it
	if email := params.Email; helpers.ParamIsProvided(email) && !validateEmail(*email) {
		return ErrInvalidEmail
	}

	// Password is optional, but if provided, validate it
	if password := params.Password; helpers.ParamIsProvided(password) && !validatePassword(*password) {
		return ErrInvalidPassword
	}

	// LicenceID is optional, but if provided, validate it
	// (Licence existence will be checked in service layer)
	if licenceID := params.LicenceID; licenceID != nil && *licenceID == uuid.Nil {
		return ErrLicenceIDNotFound
	}

	// Role is optional, but if provided, validate it
	if role := params.Role; helpers.ParamIsProvided(role) && !validateRole(Role(*role)) {
		return ErrInvalidRole
	}
	// First name is optional, but if provided, validate it
	if firstName := params.FirstName; helpers.ParamIsProvided(firstName) && !validateFirstName(*firstName) {
		return ErrInvalidFirstName
	}

	// Last name is optional, but if provided, validate it
	if lastName := params.LastName; helpers.ParamIsProvided(lastName) && !validateLastName(*lastName) {
		return ErrInvalidLastName
	}

	// IsActive is optional, so no need to validate its presence

	return nil
}

// hashPassword hashes the provided password using bcrypt.
// Returns the hashed password as a string or an error if hashing fails.
// Example usage: hashPassword("mysecretpassword") => "$2a$10$EixZaYVK1fsbw1ZfbX3OXePaWxn96p36Z1Z6Fh5j6K5j6K5j6K5j6"
func HashPassword(password string) (string, error) {
	// Generate the bcrypt hash of the password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// Handle potential error during hashing
	if err != nil {
		return "", err
	}

	// Convert hashed bytes to string
	hashedPassword := string(hashedBytes)

	// Return the hashed password
	return hashedPassword, nil
}
