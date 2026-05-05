package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	notNullCols := uniqueWordMatches(reIsNotNull, clause)

	// Strip IS NOT NULL occurrences first so the IS NULL regex doesn't also
	// match the "NULL" suffix of IS NOT NULL.
	simplified := reIsNotNull.ReplaceAllString(clause, "")
	nullCols := uniqueWordMatches(reIsNull, simplified)

	notNullCols = filterColNames(notNullCols)
	nullCols = filterColNames(nullCols)

	if len(notNullCols) == 2 && stringSlicesEqual(notNullCols, nullCols) {
		return notNullCols[0], notNullCols[1]
	}
	return "", ""
}

func uniqueWordMatches(re interface{ FindAllStringSubmatch(string, int) [][]string }, s string) []string {
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

// suppress unused import — fmt is used in main.go but parse.go needs it for
// parseCrossCol callers; keep the reference here.
var _ = fmt.Sprintf
