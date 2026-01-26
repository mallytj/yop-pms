package properties

import (
	"errors"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	hf "ollerod-pms/internal/helpers"
)

var (
	ErrInvalidName          = errors.New("invalid property name")
	ErrInvalidAddress       = errors.New("invalid property address")
	ErrInvalidTimezone      = errors.New("invalid property timezone")
	ErrInvalidPropertyNotes = errors.New("invalid property notes")
	ErrPropertyNotFound     = errors.New("property not found")
	ErrLicenceNotFound      = errors.New("licence not found")
	ErrNoPropertiesFound    = errors.New("no properties found for the given licence ID")
	ErrNoFieldsToUpdate     = errors.New("no fields to update provided")
)

// validateName checks if the property name length is between 2 and 100 characters.
// Example usage: validateName("My Property") => true
// Example usage: validateName("A") => false
// Example usage: validateName("This property name is way too long to be considered valid because it exceeds the one hundred character limit imposed by the validation rules") => false
// Returns true if the name is valid, false otherwise.
// Further validation logic can be added as needed.
func validateName(name string) bool {
	// Add actual validation logic here
	return len(name) > 2 && len(name) < 100
}

// validateAddress checks if the property address length is between 5 and 200 characters.
// Example usage: validateAddress("123 Main St, Cityville") => true
// Example usage: validateAddress("123") => false
// Example usage: validateAddress("This address is way too long to be considered valid because it exceeds the two hundred character limit imposed by the validation rules. It just keeps going and going without any sign of stopping, making it an invalid address for our property management system.") => false
// Returns true if the address is valid, false otherwise.
// Further validation logic can be added as needed.
func validateAddress(address string) bool {
	// Add actual validation logic here
	return len(address) > 5 && len(address) < 200
}

// validateTimezone checks if the property timezone is a valid timezone string.
// Example usage: validateTimezone("Europe/Copenhagen") => true
// Example usage: validateTimezone("") => false
// Example usage: validateTimezone("ThisIsAnInvalidTimezoneStringThatExceedsTheFiftyCharacterLimitImposedByTheValidationRules") => false
// Returns true if the timezone is valid, false otherwise.
// Further validation logic can be added as needed.
func validateTimezone(timezone string) bool {
	// Add actual validation logic here
	return len(timezone) > 0 && len(timezone) < 50
}

// validatePropertyNotes checks if the property notes are valid (can be empty), but if provided, should not exceed 500 characters.
// Example usage: validatePropertyNotes("These are some notes about the property.") => true
// Example usage: validatePropertyNotes("") => true
// Example usage: validatePropertyNotes(Lorem * 500) => false
// Returns true if the notes are valid, false otherwise.
func validatePropertyNotes(notes string) bool {
	// Add actual validation logic here
	return len(notes) <= 500
}

// validateCreatePropertyParams validates all parameters required to create a property.
// Currently a placeholder for actual validation logic.
func validateCreatePropertyParams(params repo.CreatePropertyParams) error {
	// Add validation logic for property creation parameters here
	if !validateName(params.Name) {
		return ErrInvalidName
	}
	if !validateAddress(params.Address) {
		return ErrInvalidAddress
	}
	if !validateTimezone(params.Timezone) {
		return ErrInvalidTimezone
	}
	if !validatePropertyNotes(params.PropertyNotes.String) {
		return ErrInvalidPropertyNotes
	}

	return nil
}

type sanitisedUpdatePropertyParams struct {
	Name          string
	Address       string
	Timezone      string
	PropertyNotes string
}

// validateUpdatePropertyParams validates all parameters required to update a property.
// Currently a placeholder for actual validation logic.
func validateUpdatePropertyParams(params repo.UpdatePropertyParams) error {
	sanitisedParams := sanitisedUpdatePropertyParams{
		Name:          params.Name.String,
		Address:       params.Address.String,
		Timezone:      params.Timezone.String,
		PropertyNotes: params.PropertyNotes.String,
	}
	// Add validation logic for property update parameters here
	if hf.ParamIsProvided(&sanitisedParams.Name) && !validateName(sanitisedParams.Name) {
		return ErrInvalidName
	}
	if hf.ParamIsProvided(&sanitisedParams.Address) && !validateAddress(sanitisedParams.Address) {
		return ErrInvalidAddress
	}
	if hf.ParamIsProvided(&sanitisedParams.Timezone) && !validateTimezone(sanitisedParams.Timezone) {
		return ErrInvalidTimezone
	}
	if hf.ParamIsProvided(&sanitisedParams.PropertyNotes) && !validatePropertyNotes(sanitisedParams.PropertyNotes) {
		return ErrInvalidPropertyNotes
	}

	return nil
}
