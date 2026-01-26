package db_tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitMigration(t *testing.T) {
	t.Run("TC-MIG-01 - Initialise Extensions (uuid-ossp, pgcrypto)", func(t *testing.T) {
		t.Parallel()

		// 1. Check if uuid-ossp is installed
		var uuidOsspInstalled bool
		err := testDB.QueryRow(context.Background(),
			`SELECT EXISTS (
				SELECT 1
				FROM pg_extension
				WHERE extname = 'uuid-ossp'
			)`).Scan(&uuidOsspInstalled)
		assert.NoError(t, err)
		assert.True(t, uuidOsspInstalled, "uuid-ossp extension is not installed")
		t.Run("TC-MIG-02 - Initialise Extensions (pgcrypto)", func(t *testing.T) {
			t.Parallel()

			// 2. Check if pgcrypto is installed
			var pgcryptoInstalled bool
			err := testDB.QueryRow(context.Background(),
				`SELECT EXISTS (
					SELECT 1
					FROM pg_extension
					WHERE extname = 'pgcrypto'
				)`).Scan(&pgcryptoInstalled)
			assert.NoError(t, err)
			assert.True(t, pgcryptoInstalled, "pgcrypto extension is not installed")
		})
	})

	t.Run("TC-MIG-02 - Create empty schemas", func(t *testing.T) {
		t.Parallel()

		schemas := []string{"auth", "identity", "operations", "inventory", "finance", "relations"}

		for _, schema := range schemas {
			t.Run("Schema: "+schema, func(t *testing.T) {
				t.Parallel()

				var exists bool
				err := testDB.QueryRow(context.Background(),
					`SELECT EXISTS (
						SELECT 1
						FROM information_schema.schemata
						WHERE schema_name = $1
					)`, schema).Scan(&exists)
				assert.NoError(t, err)
				assert.True(t, exists, "Schema %s does not exist", schema)
			})
		}
	})

	t.Run("Enum Types Created", func(t *testing.T) {
		t.Parallel()

		type enumTest struct {
			name     string
			testCase string
		}

		enumTests := []enumTest{
			{"auth.user_role", "TC-USER-21"},
			{"auth.audit_log_entity", "TC-AUDIT-07"},
			{"auth.audit_log_action", "TC-AUDIT-08"},
			{"identity.identity_doc_type", "TC-IDDOC-07"},
			{"inventory.housekeeping_status", "TC-ROOM-05"},
			{"inventory.occupancy_status", "TC-ROOM-06"},
			{"operations.reservation_source", "TC-RESV-07"},
			{"operations.reservation_status", "TC-RESV-08"},
			{"operations.reservation_guest_role", "TC-RESG-04"},
			{"operations.reservation_item_status", "TC-RESI-19"},
			{"inventory.maintenance_block_types", "TC-MAINT-05"},
		}

		for _, et := range enumTests {
			t.Run(et.testCase+" - Enum Type: "+et.name, func(t *testing.T) {
				t.Parallel()

				schema, typeName := splitEnumName(et.name)

				var exists bool
				err := testDB.QueryRow(context.Background(),
					`SELECT EXISTS (
						SELECT 1
						FROM pg_type t
						JOIN pg_namespace n ON t.typnamespace = n.oid
						WHERE n.nspname = $2
						AND t.typname = $1
						AND t.typtype = 'e'
					)`, typeName, schema).Scan(&exists)
				assert.NoError(t, err)
				assert.True(t, exists, "Enum type %s does not exist in schema %s", typeName, schema)
			})
		}
	})

	t.Run("Tables Created", func(t *testing.T) {
		t.Parallel()

		type tableTest struct {
			name     string
			testCase string
		}

		tableTests := []tableTest{
			{"auth.users", "TC-USER-01"},
			{"operations.properties", "TC-PROP-01"},
			{"inventory.rooms", "TC-ROOM-01"},
			{"finance.pricing_blocks", "TC-PRICING-01"},
		}

		for _, tt := range tableTests {
			t.Run(tt.testCase+" - Table: "+tt.name, func(t *testing.T) {
				t.Parallel()

				schema, tableName := splitEnumName(tt.name)

				var exists bool
				err := testDB.QueryRow(context.Background(),
					`SELECT EXISTS (
						SELECT 1
						FROM information_schema.tables
						WHERE table_schema = $1
						AND table_name = $2
					)`, schema, tableName).Scan(&exists)
				assert.NoError(t, err)
				assert.True(t, exists, "Table %s does not exist in schema %s", tableName, schema)
			})
		}
	})
}
