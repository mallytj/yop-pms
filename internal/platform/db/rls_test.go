package db_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/stretchr/testify/assert"
)

func TestRowLevelSecurity(t *testing.T) {

	ctx := context.Background()

	licence := seedLicence(ctx)
	propertyAID := seedProperty(ctx, licence)
	propertyBID := seedProperty(ctx, licence)

	tx, err := testDB.BeginTx(ctx, pgx.TxOptions{})
	assert.NoError(t, err)

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx) // Quietly rollback if not committed
		}
	}()
	_, err = tx.Exec(ctx, "SELECT set_config('app.current_property_id', $1, true)", propertyAID)
	assert.NoError(t, err)

	_, err = tx.Exec(ctx, "INSERT INTO inventory.room_types (property_id, name, code) VALUES ($1, $2, $3)", propertyAID, "Guest A", "RT-01")
	assert.NoError(t, err)

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
	committed = true

	t.Run("REQ-022: Test Tenant Isolation Strict", func(t *testing.T) {
		t.Parallel()

		tx, err := testUserDB.BeginTx(ctx, pgx.TxOptions{})
		defer func() {
			if err := tx.Rollback(ctx); err != nil {
				t.Fatalf("error rolling back transaction: %v", err)
			}
		}()

		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// REQ-022: Set context to Property B
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_property_id', $1, true)", propertyBID)
		if err != nil {
			t.Fatalf("Failed to set context: %v", err)
		}

		var count int
		// This should return 0 rows even though the record exists in the table
		err = tx.QueryRow(ctx, "SELECT count(*) FROM inventory.room_types").Scan(&count)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count > 0 {
			t.Errorf("SECURITY BREACH: Property B accessed Property A data!")
		}
	})

	t.Run("REQ-022b: Test Tenant Isolation Happy Path", func(t *testing.T) {
		t.Parallel()

		tx, err := testUserDB.BeginTx(ctx, pgx.TxOptions{})
		defer func() {
			if err := tx.Rollback(ctx); err != nil {
				t.Fatalf("error rolling back transaction: %v", err)
			}
		}()

		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// REQ-022: Set context to Property A
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_property_id', $1, true)", propertyAID)
		if err != nil {
			t.Fatalf("Failed to set context: %v", err)
		}

		var count int
		// This should return 1 row
		err = tx.QueryRow(ctx, "SELECT count(*) FROM inventory.room_types").Scan(&count)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if count != 1 {
			t.Errorf("SECURITY BREACH: Property A accessed Property A data!")
		}
	})

	t.Run("REQ-022c: No config returns no rows", func(t *testing.T) {
		t.Parallel()

		tx, err := testUserDB.BeginTx(ctx, pgx.TxOptions{})
		defer func() {
			if err := tx.Rollback(ctx); err != nil {
				t.Fatalf("error rolling back transaction: %v", err)
			}
		}()

		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		var count int
		// This should return 0 rows even though the record exists in the table
		err = tx.QueryRow(ctx, "SELECT count(*) FROM inventory.room_types").Scan(&count)

		if err != nil {
			// This is expected as the config is not set
			assert.True(t, helpers.CheckErrorCode(err, helpers.UndefinedObjectCode))
		}
		if count > 0 {
			t.Errorf("SECURITY BREACH: Property B accessed Property A data!")
		}
	})

	t.Run("REQ-020: All tables with property_id column MUST have RLS enabled", func(t *testing.T) {
		t.Parallel()

		query := `
			SELECT 
				nspname AS schema,
				relname AS table
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			JOIN pg_attribute a ON a.attrelid = c.oid
			WHERE nspname IN ('public', 'inventory', 'operations', 'finance', 'pricing', 'sales_ledgers', 'identity', 'auth', 'relations') 
			AND a.attname = 'property_id'
			AND c.relkind = 'r' -- regular tables only
			AND (NOT c.relrowsecurity);
		`

		rows, err := testDB.Query(ctx, query)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var schema, table string
			if err := rows.Scan(&schema, &table); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			t.Errorf("Table %s.%s does not have RLS enabled!", schema, table)
		}
	})
}
