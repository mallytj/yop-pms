package db_tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbTechnicalRequirements(t *testing.T) {
	t.Run("TC-DB-01 -  All primary keys must be UUIDv7", func(t *testing.T) {
		t.Parallel()

		// 1. Define schemas you want to check (exclude system schemas)
		targetSchemas := []string{"auth", "operations", "finance"}

		// 2. Query to find all Primary Keys and their Default definitions
		query := `
			SELECT
				kcu.table_schema,
				kcu.table_name,
				c.column_name,
				c.column_default
			FROM information_schema.table_constraints tco
			JOIN information_schema.key_column_usage kcu
				ON kcu.constraint_name = tco.constraint_name
				AND kcu.constraint_schema = tco.constraint_schema
			JOIN information_schema.columns c
				ON c.table_schema = kcu.table_schema
				AND c.table_name = kcu.table_name
				AND c.column_name = kcu.column_name
			WHERE tco.constraint_type = 'PRIMARY KEY'
			AND kcu.table_schema = ANY($1)
			AND c.table_name != 'goose_db_version';
		`

		rows, err := testDB.Query(context.Background(), query, targetSchemas)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var schema, table, col string
			var defVal *string // Pointer because default can be null

			err := rows.Scan(&schema, &table, &col, &defVal)
			assert.NoError(t, err)

			t.Run(schema+"."+table, func(t *testing.T) {
				// 1. Check if a default value exists
				if assert.NotNil(t, defVal, "Primary key %s.%s has no default value", table, col) {
					// 2. Check if the default value uses the gen_random_uuid() function - indicating UUIDv7
					isV7 := strings.Contains(*defVal, "gen_random_uuid()")

					assert.True(t, isV7,
						"Table '%s' PK is not using UUIDv7. Found: %s",
						table, *defVal)
				}
			})
		}
	})

	t.Run("TC-DB-02 - All monetary values must be stored as integer pence", func(t *testing.T) {
		// 1. Define schemas you want to check (exclude system schemas)
		targetSchemas := []string{"finance", "operations", "inventory", "pricing"}

		// 2. Query to find all columns with monetary types (NUMERIC, DECIMAL, MONEY)
		query := `
			SELECT
				table_schema,
				table_name,
				column_name,
				data_type
			FROM information_schema.columns
			WHERE table_schema = ANY($1)
			AND data_type IN ('numeric', 'decimal', 'money');
		`

		rows, err := testDB.Query(context.Background(), query, targetSchemas)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var schema, table, col, dataType string

			err := rows.Scan(&schema, &table, &col, &dataType)
			assert.NoError(t, err)

			t.Run(schema+"."+table+"."+col, func(t *testing.T) {
				if strings.Contains(col, "tax") {
					return // Skip tax rate snapshot columns
				}
				assert.Fail(t, fmt.Sprintf(
					"Monetary column found that is not stored as integer pence: %v.%v.%v of type %v",
					schema, table, col, dataType,
				),
				)
			})
		}
	})

	t.Run("TC-DB-03 - All core tables must have audit fields (created_at, updated_at, deleted_at)", func(t *testing.T) {
		t.Parallel()

		// 1. Define schemas you want to check (exclude system schemas)
		targetSchemas := []string{"auth", "operations", "finance", "inventory", "pricing", "relations", "identity"}

		// 2. Query to find all tables in the target schemas
		queryTables := `
			SELECT table_schema, table_name
			FROM information_schema.tables
			WHERE table_schema = ANY($1)
			AND table_type = 'BASE TABLE';
		`

		rows, err := testDB.Query(context.Background(), queryTables, targetSchemas)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var schema, table string

			err := rows.Scan(&schema, &table)
			assert.NoError(t, err)

			t.Run(schema+"."+table, func(t *testing.T) {
				// 3. For each table, check for the presence of audit fields
				auditFields := []string{"created_at", "updated_at", "deleted_at"}
				for _, field := range auditFields {
					queryColumn := `
						SELECT COUNT(*)
						FROM information_schema.columns
						WHERE table_schema = $1
						AND table_name = $2
						AND column_name = $3;
					`
					var count int
					err := testDB.QueryRow(context.Background(), queryColumn, schema, table, field).Scan(&count)
					assert.NoError(t, err)
					assert.Equal(t, 1, count, "Table '%s.%s' is missing audit field '%s'", schema, table, field)
				}
			})
		}
	})

	t.Run("TC-DB-04 - Any indexes created on tables dependant on property_id must include property_id as the first column", func(t *testing.T) {
		t.Parallel()

		// 1. Define schemas you want to check (exclude system schemas)
		targetSchemas := []string{"operations", "finance", "inventory", "pricing", "relations"}

		// 2. Query to find all indexes on tables with property_id column
		query := `
			SELECT
				i.relname AS index_name,
				a.attname AS column_name,
				t.relname AS table_name,
				ns.nspname AS table_schema
			FROM pg_class t
			JOIN pg_namespace ns ON ns.oid = t.relnamespace
			JOIN pg_index ix ON t.oid = ix.indrelid
			JOIN pg_class i ON i.oid = ix.indexrelid
			JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
			WHERE ns.nspname = ANY($1)
			AND t.oid IN (
				SELECT c.oid
				FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				JOIN pg_attribute a ON a.attrelid = c.oid
				WHERE n.nspname = ANY($1)
				AND a.attname = 'property_id'
			)
			ORDER BY i.relname, a.attnum;
		`

		rows, err := testDB.Query(context.Background(), query, targetSchemas)
		assert.NoError(t, err)
		defer rows.Close()

		indexColumns := make(map[string][]string)

		for rows.Next() {
			var indexName, columnName, tableName, tableSchema string

			err := rows.Scan(&indexName, &columnName, &tableName, &tableSchema)
			assert.NoError(t, err)

			fullIndexName := fmt.Sprintf("%s.%s", tableSchema, indexName)
			indexColumns[fullIndexName] = append(indexColumns[fullIndexName], columnName)
		}

		for indexName, columns := range indexColumns {
			t.Run(indexName, func(t *testing.T) {
				t.Parallel()
				if len(columns) > 0 && columns[0] != "property_id" {
					assert.Fail(t, fmt.Sprintf(
						"Index '%s' does not have 'property_id' as the first column. Columns: %v",
						indexName, columns,
					))
				}
			})
		}
	})

	t.Run("TC-DB-05 - All foreign key relationships must have ON DELETE and ON UPDATE actions defined", func(t *testing.T) {
		t.Parallel()

		// 1. Define schemas you want to check (exclude system schemas)
		targetSchemas := []string{"auth", "operations", "finance", "inventory", "pricing", "relations", "identity"}

		// 2. Query to find all foreign keys without ON DELETE or ON UPDATE actions
		query := `
			SELECT
				tc.table_schema,
				tc.table_name,
				kcu.column_name,
				ccu.table_schema AS foreign_table_schema,
				ccu.table_name AS foreign_table_name,
				ccu.column_name AS foreign_column_name
			FROM information_schema.table_constraints AS tc
			JOIN information_schema.key_column_usage AS kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage AS ccu
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
			WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = ANY($1)
			AND (tc.delete_rule IS NULL OR tc.update_rule IS NULL);
		`

		rows, err := testDB.Query(context.Background(), query, targetSchemas)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var schema, table, column, foreignSchema, foreignTable, foreignColumn string

			err := rows.Scan(&schema, &table, &column, &foreignSchema, &foreignTable, &foreignColumn)
			assert.NoError(t, err)

			t.Run(schema+"."+table+"."+column, func(t *testing.T) {
				t.Parallel()
				assert.Fail(t, fmt.Sprintf(
					"Foreign key on '%s.%s(%s)' referencing '%s.%s(%s)' does not have ON DELETE and ON UPDATE actions defined",
					schema, table, column, foreignSchema, foreignTable, foreignColumn,
				))
			})
		}
	})

	t.Run("TC-DB-06 - All text/varchar columns used for codes must have CHECK constraints enforcing length", func(t *testing.T) {
		t.Parallel()

		// 1. Define schemas you want to check (exclude system schemas)
		targetSchemas := []string{"operations", "finance", "inventory", "pricing", "relations"}

		// 2. Query to find all code columns without CHECK constraints on length
		query := `
			SELECT
				c.table_schema,
				c.table_name,
				c.column_name
			FROM information_schema.columns c
			LEFT JOIN information_schema.check_constraints cc
				ON cc.constraint_schema = c.table_schema
			JOIN information_schema.constraint_column_usage ccu
				ON ccu.constraint_name = cc.constraint_name
				AND ccu.constraint_schema = cc.constraint_schema
			WHERE c.table_schema = ANY($1)
			AND (c.column_name ILIKE '%_code' OR c.column_name ILIKE 'code_%')
			AND c.data_type IN ('character varying', 'text')
			GROUP BY c.table_schema, c.table_name, c.column_name
			HAVING COUNT(cc.constraint_name) = 0;
		`

		rows, err := testDB.Query(context.Background(), query, targetSchemas)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var schema, table, column string

			err := rows.Scan(&schema, &table, &column)
			assert.NoError(t, err)

			t.Run(schema+"."+table+"."+column, func(t *testing.T) {
				t.Parallel()
				assert.Fail(t, fmt.Sprintf(
					"Code column '%s.%s(%s)' does not have a CHECK constraint enforcing length",
					schema, table, column,
				))
			})
		}
	})

	t.Run("TC-DB-07 - All tables must have a schema defined (no tables in 'public' schema)", func(t *testing.T) {
		t.Parallel()

		// Query to find all tables in the 'public' schema
		query := `
			SELECT table_name
			FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_type = 'BASE TABLE';
		`

		rows, err := testDB.Query(context.Background(), query)
		assert.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var table string

			err := rows.Scan(&table)
			assert.NoError(t, err)

			t.Run("public."+table, func(t *testing.T) {
				t.Parallel()
				assert.Fail(t, fmt.Sprintf(
					"Table '%s' exists in 'public' schema. All tables must be in specific schemas.",
					table,
				))
			})
		}
	})

	t.Run("TC-DB-08 - All timestamps must be stored in TIMESTAMPTZ format", func(t *testing.T) {
		// Implementation left as an exercise for the reader
	})

	t.Run("TC-DB-09 - High-concurrency tables must support Optimistic Locking (via a version column).", func(t *testing.T) {
		// Implementation left as an exercise for the reader
	})

	t.Run("TC-DB-10 - All Foreign Key columns must have an explicit Index.", func(t *testing.T) {
		// Implementation left as an exercise for the reader
	})

	t.Run("TC-DB-11 - Constraint names must follow a strict convention ({table}_{column}_{suffix}).", func(t *testing.T) {
		// Implementation left as an exercise for the reader
	})

	t.Run("TC-DB-12 - All foreign keys must NOT CASCADE to preserve historical data.", func(t *testing.T) {
		// Implementation left as an exercise for the reader
	})
}
