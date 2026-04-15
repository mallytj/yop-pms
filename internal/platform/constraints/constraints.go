package constraints

import (
	"embed"
	"fmt"
	"regexp"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed ../../../config/constraints.g.yml
var raw []byte

// Field defines validation rules for a specific column.
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

// JsonbSubField defines validation rules for keys inside a JSONB column.
type JsonbSubField struct {
	Required bool    `yaml:"required"`
	Pattern  *string `yaml:"pattern"`
	Min      *int    `yaml:"min"`
	Max      *int    `yaml:"max"`
}

// Comparison defines cross-column validation logic.
type Comparison struct {
	Field    string `yaml:"field"`
	Operator string `yaml:"operator"`
	Other    string `yaml:"other"`
}

// TableEntry contains all constraints for a database table.
type TableEntry struct {
	Fields      map[string]*Field `yaml:"fields"`
	Comparisons []Comparison      `yaml:"comparisons"`
	Notes       []string          `yaml:"notes"`
}

// constraintsFile represents the root structure of the YAML file.
type constraintsFile struct {
	Constraints map[string]*TableEntry `yaml:"constraints"`
}

// Global state
var (
	File       constraintsFile
	regexCache sync.Map
)

func init() {
	if err := yaml.Unmarshal(raw, &File); err != nil {
		panic(fmt.Sprintf("failed to parse constraints.g.yml: %v", err))
	}
}

// Table returns the constraint entry for a given "schema.table" key.
func Table(key string) *TableEntry {
	return File.Constraints[key]
}

// MatchPattern efficiently checks a string against a regex pattern,
// using a cache to avoid repeated compilation.
func MatchPattern(pattern, value string) bool {
	re, ok := regexCache.Load(pattern)
	if !ok {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return false
		}
		regexCache.Store(pattern, re)
	}
	return re.(*regexp.Regexp).MatchString(value)
}
