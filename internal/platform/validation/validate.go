// Package validation provides a constraints-driven struct validator.
//
// It reads DB-derived constraints from config/constraints.g.yml (via the
// internal/platform/constraints package) and validates Go structs against them
// using reflection. Field mapping uses json struct tags — no separate validation
// tags needed.
//
// Usage:
//
//	if errs := validation.Struct(input, "operations.reservations"); len(errs) > 0 {
//	    // handle errors
//	}
//
// For nested struct slices, add a constraints:"schema.table" tag on the
// slice field to recurse into each element:
//
//	Items []CreateItemInput `json:"items" constraints:"operations.reservation_items"`
//
// JSONB sub-fields (keys inside a JSONB column) are validated when any
// constraint field has jsonb rules and the struct has fields whose json tags
// match those key names.
//
// TODO(validation):
//   - ValidRange: infer arrival < departure from stay_period fields with valid_range=true.
//   - RequiredWith: enforce co-required fields (both nil or both non-nil).
//   - Comparisons: cross-column checks (e.g. max_los > min_los) — may be better as domain logic.
//   - time.Time: validate required (non-zero time), min/max date boundaries.
//   - Custom zero-check registry: allow domains to register their own IsZero funcs for types beyond uuid.UUID.
//   - Nested structs (non-slice): support constraints:"table.key" on single struct fields.
//   - Error aggregation: collect all errors instead of returning on first violation (currently first-only per field).
package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/lexxcode1/yop-pms/internal/platform/constraints"
)

// Struct validates v against constraints for the given schema.table key.
// v must be a struct or a pointer to a struct. Returns nil if tableKey has
// no entry in constraints or if no violations are found.
//
// TODO: support single nested structs (not just slices) with constraints tag.
// TODO: register custom IsZero functions for domain types (e.g. UUID wrappers).
func Struct(v any, tableKey string) []error {
	entry := constraints.Table(tableKey)
	if entry == nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return []error{fmt.Errorf("validation: expected struct, got %s", rv.Kind())}
	}

	return validateStruct(rv, entry)
}

func validateStruct(rv reflect.Value, entry *constraints.TableEntry) []error {
	rt := rv.Type()
	var errs []error

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := jsonFieldName(field)
		fv := rv.Field(i)

		// Check top-level field constraints
		if fc := entry.Fields[tag]; fc != nil {
			if fieldErrs := checkField(fc, tag, fv); len(fieldErrs) > 0 {
				errs = append(errs, fieldErrs...)
			}
		}

		// Recurse into nested struct slices
		if nestedTable, ok := field.Tag.Lookup("constraints"); ok && fv.Kind() == reflect.Slice {
			for j := 0; j < fv.Len(); j++ {
				elem := fv.Index(j)
				if elem.Kind() == reflect.Pointer {
					elem = elem.Elem()
				}
				if nested := validateStruct(elem, constraints.Table(nestedTable)); len(nested) > 0 {
					errs = append(errs, fmt.Errorf("%s[%d]: %w", tag, j, joinErrs(nested)))
				}
			}
		}
	}

	// Check JSONB sub-field rules: if any constraint field has jsonb rules,
	// match struct fields to jsonb key names and validate.
	// TODO: support nested JSONB structs (e.g. adjustment → struct with type/value/reason fields).
	errs = append(errs, checkJSONB(rt, rv, entry)...)

	return errs
}

func checkField(fc *constraints.Field, name string, fv reflect.Value) []error {
	var errs []error

	kind := fv.Kind()
	if kind == reflect.Pointer {
		if fv.IsNil() {
			if fc.Required {
				errs = append(errs, fieldErr(name, "is required"))
			}
			return errs
		}
		fv = fv.Elem()
		kind = fv.Kind()
	}

	// Required: skip numeric types — Go ints are always non-nil and zero is
	// often a valid default. Use min=1 in constraints to enforce positive values.
	if fc.Required && !isNumeric(kind) && isZero(fv, kind) {
		errs = append(errs, fieldErr(name, "is required"))
	}

	if kind == reflect.String {
		if err := checkString(fc, name, fv.String()); err != nil {
			errs = append(errs, err)
		}
	}

	if isNumeric(kind) {
		n := int(fv.Int())
		if err := checkInt(fc, name, n); err != nil {
			errs = append(errs, err)
		}
	}

	// TODO: time.Time values — validate required (non-zero), min/max date boundaries.

	return errs
}

func checkString(fc *constraints.Field, name, value string) error {
	if fc.MaxLength != nil && len(value) > *fc.MaxLength {
		return fieldErr(name, "must be at most %d characters", *fc.MaxLength)
	}
	if fc.MinLength != nil && len(value) < *fc.MinLength {
		return fieldErr(name, "must be at least %d characters", *fc.MinLength)
	}
	if fc.ExactLength != nil && len(value) != *fc.ExactLength {
		return fieldErr(name, "must be exactly %d characters", *fc.ExactLength)
	}
	if fc.Pattern != nil && !constraints.MatchPattern(*fc.Pattern, value) {
		return fieldErr(name, "does not match required pattern")
	}
	return nil
}

func checkInt(fc *constraints.Field, name string, n int) error {
	if fc.Min != nil && n < *fc.Min {
		return fieldErr(name, "must be at least %d", *fc.Min)
	}
	if fc.Max != nil && n > *fc.Max {
		return fieldErr(name, "must be at most %d", *fc.Max)
	}
	return nil
}

func checkJSONB(rt reflect.Type, rv reflect.Value, entry *constraints.TableEntry) []error {
	var errs []error
	for _, fc := range entry.Fields {
		if fc.Jsonb == nil {
			continue
		}
		for jsonbKey, jsf := range fc.Jsonb {
			for i := 0; i < rt.NumField(); i++ {
				field := rt.Field(i)
				if !field.IsExported() {
					continue
				}
				if jsonFieldName(field) != jsonbKey {
					continue
				}
				fv := rv.Field(i)
				if subErrs := checkJSONBSubField(jsf, jsonbKey, fv); len(subErrs) > 0 {
					errs = append(errs, subErrs...)
				}
			}
		}
	}
	return errs
}

func checkJSONBSubField(jsf *constraints.JsonbSubField, name string, fv reflect.Value) []error {
	var errs []error

	kind := fv.Kind()
	if kind == reflect.Pointer {
		if fv.IsNil() {
			return nil // nil pointer in JSONB context — skip
		}
		fv = fv.Elem()
		kind = fv.Kind()
	}

	if jsf.Required && isZero(fv, kind) {
		errs = append(errs, fieldErr(name, "is required"))
	}

	if kind == reflect.String && jsf.Pattern != nil && !constraints.MatchPattern(*jsf.Pattern, fv.String()) {
		errs = append(errs, fieldErr(name, "does not match required pattern"))
	}

	if isSignedInt(kind) {
		n := int(fv.Int())
		if jsf.Min != nil && n < *jsf.Min {
			errs = append(errs, fieldErr(name, "must be at least %d", *jsf.Min))
		}
		if jsf.Max != nil && n > *jsf.Max {
			errs = append(errs, fieldErr(name, "must be at most %d", *jsf.Max))
		}
	}
	// TODO: unsigned int handling for JSONB sub-field min/max.

	return errs
}

// --- helpers ---

// jsonFieldName extracts the json tag name (before the first comma).
func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	if before, _, ok := strings.Cut(tag, ","); ok {
		return before
	}
	return tag
}

// isZero checks whether a value is the zero value for its kind.
// Required fields must be non-zero.
func isZero(fv reflect.Value, kind reflect.Kind) bool {
	switch kind {
	case reflect.String:
		return fv.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return fv.Float() == 0
	case reflect.Bool:
		return !fv.Bool()
	case reflect.Slice, reflect.Map:
		return fv.Len() == 0
	case reflect.Array:
		// uuid.UUID is [16]byte — check against uuid.Nil
		if fv.Type() == reflect.TypeFor[uuid.UUID]() {
			return fv.Interface().(uuid.UUID) == uuid.Nil
		}
		return false
	case reflect.Struct:
		return false // structs are non-zero by default
	default:
		return false
	}
}

func isSignedInt(kind reflect.Kind) bool {
	return kind >= reflect.Int && kind <= reflect.Int64
}

func isUnsignedInt(kind reflect.Kind) bool {
	return kind >= reflect.Uint && kind <= reflect.Uint64
}

func isNumeric(kind reflect.Kind) bool {
	return isSignedInt(kind) || isUnsignedInt(kind)
}

func fieldErr(name, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s %s", name, msg)
}

func joinErrs(errs []error) error {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return fmt.Errorf("%s", strings.Join(msgs, "; "))
}
