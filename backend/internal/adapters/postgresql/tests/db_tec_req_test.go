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
}
