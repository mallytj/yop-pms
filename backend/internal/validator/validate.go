package validator

import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func setupValidator() *validator.Validate {
	v := validator.New()

	// Register custom validation for UUIDv7
	v.RegisterValidation("uuid7", func(fl validator.FieldLevel) bool {
		return validateUUIDv7(fl)
	})

	v.RegisterValidation("datestring", func(fl validator.FieldLevel) bool {
		return validateDatestring(fl)
	})

	return v
}

func validateUUIDv7(fl validator.FieldLevel) bool {
	field := fl.Field()

	switch field.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		if field.IsNil() {
			return true
		}
	}

	var id uuid.UUID
	var err error

	switch v := field.Interface().(type) {
	case uuid.UUID:
		id = v
	case string:
		id, err = uuid.Parse(v)
		if err != nil {
			return false
		}
	case *string:
		if v == nil {
			return true
		}
		id, err = uuid.Parse(*v)
	default:
		return false
	}

	return id != uuid.Nil
}

func validateDatestring(fl validator.FieldLevel) bool {
	field := fl.Field()

	switch field.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		if field.IsNil() {
			return true
		}
	}

	dateStr, ok := field.Interface().(string)
	if !ok {
		return false
	}

	fmt.Println("Validating date string:", dateStr)

	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

var validate = setupValidator()

// ValidateStruct validates a struct based on the `validate` tags and returns a formatted error message if validation fails.
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}
