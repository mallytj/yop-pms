package licences

import (
	"errors"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	"ollerod-pms/internal/helpers"
	"regexp"
)

var (
	ErrInvalidLicenceKey       = errors.New("Licence key must be in format XXX-YYYY where X is uppercase letter and Y is digit (e.g., ABC-1234)")
	ErrInvalidOrganisationName = errors.New("Organisation name must be between 2 and 100 characters")
	ErrInvalidContactEmail     = errors.New("invalid contact email")
	ErrInvalidLicenceNotes     = errors.New("invalid licence notes")
)

/*
validateLicenceKey checks if the licence key is in the correct format.
Expected format: XXX-YYYY where X is uppercase letter and Y is digit
Example: ABC-1234
*/
func validateLicenceKey(licenceKey string) bool {
	// Regex to match format XXX-YYYY where X is uppercase letter and Y is digit
	re := regexp.MustCompile(`^[A-Z]{3}-\d{4}$`)

	if !re.MatchString(licenceKey) {
		return false
	}

	return true
}

// validateOrganisationName checks if the organisation name length is between 2 and 100 characters.
func validateOrganisationName(name string) bool {

	// Add actual validation logic here
	return helpers.StringCharCount(name) > 2 && helpers.StringCharCount(name) < 100
}

// validateContactEmail checks if the contact email is in a valid email format.
func validateContactEmail(email string) bool {
	// Add actual validation logic here
	return helpers.IsValidEmail(email)
}

// validateLicenceNotes checks if the licence notes are valid (can be empty), but if provided, should not exceed 500 characters.
func validateLicenceNotes(notes string) bool {
	return helpers.StringCharCount(notes) <= 500
}

// validateCreateLicenceParams validates all parameters required to create a licence.
func validateCreateLicenceParams(params repo.CreateLicenceParams) error {
	if !validateLicenceKey(params.LicenceKey) {
		return ErrInvalidLicenceKey
	}

	if helpers.ParamIsProvided(&params.OrganisationName) && !validateOrganisationName(params.OrganisationName) {
		return ErrInvalidOrganisationName
	}

	if helpers.ParamIsProvided(&params.ContactEmail) && !validateContactEmail(params.ContactEmail) {
		return ErrInvalidContactEmail
	}

	if helpers.ParamIsProvided(&params.LicenceNotes.String) && !validateLicenceNotes(params.LicenceNotes.String) {
		return ErrInvalidLicenceNotes
	}
	return nil
}

// validateUpdateLicenceParams validates all parameters required to update a licence.
func validateUpdateLicenceParams(params repo.UpdateLicenceParams) error {
	if params == (repo.UpdateLicenceParams{}) {
		return nil // No fields to update
	}
	if helpers.ParamIsProvided(&params.OrganisationName) && !validateOrganisationName(params.OrganisationName) {
		return ErrInvalidOrganisationName
	}

	if helpers.ParamIsProvided(&params.ContactEmail) && !validateContactEmail(params.ContactEmail) {
		return ErrInvalidContactEmail
	}

	if helpers.ParamIsProvided(&params.LicenceNotes.String) && !validateLicenceNotes(params.LicenceNotes.String) {
		return ErrInvalidLicenceNotes
	}
	return nil
}
