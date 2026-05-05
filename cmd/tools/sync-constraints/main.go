// sync-constraints queries the live Postgres database for CHECK constraints,
// parses user-facing validation rules, and writes:
//   - config/constraints.g.yml  (shared source of truth)
//   - web/src/lib/types/constraints.g.ts  (TypeScript constants for the frontend)
//
// Run via: make gen-constraints
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

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
//   char_length((organisation_name)::text) <= 50
//   (username)::citext ~ '^[a-zA-Z0-9_]+$'::citext
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

func intPtr(n int) *int       { return &n }
func strPtr(s string) *string { return &s }

// stripOuterParens removes balanced outer parentheses added by Postgres without
// touching inner parens that belong to function calls (e.g. lower(col)).
// strings.Trim would strip trailing ')' from inside function calls.
func stripOuterParens(s string) string {
	s = strings.TrimSpace(s)
	for len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')' {
		inner := s[1 : len(s)-1]
		if parenDepth(inner) >= 0 { // inner is balanced on its own
			s = strings.TrimSpace(inner)
		} else {
			break
		}
	}
	return s
}

// parenDepth returns the minimum running depth of parentheses in s.
// A non-negative result means the string is self-balanced (no unmatched ')').
func parenDepth(s string) int {
	depth, min := 0, 0
	for _, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth < min {
				min = depth
			}
		}
	}
	return min
}

// isNullCheck reports whether a clause is solely an IS [NOT] NULL assertion.
func isNullCheck(clause string) bool {
	upper := strings.ToUpper(strings.TrimSpace(clause))
	return strings.HasSuffix(upper, "IS NOT NULL") || strings.HasSuffix(upper, "IS NULL")
}

// isEmpty reports whether a FieldConstraint carries no data at all.
func isEmpty(fc *FieldConstraint) bool {
	return !fc.Required && !fc.ValidRange && fc.RequiredWith == "" &&
		fc.MaxLength == nil && fc.MinLength == nil && fc.ExactLength == nil &&
		fc.Min == nil && fc.Max == nil && fc.Pattern == nil &&
		len(fc.Jsonb) == 0
}

func getOrCreate(cm constraintMap, key string) *TableEntry {
	if cm[key] == nil {
		cm[key] = &TableEntry{Fields: make(map[string]*FieldConstraint)}
	}
	return cm[key]
}

func getOrCreateField(entry *TableEntry, field string) *FieldConstraint {
	if entry.Fields[field] == nil {
		entry.Fields[field] = &FieldConstraint{}
	}
	return entry.Fields[field]
}

// parseClause handles single-field constraints (length, numeric, regex).
// Returns ("", nil) when nothing matched.
func parseClause(clause, columnName string) (string, *FieldConstraint) {
	fc := &FieldConstraint{}
	matched := false
	col := columnName

	if m := reCharLenBetween.FindStringSubmatch(clause); m != nil {
		if col == "" {
			col = m[1]
		}
		lo, _ := strconv.Atoi(m[2])
		hi, _ := strconv.Atoi(m[3])
		fc.MinLength = intPtr(lo)
		fc.MaxLength = intPtr(hi)
		matched = true
	}
	if m := reCharLenEQ.FindStringSubmatch(clause); m != nil {
		if col == "" {
			col = m[1]
		}
		n, _ := strconv.Atoi(m[2])
		fc.ExactLength = intPtr(n)
		matched = true
	}
	if m := reCharLenLTE.FindStringSubmatch(clause); m != nil {
		if col == "" {
			col = m[1]
		}
		n, _ := strconv.Atoi(m[2])
		fc.MaxLength = intPtr(n)
		matched = true
	}
	for _, m := range reRegex.FindAllStringSubmatch(clause, -1) {
		candidate := m[1]
		if functionNames[candidate] {
			continue
		}
		if col == "" {
			col = candidate
		}
		fc.Pattern = strPtr(m[2])
		matched = true
	}
	if m := reBetween.FindStringSubmatch(clause); m != nil {
		if !functionNames[m[1]] {
			if col == "" {
				col = m[1]
			}
			lo, _ := strconv.Atoi(m[2])
			hi, _ := strconv.Atoi(m[3])
			fc.Min = intPtr(lo)
			fc.Max = intPtr(hi)
			matched = true
		}
	}
	if m := reGTE.FindStringSubmatch(clause); m != nil {
		if !functionNames[m[1]] {
			if col == "" {
				col = m[1]
			}
			n, _ := strconv.Atoi(m[2])
			fc.Min = intPtr(n)
			matched = true
		}
	}
	if m := reGT.FindStringSubmatch(clause); m != nil {
		if !functionNames[m[1]] {
			if col == "" {
				col = m[1]
			}
			n, _ := strconv.Atoi(m[2])
			fc.Min = intPtr(n + 1)
			matched = true
		}
	}

	if !matched {
		return "", nil
	}
	return col, fc
}

// mergeField adds non-nil scalar fields from src into dst.
func mergeField(dst, src *FieldConstraint) {
	if src.MaxLength != nil {
		dst.MaxLength = src.MaxLength
	}
	if src.MinLength != nil {
		dst.MinLength = src.MinLength
	}
	if src.ExactLength != nil {
		dst.ExactLength = src.ExactLength
	}
	if src.Min != nil {
		dst.Min = src.Min
	}
	if src.Max != nil {
		dst.Max = src.Max
	}
	if src.Pattern != nil {
		dst.Pattern = src.Pattern
	}
	if src.ValidRange {
		dst.ValidRange = true
	}
	if src.RequiredWith != "" {
		dst.RequiredWith = src.RequiredWith
	}
}

// parseRequiredWith detects the co-required pattern:
//
//	(colA IS NULL AND colB IS NULL) OR (colA IS NOT NULL AND colB IS NOT NULL)
//
// Returns (colA, colB) if matched, or ("", "") otherwise.
// The constraint is symmetric — callers should set required_with on both fields.
func parseRequiredWith(clause string) (string, string) {
	// Extract IS NOT NULL columns.
	notNullCols := uniqueWordMatches(reIsNotNull, clause)

	// Extract IS NULL columns: strip IS NOT NULL occurrences first so the
	// IS NULL regex doesn't also match the "NULL" suffix of IS NOT NULL.
	simplified := reIsNotNull.ReplaceAllString(clause, "")
	nullCols := uniqueWordMatches(reIsNull, simplified)

	// Filter infrastructure / function names.
	notNullCols = filterColNames(notNullCols)
	nullCols = filterColNames(nullCols)

	// Co-required: exactly 2 identical columns appear in both sets.
	if len(notNullCols) == 2 && stringSlicesEqual(notNullCols, nullCols) {
		return notNullCols[0], notNullCols[1]
	}
	return "", ""
}

func uniqueWordMatches(re *regexp.Regexp, s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		col := m[1]
		if !seen[col] {
			seen[col] = true
			out = append(out, col)
		}
	}
	sort.Strings(out)
	return out
}

func filterColNames(cols []string) []string {
	var out []string
	for _, c := range cols {
		if !functionNames[c] && !systemColumns[c] {
			out = append(out, c)
		}
	}
	return out
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// parseRangeValidity detects lower(col) < upper(col) — marks the field valid_range.
// Returns ("", false) when not matched.
func parseRangeValidity(clause string) (string, bool) {
	m := reRangeValid.FindStringSubmatch(clause)
	if m == nil {
		return "", false
	}
	// m[1]+m[2] for lower<upper form, m[3]+m[4] for upper>lower form
	a, b := m[1], m[2]
	if a == "" {
		a, b = m[3], m[4]
	}
	if a == b { // same column → valid range constraint
		return a, true
	}
	return "", false
}

// parseCrossCol detects col1 op col2 where neither side is a number or function.
func parseCrossCol(clause string) *Comparison {
	m := reCrossCol.FindStringSubmatch(clause)
	if m == nil {
		return nil
	}
	left, op, right := m[1], m[2], m[3]
	if functionNames[left] || functionNames[right] {
		return nil
	}
	// Ensure neither side looks like a pure number
	if _, err := strconv.Atoi(left); err == nil {
		return nil
	}
	if _, err := strconv.Atoi(right); err == nil {
		return nil
	}
	return &Comparison{Field: left, Operator: op, Other: right}
}

// parseJsonb detects JSONB structure constraints and returns the column name plus
// a map of sub-field rules. Returns ("", nil) when not a JSONB constraint.
func parseJsonb(clause string) (string, map[string]*JsonbSubField) {
	if !strings.Contains(clause, "?") && !strings.Contains(clause, "->>") {
		return "", nil
	}

	col := ""
	fields := make(map[string]*JsonbSubField)

	// key presence: col ? 'key'
	for _, m := range reJsonbHasKey.FindAllStringSubmatch(clause, -1) {
		candidate, key := m[1], m[2]
		if functionNames[candidate] {
			continue
		}
		if col == "" {
			col = candidate
		}
		if fields[key] == nil {
			fields[key] = &JsonbSubField{}
		}
		fields[key].Required = true
	}

	// ->> 'key' IN ('a','b')
	for _, m := range reJsonbInList.FindAllStringSubmatch(clause, -1) {
		candidate, key, list := m[1], m[2], m[3]
		if functionNames[candidate] {
			continue
		}
		if col == "" {
			col = candidate
		}
		vals := extractQuotedValues(list)
		if len(vals) > 0 {
			if fields[key] == nil {
				fields[key] = &JsonbSubField{}
			}
			fields[key].Pattern = strPtr(strings.Join(vals, "|"))
		}
	}

	// ->> 'key' = ANY(ARRAY[...])
	for _, m := range reJsonbAnyArr.FindAllStringSubmatch(clause, -1) {
		candidate, key, arr := m[1], m[2], m[3]
		if functionNames[candidate] {
			continue
		}
		if col == "" {
			col = candidate
		}
		vals := extractQuotedValues(arr)
		if len(vals) > 0 {
			if fields[key] == nil {
				fields[key] = &JsonbSubField{}
			}
			fields[key].Pattern = strPtr(strings.Join(vals, "|"))
		}
	}

	if col == "" || len(fields) == 0 {
		return "", nil
	}
	return col, fields
}

func extractQuotedValues(s string) []string {
	var vals []string
	for _, m := range reQuotedVal.FindAllStringSubmatch(s, -1) {
		vals = append(vals, m[1])
	}
	return vals
}

// repoRoot returns the working directory — the repo root when invoked via
// 'go run ./cmd/tools/sync-constraints/...' from the Makefile.
func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}
	return wd
}

func main() {
	root := repoRoot()

	_ = godotenv.Load(filepath.Join(root, ".env"))

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set — run 'make docker-up' or set DB_URL in .env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	constraints := make(constraintMap)

	// --- 1. CHECK constraints ---
	checkRows, err := conn.Query(ctx, `
		SELECT
		    tc.table_schema,
		    tc.table_name,
		    cc.constraint_name,
		    kcu.column_name,
		    cc.check_clause
		FROM information_schema.table_constraints tc
		JOIN information_schema.check_constraints cc
		    ON tc.constraint_name = cc.constraint_name
		    AND tc.constraint_schema = cc.constraint_schema
		LEFT JOIN information_schema.constraint_column_usage kcu
		    ON tc.constraint_name = kcu.constraint_name
		    AND tc.constraint_schema = kcu.constraint_schema
		WHERE tc.constraint_type = 'CHECK'
		    AND tc.table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY tc.table_schema, tc.table_name, cc.constraint_name
	`)
	if err != nil {
		log.Fatalf("query check constraints: %v", err)
	}

	for checkRows.Next() {
		var schema, table, constraintName string
		var columnName *string
		var checkClause string

		if err := checkRows.Scan(&schema, &table, &constraintName, &columnName, &checkClause); err != nil {
			log.Fatalf("scan check: %v", err)
		}

		clause := stripOuterParens(checkClause)

		col := ""
		if columnName != nil {
			col = *columnName
		}

		if internalFields[col] {
			continue
		}
		if isNullCheck(clause) {
			continue
		}

		key := schema + "." + table
		entry := getOrCreate(constraints, key)

		// Try JSONB expansion first (before other parsers strip partial matches).
		if jsonbCol, jsonbFields := parseJsonb(clause); jsonbCol != "" {
			fc := getOrCreateField(entry, jsonbCol)
			if fc.Jsonb == nil {
				fc.Jsonb = make(map[string]*JsonbSubField)
			}
			for k, v := range jsonbFields {
				if fc.Jsonb[k] == nil {
					fc.Jsonb[k] = v
				} else {
					if v.Required {
						fc.Jsonb[k].Required = true
					}
					if v.Pattern != nil {
						fc.Jsonb[k].Pattern = v.Pattern
					}
				}
			}
			continue
		}

		// Range validity: lower(col) < upper(col)
		if rangeCol, ok := parseRangeValidity(clause); ok {
			getOrCreateField(entry, rangeCol).ValidRange = true
			continue
		}

		// Single-field constraints.
		if detectedCol, fc := parseClause(clause, col); fc != nil && !isEmpty(fc) {
			existing := getOrCreateField(entry, detectedCol)
			mergeField(existing, fc)
			continue
		}

		// Co-required: (colA IS NULL AND colB IS NULL) OR (colA IS NOT NULL AND colB IS NOT NULL)
		if colA, colB := parseRequiredWith(clause); colA != "" {
			getOrCreateField(entry, colA).RequiredWith = colB
			getOrCreateField(entry, colB).RequiredWith = colA
			continue
		}

		// Cross-column comparison.
		if cmp := parseCrossCol(clause); cmp != nil {
			entry.Comparisons = append(entry.Comparisons, *cmp)
			continue
		}

		// Unclassified — store as reference note.
		entry.Notes = append(entry.Notes, fmt.Sprintf("%s: %s", constraintName, clause))
	}
	if err := checkRows.Err(); err != nil {
		log.Fatalf("check rows: %v", err)
	}
	checkRows.Close()

	// --- 2. NOT NULL columns → required: true ---
	nullRows, err := conn.Query(ctx, `
		SELECT table_schema, table_name, column_name
		FROM information_schema.columns
		WHERE is_nullable = 'NO'
		  AND table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name, column_name
	`)
	if err != nil {
		log.Fatalf("query not null: %v", err)
	}
	defer nullRows.Close()

	for nullRows.Next() {
		var schema, table, column string
		if err := nullRows.Scan(&schema, &table, &column); err != nil {
			log.Fatalf("scan not null: %v", err)
		}
		if systemColumns[column] || internalFields[column] {
			continue
		}
		key := schema + "." + table
		entry := getOrCreate(constraints, key)
		getOrCreateField(entry, column).Required = true
	}
	if err := nullRows.Err(); err != nil {
		log.Fatalf("not null rows: %v", err)
	}

	// --- 3. Exclusion (GIST) constraints → notes ---
	gistRows, err := conn.Query(ctx, `
		SELECT
		    n.nspname AS schema,
		    c.relname AS table,
		    con.conname AS name,
		    pg_get_constraintdef(con.oid) AS def
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE con.contype = 'x'
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, c.relname, con.conname
	`)
	if err != nil {
		log.Fatalf("query gist: %v", err)
	}
	defer gistRows.Close()

	for gistRows.Next() {
		var schema, table, name, def string
		if err := gistRows.Scan(&schema, &table, &name, &def); err != nil {
			log.Fatalf("scan gist: %v", err)
		}
		key := schema + "." + table
		entry := getOrCreate(constraints, key)
		entry.Notes = append(entry.Notes, fmt.Sprintf("GIST %s: %s", name, def))
	}
	if err := gistRows.Err(); err != nil {
		log.Fatalf("gist rows: %v", err)
	}

	if err := writeYAML(constraints, filepath.Join(root, "config", "constraints.g.yml")); err != nil {
		log.Fatalf("write yaml: %v", err)
	}
	fmt.Println("✅ config/constraints.g.yml written")

	if err := writeTS(constraints, filepath.Join(root, "web", "src", "lib", "types", "constraints.g.ts")); err != nil {
		log.Fatalf("write ts: %v", err)
	}
	fmt.Println("✅ web/src/lib/types/constraints.g.ts written")
}

// --- YAML output ---

func writeYAML(cm constraintMap, path string) error {
	tables := make([]string, 0, len(cm))
	for k := range cm {
		tables = append(tables, k)
	}
	sort.Strings(tables)

	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	constraintsKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "constraints"}
	constraintsVal := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	for _, table := range tables {
		entry := cm[table]
		tableNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

		// fields section
		if len(entry.Fields) > 0 {
			fieldNames := sortedKeys(entry.Fields)
			fieldsKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "fields"}
			fieldsVal := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

			for _, field := range fieldNames {
				fc := entry.Fields[field]
				if isEmpty(fc) {
					continue
				}
				fieldNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
				appendBoolNode(fieldNode, "required", fc.Required)
				if fc.RequiredWith != "" {
					appendStrNode(fieldNode, "required_with", strPtr(fc.RequiredWith))
				}
				appendBoolNode(fieldNode, "valid_range", fc.ValidRange)
				appendIntNode(fieldNode, "max_length", fc.MaxLength)
				appendIntNode(fieldNode, "min_length", fc.MinLength)
				appendIntNode(fieldNode, "exact_length", fc.ExactLength)
				appendIntNode(fieldNode, "min", fc.Min)
				appendIntNode(fieldNode, "max", fc.Max)
				appendStrNode(fieldNode, "pattern", fc.Pattern)

				// jsonb sub-fields
				if len(fc.Jsonb) > 0 {
					jsonbKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "jsonb"}
					jsonbVal := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
					for _, subKey := range sortedJsonbKeys(fc.Jsonb) {
						sf := fc.Jsonb[subKey]
						sfNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
						appendBoolNode(sfNode, "required", sf.Required)
						appendStrNode(sfNode, "pattern", sf.Pattern)
						appendIntNode(sfNode, "min", sf.Min)
						appendIntNode(sfNode, "max", sf.Max)
						jsonbVal.Content = append(jsonbVal.Content,
							&yaml.Node{Kind: yaml.ScalarNode, Value: subKey},
							sfNode,
						)
					}
					fieldNode.Content = append(fieldNode.Content, jsonbKey, jsonbVal)
				}

				fieldsVal.Content = append(fieldsVal.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: field},
					fieldNode,
				)
			}
			if len(fieldsVal.Content) > 0 {
				tableNode.Content = append(tableNode.Content, fieldsKey, fieldsVal)
			}
		}

		// comparisons section
		if len(entry.Comparisons) > 0 {
			cmpKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "comparisons"}
			cmpVal := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			for _, cmp := range entry.Comparisons {
				item := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
				item.Content = append(item.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "field"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: cmp.Field},
					&yaml.Node{Kind: yaml.ScalarNode, Value: "operator"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: cmp.Operator},
					&yaml.Node{Kind: yaml.ScalarNode, Value: "other"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: cmp.Other},
				)
				cmpVal.Content = append(cmpVal.Content, item)
			}
			tableNode.Content = append(tableNode.Content, cmpKey, cmpVal)
		}

		// notes section
		if len(entry.Notes) > 0 {
			notesKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "notes"}
			notesVal := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			for _, note := range entry.Notes {
				notesVal.Content = append(notesVal.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: note},
				)
			}
			tableNode.Content = append(tableNode.Content, notesKey, notesVal)
		}

		if len(tableNode.Content) == 0 {
			continue
		}
		constraintsVal.Content = append(constraintsVal.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: table},
			tableNode,
		)
	}

	root.Content = append(root.Content, constraintsKey, constraintsVal)
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}

	var buf bytes.Buffer
	buf.WriteString("# Auto-generated by 'make gen-constraints'. Do not edit manually.\n")
	buf.WriteString("# Run 'make gen-constraints' after any schema migration.\n\n")

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func sortedKeys(m map[string]*FieldConstraint) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedJsonbKeys(m map[string]*JsonbSubField) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func appendBoolNode(parent *yaml.Node, key string, val bool) {
	if !val {
		return
	}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
	)
}

func appendIntNode(parent *yaml.Node, key string, val *int) {
	if val == nil {
		return
	}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(*val)},
	)
}

func appendStrNode(parent *yaml.Node, key string, val *string) {
	if val == nil {
		return
	}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: *val},
	)
}

// --- TypeScript output ---

const tsTmpl = `// Auto-generated by 'make gen-constraints'. Do not edit manually.
// Run 'make gen-constraints' after any schema migration.

export const CONSTRAINTS = {
{{- range .Tables}}
  "{{.Name}}": {
    fields: {
    {{- range .Fields}}
      {{.Name}}: { {{.Props}}{{if and .Props .Jsonb}}, {{end}}{{if .Jsonb}}
        jsonb: {
        {{- range .Jsonb}}
          {{.Key}}: { {{.Props}} },
        {{- end}}
        }{{end}} },
    {{- end}}
    },{{if .Comparisons}}
    comparisons: [
    {{- range .Comparisons}}
      { field: "{{.Field}}", operator: "{{.Operator}}", other: "{{.Other}}" },
    {{- end}}
    ],{{end}}{{if .Notes}}
    notes: [
    {{- range .Notes}}
      "{{js .}}",
    {{- end}}
    ],{{end}}
  },
{{- end}}
} as const;

export type ConstraintTable = keyof typeof CONSTRAINTS;
`

type tsTable struct {
	Name        string
	Fields      []tsTSField
	Comparisons []Comparison
	Notes       []string
}

type tsTSField struct {
	Name  string
	Props string
	Jsonb []tsJsonbEntry
}

type tsJsonbEntry struct {
	Key   string
	Props string
}

func writeTS(cm constraintMap, path string) error {
	tables := make([]string, 0, len(cm))
	for k := range cm {
		tables = append(tables, k)
	}
	sort.Strings(tables)

	var tsTables []tsTable
	for _, table := range tables {
		entry := cm[table]

		var tsFields []tsTSField
		for _, field := range sortedKeys(entry.Fields) {
			fc := entry.Fields[field]
			props := buildTSProps(fc)

			var jsonbEntries []tsJsonbEntry
			for _, subKey := range sortedJsonbKeys(fc.Jsonb) {
				sf := fc.Jsonb[subKey]
				jsonbEntries = append(jsonbEntries, tsJsonbEntry{
					Key:   subKey,
					Props: buildTSJsonbProps(sf),
				})
			}

			if props == "" && len(jsonbEntries) == 0 {
				continue
			}
			tsFields = append(tsFields, tsTSField{Name: field, Props: props, Jsonb: jsonbEntries})
		}

		if len(tsFields) == 0 && len(entry.Comparisons) == 0 && len(entry.Notes) == 0 {
			continue
		}
		tsTables = append(tsTables, tsTable{
			Name:        table,
			Fields:      tsFields,
			Comparisons: entry.Comparisons,
			Notes:       entry.Notes,
		})
	}

	tmpl, err := template.New("ts").Parse(tsTmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ Tables []tsTable }{tsTables}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func buildTSProps(fc *FieldConstraint) string {
	var parts []string
	if fc.Required {
		parts = append(parts, "required: true")
	}
	if fc.RequiredWith != "" {
		parts = append(parts, fmt.Sprintf("requiredWith: %q", fc.RequiredWith))
	}
	if fc.ValidRange {
		parts = append(parts, "validRange: true")
	}
	if fc.MaxLength != nil {
		parts = append(parts, fmt.Sprintf("maxLength: %d", *fc.MaxLength))
	}
	if fc.MinLength != nil {
		parts = append(parts, fmt.Sprintf("minLength: %d", *fc.MinLength))
	}
	if fc.ExactLength != nil {
		parts = append(parts, fmt.Sprintf("exactLength: %d", *fc.ExactLength))
	}
	if fc.Min != nil {
		parts = append(parts, fmt.Sprintf("min: %d", *fc.Min))
	}
	if fc.Max != nil {
		parts = append(parts, fmt.Sprintf("max: %d", *fc.Max))
	}
	if fc.Pattern != nil {
		parts = append(parts, fmt.Sprintf("pattern: /%s/", *fc.Pattern))
	}
	return strings.Join(parts, ", ")
}

func buildTSJsonbProps(sf *JsonbSubField) string {
	var parts []string
	if sf.Required {
		parts = append(parts, "required: true")
	}
	if sf.Pattern != nil {
		parts = append(parts, fmt.Sprintf("pattern: /%s/", *sf.Pattern))
	}
	if sf.Min != nil {
		parts = append(parts, fmt.Sprintf("min: %d", *sf.Min))
	}
	if sf.Max != nil {
		parts = append(parts, fmt.Sprintf("max: %d", *sf.Max))
	}
	return strings.Join(parts, ", ")
}
