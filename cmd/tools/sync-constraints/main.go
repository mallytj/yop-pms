// sync-constraints queries the live Postgres database for CHECK constraints,
// parses user-facing validation rules, and writes:
//   - config/constraints.g.yml  (shared source of truth)
//   - web/src/lib/types/constraints.g.ts  (TypeScript constants for the frontend)
//
// Run via: make gen-constraints
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

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
	dbURL := os.ExpandEnv(os.Getenv("DB_URL"))
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
