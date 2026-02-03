package db_tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ExistenceTest struct {
	name     string
	testCase string
}

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

		enumTests := []ExistenceTest{
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
			{"inventory.maintenance_block_type", "TC-MAINT-05"},
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

		tableTests := []ExistenceTest{
			{"operations.licences", "TC-LICE-01"},
			{"operations.properties", "TC-PROP-01"},
			{"auth.users", "TC-USER-01"},
			{"identity.guests", "TC-GUEST-01"},
			{"auth.audit_logs", "TC-AUDIT-01"},
			{"operations.amenities", "TC-AMEN-01"},
			{"relations.property_amenities", "TC-PRAM-01"},
			{"identity.travel_agents", "TC-TRAV-01"},
			{"inventory.room_types", "TC-RTYPE-01"},
			{"relations.room_type_amenities", "TC-RTYPE-15"},
			{"inventory.rooms", "TC-ROOM-01"},
			{"relations.room_amenities", "TC-RMAM-01"},
			{"inventory.maintenance_blocks", "TC-MAINT-01"},
			{"pricing.rate_plans", "TC-RPLAN-01"},
			{"identity.company_profiles", "TC-CPROF-01"},
			{"finance.tax_rules", "TC-TAXR-01"},
			{"finance.ledger_codes", "TC-LEDC-01"},
			{"sales_ledgers.accounts", "TC-SLACC-01"},
			{"operations.reservation_groups", "TC-RGRP-01"},
			{"operations.reservations", "TC-RESV-01"},
			{"operations.reservation_items", "TC-RESI-01"},
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

	t.Run("Indexes Created", func(t *testing.T) {
		t.Parallel()

		indexTests := []ExistenceTest{
			{"idx_licence_properties_name", "TC-PROP-13"},
			{"idx_licence_properties_active", "TC-PROP-04"},
			{"idx_properties_licence", "TC-PROP-11"},
			{"idx_users_licence", "TC-USER-22"},
			{"idx_users_name", "TC-USER-23"},
			{"idx_users_email", "TC-USER-27"},
			{"idx_users_role", "TC-USER-24"},
			{"idx_users_active", "TC-USER-25"},
			{"idx_property_guests_name", "TC-GUEST-14"},
			{"idx_property_guests_email", "TC-GUEST-15"},
			{"idx_property_guests_phone", "TC-GUEST-16"},
			{"idx_property_guests_marketing_opt_in", "TC-GUEST-17"},
			{"idx_property_guests_anonymised", "TC-GUEST-18"},
			{"idx_property_guests", "TC-GUEST-19"},
			{"idx_property_audit_logs_user", "TC-AUDIT-09"},
			{"idx_property_audit_logs_entity", "TC-AUDIT-10"},
			{"idx_property_audit_logs_action", "TC-AUDIT-11"},
			{"idx_amenities_property", "TC-AMEN-09"},
			{"idx_amenities_active_by_property", "TC-AMEN-10"},
			{"idx_property_amenities_property", "TC-PRAM-04"},
			{"idx_property_amenities_amenity", "TC-PRAM-05"},
			{"idx_travel_agents_property", "TC-TRAV-10"},
			{"idx_room_types_property", "TC-RTYPE-14"},
			{"idx_room_type_amenities_room_type", "TC-RTYPE-18"},
			{"idx_room_type_amenities_amenity", "TC-RTYPE-19"},
			{"idx_rooms_room_type", "TC-ROOM-08"},
			{"idx_housekeeping_status", "TC-ROOM-09"},
			{"idx_occupancy_status", "TC-ROOM-10"},
			{"idx_rooms_property", "TC-ROOM-11"},
			{"idx_property_room_amenities_room", "TC-RMAM-04"},
			{"idx_property_room_amenities_amenity", "TC-RMAM-05"},
			{"idx_maintenance_blocks_room_period", "TC-MAINT-08"},
			{"idx_maintenance_blocks_period", "TC-MAINT-09"},
			{"idx_maintenance_blocks_type", "TC-MAINT-10"},
			{"idx_maintenance_blocks_created_by", "TC-MAINT-11"},
			{"idx_maintenance_blocks_room", "TC-MAINT-12"},
			{"idx_rate_plans_property", "TC-RPLAN-09"},
			{"idx_property_rate_plans_parent_rate_plan", "TC-RPLAN-10"},
			{"idx_property_rate_plans_active", "TC-RPLAN-11"},
			{"idx_company_profiles_property", "TC-CPROF-13"},
			{"idx_company_profiles_negotiated_rate_plan", "TC-CPROF-14"},
			{"idx_tax_rules_property", "TC-TAXR-09"},
			{"idx_ledger_codes_property", "TC-LEDC-08"},
			{"idx_ledger_codes_tax_rule", "TC-LEDC-09"},
			{"idx_accounts_property", "TC-SLACC-07"},
			{"idx_property_accounts_company_profile", "TC-SLACC-08"},
			{"idx_reservation_groups_property", "TC-RGRP-11"},
			{"idx_reservations_property", "TC-RESV-11"},
			{"idx_reservations_primary_guest", "TC-RESV-12"},
			{"idx_reservations_group", "TC-RESV-13"},
			{"idx_reservations_travel_agent", "TC-RESV-14"},
			{"idx_reservations_status", "TC-RESV-15"},
			{"idx_reservations_source", "TC-RESV-16"},
			{"idx_reservation_items_reservation", "TC-RESI-21"},
			{"idx_reservation_items_assigned_room", "TC-RESI-22"},
			{"idx_reservation_items_booked_room_type", "TC-RESI-23"},
			{"idx_reservation_items_rate_plan", "TC-RESI-24"},
			{"idx_reservation_items_status", "TC-RESI-25"},
			{"idx_reservation_items_stay_period", "TC-RESI-26"},
		}

		for _, it := range indexTests {
			t.Run(it.testCase+" - Index: "+it.name, func(t *testing.T) {
				t.Parallel()

				var exists bool
				err := testDB.QueryRow(context.Background(),
					`SELECT EXISTS (
						SELECT 1
						FROM pg_indexes
						WHERE indexname = $1
					)`, it.name).Scan(&exists)

				assert.NoError(t, err)

				assert.True(t, exists, "Index %s does not exist", it.name)
			})
		}
	})

	t.Run("Functions Created", func(t *testing.T) {
		t.Parallel()
		functionTests := []ExistenceTest{
			{"operations.check_licence_is_active", "TC-PROP-06"},
			{"operations.fn_validate_room_occupancy", "TC-RESI-31"},
		}

		for _, ft := range functionTests {
			t.Run(ft.testCase+" - Function: "+ft.name, func(t *testing.T) {
				t.Parallel()

				schema, functionName := splitEnumName(ft.name)

				var exists bool
				err := testDB.QueryRow(context.Background(),
					`SELECT EXISTS (
						SELECT 1
						FROM pg_proc p
						JOIN pg_namespace n ON p.pronamespace = n.oid
						WHERE n.nspname = $1
						AND p.proname = $2
					)`, schema, functionName).Scan(&exists)
				assert.NoError(t, err)
				assert.True(t, exists, "Function %s does not exist in schema %s", functionName, schema)
			})
		}
	})
}
