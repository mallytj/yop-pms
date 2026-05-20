# Validation Guide

Constraints-driven struct validation using DB-derived rules. No manual tags needed — field mapping uses `json` struct tags.

## 1. Overview

`internal/platform/validation/validate.go` validates Go structs against constraints from `config/constraints.g.yml`.

**Key features:**
- Schema-first: constraints sync from PostgreSQL check constraints via `make gen-constraints`
- JSONB support: validates keys inside JSONB columns
- Nested slices: recurse into struct slices with `constraints:"schema.table"` tag
- UUID zero-check: detects `uuid.Nil` as zero value

## 2. Usage

```go
import "github.com/lexxcode1/yop-pms/internal/platform/validation"

type CreateReservationInput struct {
    Source     string    `json:"source"`
    PropertyID uuid.UUID `json:"property_id"`
    Notes      string    `json:"notes"`
}

func (h *Handler) CreateReservation(w http.ResponseWriter, r *http.Request) {
    var input CreateReservationInput
    if err := json.ReadJSON(r, &input); err != nil {
        json.WriteError(w, r, err)
        return
    }

    // Validate against operations.reservations constraints
    if errs := validation.Struct(input, "operations.reservations"); len(errs) > 0 {
        json.WriteError(w, r, apierror.ErrUnprocessableEntity.WithDetails(errs))
        return
    }

    // Proceed with business logic...
}
```

## 3. Nested Slices

For struct slices, add `constraints:"schema.table"` tag to recurse:

```go
type CreateReservationInput struct {
    Source     string            `json:"source"`
    PropertyID uuid.UUID         `json:"property_id"`
    Items      []CreateItemInput `json:"items" constraints:"operations.reservation_items"`
}

type CreateItemInput struct {
    BookedRoomTypeID uuid.UUID `json:"booked_room_type_id"`
    AdultsCount      int       `json:"adults_count"`
    StayPeriod       string    `json:"stay_period"`
}
```

Errors include array index: `items[0]: adults_count must be at least 1`.

## 4. JSONB Sub-fields

JSONB columns with sub-field constraints validate automatically when struct fields match JSONB key names:

```go
// pricing.booked_daily_rates has adjustment.jsonb with type/value/reason required
type UpdateRateInput struct {
    Type   string `json:"type"`   // matches adjustment.jsonb.type
    Value  int    `json:"value"`  // matches adjustment.jsonb.value
    Reason string `json:"reason"` // matches adjustment.jsonb.reason
}
```

## 5. Supported Rules

| Rule | Description | Example |
|------|-------------|---------|
| `required` | Non-zero value required | `source is required` |
| `max_length` | String length ≤ N | `notes must be at most 2500 characters` |
| `min_length` | String length ≥ N | `name must be at least 2 characters` |
| `exact_length` | String length = N | `currency_code must be exactly 3 characters` |
| `min` / `max` | Numeric range | `adults_count must be at least 1` |
| `pattern` | Regex match | `username does not match required pattern` |

## 6. Pointer Fields

Nil pointers fail `required` check. Non-nil pointers dereference for validation:

```go
type Input struct {
    PropertyID *uuid.UUID `json:"property_id"`
}

// Nil pointer → error: "property_id is required"
// Non-nil pointer → validates UUID value
```

## 7. Error Format

Each error follows pattern: `{field_name} {message}`

Examples:
- `source is required`
- `notes must be at most 2500 characters`
- `adults_count must be at least 1`
- `username does not match required pattern`
- `items[0]: adults_count must be at least 1` (nested slice)

## 8. Regeneration

After schema changes:

```bash
make gen-constraints
git add config/constraints.g.yml web/src/lib/types/constraints.g.ts
```

## 9. Limitations (TODO)

See `internal/platform/validation/validate.go` package-level comments:

- **ValidRange**: infer arrival < departure from stay_period fields
- **RequiredWith**: enforce co-required fields
- **Comparisons**: cross-column checks (e.g. max_los > min_los)
- **time.Time**: validate non-zero time, min/max date boundaries
- **Custom zero-check registry**: allow domain-specific IsZero funcs
- **Nested structs (non-slice)**: support single struct fields with constraints tag
- **Error aggregation**: collect all errors per field instead of first-only

## 10. Testing

See `internal/platform/validation/validate_test.go` for examples:

```go
func TestStruct_RequiredString(t *testing.T) {
    input := validReservation()
    input.Source = ""
    errs := Struct(input, "operations.reservations")
    require.Len(t, errs, 1)
    require.Contains(t, errs[0].Error(), "source")
    require.Contains(t, errs[0].Error(), "required")
}
```
