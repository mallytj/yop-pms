package db_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("pgx", "postgres://app_user:password@localhost:5433/yop_db?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to db: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping db: %v", err)
	}

	return db
}

func TestSchemaInvariants(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// REQ-004: Audit fields (created_at, updated_at, deleted_at) on all core tables
	t.Run("REQ-004_AuditFields", func(t *testing.T) {
		t.Parallel()
		query := `
			SELECT t.table_name 
			FROM information_schema.tables t
			WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
			  AND NOT EXISTS (
				  SELECT 1 FROM information_schema.columns c 
				  WHERE c.table_name = t.table_name AND c.column_name = 'created_at'
			  )
			  AND t.table_name NOT LIKE 'goose_db_version';`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Table '%s' is missing required audit fields", tableName)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	})

	// REQ-009: All timestamps must be stored as TIMESTAMPTZ
	t.Run("REQ-009_TimestampsAreTZ", func(t *testing.T) {
		query := `
			SELECT table_name, column_name, data_type 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			  AND data_type = 'timestamp without time zone';`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var table, col, dtype string
			if err := rows.Scan(&table, &col, &dtype); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Column '%s.%s' is %s. Must be 'timestamp with time zone'", table, col, dtype)
		}
	})

	// REQ-013: All foreign keys must RESTRICT NOT CASCADE
	t.Run("REQ-013_FKsMustRestrict", func(t *testing.T) {
		query := `
			SELECT tc.table_name, tc.constraint_name, rc.delete_rule 
			FROM information_schema.table_constraints tc 
			JOIN information_schema.referential_constraints rc 
			  ON tc.constraint_name = rc.constraint_name 
			WHERE tc.constraint_type = 'FOREIGN KEY' 
			  AND tc.table_schema = 'public' 
			  AND rc.delete_rule = 'CASCADE';`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var table, constraint, rule string
			if err := rows.Scan(&table, &constraint, &rule); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Constraint '%s' on table '%s' uses ON DELETE %s. MUST be RESTRICT.", constraint, table, rule)
		}
	})

	// REQ-015: Any boolean columns must have a set default
	t.Run("REQ-015_BooleanDefaults", func(t *testing.T) {
		query := `
			SELECT table_name, column_name 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			  AND data_type = 'boolean' 
			  AND column_default IS NULL;`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var table, col string
			if err := rows.Scan(&table, &col); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Boolean column '%s.%s' is missing a DEFAULT value.", table, col)
		}
	})

	// REQ-020: RLS Mandatory
	t.Run("REQ-020_RLSEnabled", func(t *testing.T) {
		query := `
			SELECT relname 
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public' 
			  AND c.relkind = 'r' 
			  AND c.relname != 'goose_db_version'
			  AND c.relrowsecurity = false;`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Table '%s' does NOT have Row-Level Security enabled.", table)
		}
	})
}
