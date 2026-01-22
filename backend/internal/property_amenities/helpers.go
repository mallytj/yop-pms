package property_amenities

import (
	"errors"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"
	hf "ollerod-pms/internal/helpers"
)

var (
	ErrPropertyAmenityNotFound    = errors.New("property amenity not found")
	ErrInvalidPropertyAmenityName = errors.New("invalid property amenity name")
	ErrInvalidShortcode           = errors.New("invalid shortcode")
	ErrInvalidDescription         = errors.New("invalid description")
	ErrCreatingPropertyAmenity    = errors.New("error creating property amenity in database")
	ErrListingPropertyAmenities   = errors.New("error listing property amenities from database")
	ErrUpdatingPropertyAmenity    = errors.New("error updating property amenity in database")
	ErrDeletingPropertyAmenity    = errors.New("error deleting property amenity from database")
	ErrRelatedEntityNotFound      = errors.New("related entity not found")
	ErrDuplicatedField            = errors.New("duplicated field error")
	ErrNoFieldsToUpdate           = errors.New("no fields to update")
)

// validatePropertyAmenityName checks if the property amenity name length is between 2 and 100 characters.
func validatePropertyAmenityName(name string) bool {
	// Add actual validation logic here
	return hf.StringCharCount(name) > 2 && hf.StringCharCount(name) < 100
}

// validateShortcode checks if the shortcode length is between 2 and 5 characters.
func validateShortcode(shortcode string) bool {
	// Add actual validation logic here
	return hf.StringCharCount(shortcode) >= 2 && hf.StringCharCount(shortcode) <= 5
}

// validateDescription checks if the description length is between 0 and 500 characters.
func validateDescription(description string) bool {
	// Add actual validation logic here
	return hf.StringCharCount(description) <= 500
}

// validateCreatePropertyAmenityParams validates all parameters required to create a property amenity.
func validateCreatePropertyAmenityParams(params repo.CreatePropertyAmenityParams) error {
	// Name is required and should be validated
	if !validatePropertyAmenityName(params.Name) {
		return ErrInvalidPropertyAmenityName
	}

	// Shortcode is required and should be validated
	if !validateShortcode(params.ShortCode) {
		return ErrInvalidShortcode
	}

	// Description can be empty, but if provided, should be validated
	if hf.ParamIsProvided(&params.Description.String) && !validateDescription(params.Description.String) {
		return ErrInvalidDescription
	}
	return nil
}

// validateUpdatePropertyAmenityParams validates all parameters required to update a property amenity.
func validateUpdatePropertyAmenityParams(params repo.UpdatePropertyAmenityParams) error {
	if params == (repo.UpdatePropertyAmenityParams{}) {
		return ErrNoFieldsToUpdate // No fields to update
	}

	// Name is optional, but if provided, should be validated
	if hf.ParamIsProvided(&params.Name.String) && !validatePropertyAmenityName(params.Name.String) {
		return ErrInvalidPropertyAmenityName
	}

	// Shortcode is optional, but if provided, should be validated
	if hf.ParamIsProvided(&params.ShortCode.String) && !validateShortcode(params.ShortCode.String) {
		return ErrInvalidShortcode
	}

	// Description is optional, but if provided, should be validated
	if hf.ParamIsProvided(&params.Description.String) && !validateDescription(params.Description.String) {
		return ErrInvalidDescription
	}
	return nil
}
