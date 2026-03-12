-- +goose Up
-- +goose StatementBegin
ALTER TABLE operations.amenities ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.reservation_groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.reservations ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.reservation_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.checkout_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity.guests ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity.travel_agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity.identity_docs ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity.company_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.rooms ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.room_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.maintenance_blocks ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.room_inventory_ledger ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.rate_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.daily_price_grid ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.booked_daily_rates ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.tax_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.ledger_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.folios ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.folio_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_ledgers.accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_ledgers.transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE relations.property_amenities ENABLE ROW LEVEL SECURITY;
ALTER TABLE relations.room_type_amenities ENABLE ROW LEVEL SECURITY;
ALTER TABLE relations.room_amenities ENABLE ROW LEVEL SECURITY;
ALTER TABLE relations.reservation_item_guests ENABLE ROW LEVEL SECURITY;
DO $$
DECLARE r RECORD;
BEGIN -- Loop through all tables that have RLS enabled in your specific schemas
FOR r IN
SELECT n.nspname AS schema_name,
    c.relname AS table_name
FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relrowsecurity = true -- Only tables where you ran ENABLE RLS
    AND n.nspname IN (
        'operations',
        'auth',
        'identity',
        'inventory',
        'pricing',
        'finance',
        'sales_ledgers',
        'relations'
    ) LOOP -- Ensure RLS is enforced
    EXECUTE format(
        'ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY;',
        r.schema_name,
        r.table_name
    );
-- Create a policy for property_id
EXECUTE format(
    $fmt$ CREATE POLICY enforce_property_consistency ON %I. %I USING (
        property_id = NULLIF(
            current_setting('app.current_property_id', TRUE),
            ''
        )::UUID
    );
$fmt$,
r.schema_name,
r.table_name,
r.schema_name,
r.table_name
);
RAISE NOTICE 'RLS policy created for %.%',
r.schema_name,
r.table_name;
END LOOP;
END $$;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE r RECORD;
BEGIN -- Loop through all tables in the specified schemas
FOR r IN
SELECT n.nspname AS schema_name,
    c.relname AS table_name
FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname IN (
        'operations',
        'auth',
        'identity',
        'inventory',
        'pricing',
        'finance',
        'sales_ledgers',
        'relations'
    )
    AND c.relkind = 'r' -- Only target actual tables
    LOOP -- 1. Disable RLS
    EXECUTE format(
        'ALTER TABLE %I.%I DISABLE ROW LEVEL SECURITY',
        r.schema_name,
        r.table_name
    );
-- 2. Turn off Forced RLS
EXECUTE format(
    'ALTER TABLE %I.%I NO FORCE ROW LEVEL SECURITY',
    r.schema_name,
    r.table_name
);
-- 3. Drop the tenant isolation policy
EXECUTE format(
    'DROP POLICY IF EXISTS enforce_property_consistency ON %I.%I',
    r.schema_name,
    r.table_name
);
RAISE NOTICE 'Removed RLS and policy from %.%',
r.schema_name,
r.table_name;
END LOOP;
END $$;
-- +goose StatementEnd