# Backend Constraints

Single source of truth for validation rules synced from PostgreSQL. **Never edit `config/constraints.g.yml` manually.**

## 1. Setup

The implementation is located in `internal/platform/constraints/constraints.go`. It embeds the YAML at compile time and caches compiled regex patterns for maximum performance.

```go
package constraints

import (
	"embed"
	"regexp"
	"sync"
	"gopkg.in/yaml.v3"
)

//go:embed ../../../config/constraints.g.yml
var raw []byte

type Field struct {
	Required     bool                      `yaml:"required"`
	MaxLength    *int                      `yaml:"max_length"`
	MinLength    *int                      `yaml:"min_length"`
	ExactLength  *int                      `yaml:"exact_length"`
	Min          *int                      `yaml:"min"`
	Max          *int                      `yaml:"max"`
	Pattern      *string                   `yaml:"pattern"`
	RequiredWith string                    `yaml:"required_with"`
	ValidRange   bool                      `yaml:"valid_range"`
	Jsonb        map[string]*JsonbSubField `yaml:"jsonb"`
}

type TableEntry struct {
	Fields      map[string]*Field `yaml:"fields"`
	Comparisons []Comparison      `yaml:"comparisons"`
	Notes       []string          `yaml:"notes"`
}

// ... see internal/platform/constraints/constraints.go for full implementation
```

## 2. Production Usage (High Performance)

Use the `MatchPattern` helper to validate regex without recompiling it on every request.

```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // 1. Get rules for the table
    rules := constraints.Table("auth.users").Fields

    // 2. Validate a field
    username := rules["username"]
    
    if username.Required && req.Username == "" {
        json.WriteError(w, r, apierror.ErrBadRequest.WithMessage("username is required"))
        return
    }

    if username.Pattern != nil && !constraints.MatchPattern(*username.Pattern, req.Username) {
        json.WriteError(w, r, apierror.ErrBadRequest.WithMessage("invalid username format"))
        return
    }

    // 3. Proceed to store...
}
```

## 3. Supported Rules

| Rule | Description |
| :--- | :--- |
| `required` | Column is `NOT NULL` |
| `max_length` | Maximum string length |
| `exact_length` | String must be exactly N characters (e.g. ISO codes) |
| `min` / `max` | Numeric range boundaries |
| `pattern` | Regex pattern (cached via `constraints.MatchPattern`) |
| `required_with` | Both fields must be set or both absent |
| `valid_range` | Check if a range type (TSTZRANGE) is valid (lower < upper) |

## 4. Performance Notes

- **Zero I/O:** Rules are embedded in the binary.
- **O(1) Access:** Rules are parsed once into memory-resident maps.
- **Regex Caching:** `MatchPattern` uses a `sync.Map` to ensure patterns are compiled only once, making validation extremely fast even under high load.

## 5. Regeneration

Run after any database schema change:

```bash
make gen-constraints
git add config/constraints.g.yml web/src/lib/types/constraints.g.ts
```
