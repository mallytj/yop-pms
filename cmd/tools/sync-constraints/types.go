package main

import "regexp"

// JsonbSubField holds validation rules for a single key inside a JSONB column.
type JsonbSubField struct {
	Required bool    `yaml:"required,omitempty"`
	Pattern  *string `yaml:"pattern,omitempty"`
	Min      *int    `yaml:"min,omitempty"`
	Max      *int    `yaml:"max,omitempty"`
}

// FieldConstraint holds the parsed validation rules for a single DB column.
type FieldConstraint struct {
	Required     bool                      `yaml:"required,omitempty"`
	RequiredWith string                    `yaml:"required_with,omitempty"` // co-required: both NULL or both NOT NULL
	ValidRange   bool                      `yaml:"valid_range,omitempty"`   // range type: lower < upper enforced
	MaxLength    *int                      `yaml:"max_length,omitempty"`
	MinLength    *int                      `yaml:"min_length,omitempty"`
	ExactLength  *int                      `yaml:"exact_length,omitempty"`
	Min          *int                      `yaml:"min,omitempty"`
	Max          *int                      `yaml:"max,omitempty"`
	Pattern      *string                   `yaml:"pattern,omitempty"`
	Jsonb        map[string]*JsonbSubField `yaml:"jsonb,omitempty"`
}

// Comparison captures a cross-column ordering constraint (e.g. max_los > min_los).
type Comparison struct {
	Field    string
	Operator string
	Other    string
}

// TableEntry groups field-level constraints, cross-column comparisons, and
// table-level notes for complex constraints that cannot map to a single field.
type TableEntry struct {
	Fields      map[string]*FieldConstraint
	Comparisons []Comparison
	Notes       []string
}

// tableKey is "schema.table".
type tableKey = string

// constraintMap is the full in-memory representation before serialisation.
type constraintMap = map[tableKey]*TableEntry

// internalFields are skipped — contain bcrypt/internal invariants not user-facing.
var internalFields = map[string]bool{
	"password_hash": true,
}

// systemColumns are always DB-managed — skip from required tracking.
var systemColumns = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"deleted_at": true,
}

// functionNames are SQL function names that look like column names in some positions.
var functionNames = map[string]bool{
	"char_length":  true,
	"length":       true,
	"lower":        true,
	"upper":        true,
	"jsonb_typeof": true,
}

// --- regex patterns ---
// Postgres normalises check_clause with type casts, e.g.:
//
//	char_length((organisation_name)::text) <= 50
//	(username)::citext ~ '^[a-zA-Z0-9_]+$'::citext
//
// Patterns handle both plain `col` and `(col)::type` forms.
var (
	// single-field length / numeric / regex
	reCharLenLTE     = regexp.MustCompile(`(?i)char_length\(\(?(\w+)\)?(?:::\w+)?\)\s*<=\s*(\d+)`)
	reCharLenBetween = regexp.MustCompile(`(?i)char_length\(\(?(\w+)\)?(?:::\w+)?\)\s+between\s+(\d+)\s+and\s+(\d+)`)
	reCharLenEQ      = regexp.MustCompile(`(?i)char_length\(\(?(\w+)\)?(?:::\w+)?\)\s*=\s*(\d+)`)
	reRegex          = regexp.MustCompile(`\(?(\w+)\)?(?:::\w+)?\s*~\s*'([^']+)'`)
	reBetween        = regexp.MustCompile(`(?i)\(?(\w+)\)?(?:::\w+)?\s+between\s+(\d+)\s+and\s+(\d+)`)
	reGTE            = regexp.MustCompile(`(?i)\(?(\w+)\)?(?:::\w+)?\s*>=\s*(\d+)`)
	reGT             = regexp.MustCompile(`(?i)\(?(\w+)\)?(?:::\w+)?\s*>\s*(\d+)`)

	// range type validity: lower(col) < upper(col)  or  upper(col) > lower(col)
	reRangeValid = regexp.MustCompile(`(?i)(?:lower\(\(?(\w+)\)?\)\s*<\s*upper\(\(?(\w+)\)?\)|upper\(\(?(\w+)\)?\)\s*>\s*lower\(\(?(\w+)\)?\))`)

	// cross-column: col1 op col2  (whole clause, no numeric RHS)
	reCrossCol = regexp.MustCompile(`(?i)^\s*\(?(\w+)\)?(?:::\w+)?\s*(>=|<=|!=|>|<)\s*\(?(\w+)\)?(?:::\w+)?\s*$`)

	// co-required (required_with): col IS [NOT] NULL
	// Go RE2 has no lookahead, so we detect IS NOT NULL first then IS NULL separately.
	reIsNotNull = regexp.MustCompile(`(?i)\(?(\w+)\)?\s+IS\s+NOT\s+NULL`)
	reIsNull    = regexp.MustCompile(`(?i)\(?(\w+)\)?\s+IS\s+NULL`)

	// JSONB key presence: col ? 'key'
	reJsonbHasKey = regexp.MustCompile(`(\w+)\s*\?\s*'(\w+)'(?:::\w+)?`)
	// JSONB get text IN list: col ->> 'key' IN ('a','b')  or  = ANY(ARRAY[...])
	// Postgres normalises to (col ->> 'key'::text) = ANY (ARRAY['a'::text, ...])
	// so we need \)? after the key cast to skip the expression-grouping paren.
	reJsonbInList = regexp.MustCompile(`(?i)\(?(\w+)\)?\s*->>\s*'(\w+)'(?:::\w+)?\)?\s+IN\s*\(([^)]+)\)`)
	reJsonbAnyArr = regexp.MustCompile(`(?i)\(?(\w+)\)?\s*->>\s*'(\w+)'(?:::\w+)?\)?\s*=\s*ANY\s*\(ARRAY\[([^\]]+)\]\)`)
	// extract quoted values from IN / ARRAY lists
	reQuotedVal = regexp.MustCompile(`'((?:''|[^'])+)'`)
)
