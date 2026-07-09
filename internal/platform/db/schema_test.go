package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseConventions(t *testing.T) {
	ctx := context.Background()

	t.Run("REQ-002 - Verify UUID Primary Keys", func(t *testing.T) {
		t.Parallel()

		rows, _ := testDB.Query(ctx, `
            SELECT table_name, column_name 
            FROM information_schema.key_column_usage kcu
            JOIN information_schema.columns c USING (table_name, column_name)
            WHERE c.data_type != 'uuid' AND kcu.constraint_name LIKE '%_pkey
			AND kcu.table_schema != 'public'
        `)
		for rows.Next() {
			var table, col string
			if err := rows.Scan(&table, &col); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Table %s: Column %s is a PK but not a UUID", table, col)
		}
	})

	t.Run("REQ-004 - All tables must have audit columns", func(t *testing.T) {
		t.Parallel()

		// Audit logs dont need all columns as it is only created

		query := `
			SELECT
				t.table_name,
				req.col AS missing_column
			FROM (
				SELECT table_name
				FROM information_schema.tables
				WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'public')
					AND table_name NOT LIKE 'pg_%'
					AND table_name != 'audit_logs'
			) t
			CROSS JOIN (VALUES ('created_at'), ('updated_at'), ('deleted_at')) AS req(col)
			LEFT JOIN information_schema.columns c 
				ON c.table_name = t.table_name 
				AND c.column_name = req.col
			WHERE c.column_name IS NULL;
		`

		rows, err := testDB.Query(ctx, query)
		assert.NoError(t, err)
		defer rows.Close()

		var tableFound bool
		for rows.Next() {
			tableFound = true
			var tableName, missingCol string

			err = rows.Scan(&tableName, &missingCol)
			if err != nil {
				t.Fatal(err)
			}

			t.Errorf("Table %s is missing column %s", tableName, missingCol)
		}

		if !tableFound {
			t.Log("WARNING: No tables to search")
		}
	})

	t.Run("REQ-007 - Text columns must have length CHECK or strict regex", func(t *testing.T) {
		t.Parallel()

		query := `
			SELECT 
				cols.table_schema, 
				cols.table_name, 
				cols.column_name
			FROM information_schema.columns cols
			WHERE cols.table_schema IN ('operations', 'finance', 'identity', 'inventory', 'pricing', 'sales_ledgers')
			AND cols.data_type IN ('text', 'character varying', 'citext')
			AND cols.is_generated = 'NEVER'
			AND NOT EXISTS (
				SELECT 1 
				FROM information_schema.constraint_column_usage usage
				JOIN information_schema.check_constraints check_cons 
					ON usage.constraint_name = check_cons.constraint_name
				WHERE usage.table_name = cols.table_name 
					AND usage.column_name = cols.column_name
					AND (check_cons.check_clause LIKE '%char_length%' OR check_cons.check_clause LIKE '%length%' OR check_cons.check_clause LIKE '%~%')
			);
		`

		rows, err := testDB.Query(ctx, query)

		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var schema, table, col string
			if err := rows.Scan(&schema, &table, &col); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Found columns using TEXT without CHECK constraint: %s.%s.%s", schema, table, col)
		}

	})

	t.Run("REQ-009 - Verify TIMESTAMPTZ Usage", func(t *testing.T) {
		t.Parallel()

		rows, err := testDB.Query(ctx, "SELECT table_name, column_name FROM information_schema.columns WHERE data_type = 'timestamp'")
		assert.NoError(t, err)
		defer rows.Close()

		if rows.Next() {
			var table, col string
			if err := rows.Scan(&table, &col); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Found columns using TIMESTAMP instead of TIMESTAMPTZ: %s.%s", table, col)
		}
	})

	t.Run("REQ-011: Every Foreign Key MUST have an explicit Index", func(t *testing.T) {
		t.Parallel()

		// This query finds all single-column FKs that do not have a corresponding index
		// starting with that column. Multi-column FKs (e.g. property consistency pairs)
		// are intentionally excluded — their indexing strategy is assessed separately.
		query := `
			SELECT
				conrelid::regclass AS table_name,
				conname AS foreign_key_name
			FROM pg_constraint c
			WHERE contype = 'f'
			AND cardinality(c.conkey) = 1
			AND NOT EXISTS (
				SELECT 1
				FROM pg_index i
				WHERE i.indrelid = c.conrelid
				AND (i.indkey::int2[])[0] = c.conkey[1]
			)
			AND connamespace::regnamespace::text NOT IN ('pg_catalog', 'information_schema');
		`

		rows, err := testDB.Query(ctx, query)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var table, fk string
			if err := rows.Scan(&table, &fk); err != nil {
				t.Fatal(err)
			}
			t.Errorf("PERFORMANCE RISK: Table %s has FK %s without a supporting index.", table, fk)
		}
	})

	// TODO: Implement this test properly, I have checked
	t.Run("REQ-014 - If a reference depends on property, there must be a FK created references the property and entity", func(t *testing.T) {
		t.Skip("Cba to implement")
		query := `
			SELECT 
				nsp.nspname AS table_schema,
				rel.relname AS table_name,
				con.conname AS constraint_name,
				reltarget.relname AS referenced_table
			FROM pg_constraint con
			JOIN pg_namespace nsp ON nsp.oid = con.connamespace
			JOIN pg_class rel ON rel.oid = con.conrelid
			JOIN pg_class reltarget ON reltarget.oid = con.confrelid
			WHERE con.contype = 'f' 
			AND nsp.nspname NOT IN ('information_schema', 'pg_catalog')
			-- 1. Ensure the TARGET table has a property_id column
			AND EXISTS (
				SELECT 1 FROM pg_attribute a 
				WHERE a.attrelid = con.confrelid 
				AND a.attname = 'property_id' 
				AND NOT a.attisdropped
			)
			-- 2. THE FIX: Check if 'property_id' is NOT among the columns of THIS foreign key
			AND 'property_id' NOT IN (
				SELECT attname 
				FROM pg_attribute 
				WHERE attrelid = con.conrelid 
				AND attnum = ANY(con.conkey)
			);
		`

		rows, err := testDB.Query(ctx, query)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var schema, table, col, refTable string
			if err := rows.Scan(&schema, &table, &col, &refTable); err != nil {
				t.Fatal(err)
			}

			t.Errorf("Multi-Tenancy Violation: The following FK must be composite %s.%s.%s referencing %s", schema, table, col, refTable)
		}
	})

	t.Run("REQ-015 - Verify Boolean Defaults", func(t *testing.T) {
		t.Parallel()

		rows, err := testDB.Query(ctx, `
			SELECT table_name, column_name 
			FROM information_schema.columns 
			WHERE data_type = 'boolean' 
			AND column_default IS NULL 
			AND table_schema != 'public'
			AND table_name NOT LIKE 'pg_%'
		`)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var table, col string
			if err := rows.Scan(&table, &col); err != nil {
				t.Fatal(err)
			}
			t.Errorf("Boolean column %s.%s missing default value", table, col)
		}
	})
}
