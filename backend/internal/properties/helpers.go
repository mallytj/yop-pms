package properties

import (
	"errors"
	"fmt"
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
)

func validateName(name string) bool {
	// Add actual validation logic here
	return len(name) > 2 && len(name) < 100
}

func validateAddress(address string) bool {
	// Add actual validation logic here
	return len(address) > 5 && len(address) < 200
}

func validateTimezone(timezone string) bool {
	// Add actual validation logic here
	return len(timezone) > 0 && len(timezone) < 50
}

func validatePropertyNotes(notes string) bool {
	// Add actual validation logic here
	return len(notes) <= 500
}

// validateCreatePropertyParams validates all parameters required to create a property.
// Currently a placeholder for actual validation logic.
func validateCreatePropertyParams(params repo.CreatePropertyParams) error {
	// Add validation logic for property creation parameters here
	if !validateName(params.Name) {
		return fmt.Errorf("invalid property name")
	}
	if !validateAddress(params.Address) {
		return fmt.Errorf("invalid property address")
	}
	if !validateTimezone(params.Timezone) {
		return fmt.Errorf("invalid property timezone")
	}
	if !validatePropertyNotes(params.PropertyNotes.String) {
		return fmt.Errorf("invalid property notes")
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
