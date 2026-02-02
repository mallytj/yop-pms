-- +goose Up
-- Create tax rules table
CREATE TABLE IF NOT EXISTS
    finance.tax_rules (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        name TEXT NOT NULL,
        description TEXT,
        tax_percentage NUMERIC(5, 2) NOT NULL CHECK(tax_percentage >= 0.00 AND tax_percentage <= 75.00), -- e.g., 10.00 for 10%
        is_tax_inclusive BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE (property_id, name)
    );

CREATE INDEX idx_tax_rules_property ON finance.tax_rules (property_id);

-- Create ledger code table
CREATE TABLE IF NOT EXISTS
    finance.ledger_codes (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        code TEXT NOT NULL,
        description TEXT,
        tax_rule UUID REFERENCES finance.tax_rules (id) ON DELETE SET NULL,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE (property_id, code)
    );

CREATE INDEX idx_ledger_codes_property ON finance.ledger_codes (property_id);
CREATE INDEX idx_ledger_codes_tax_rule ON finance.ledger_codes (property_id, tax_rule);

CREATE TABLE IF NOT EXISTS
    sales_ledgers.accounts (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        company_profile_id UUID REFERENCES identity.company_profiles (id) ON DELETE SET NULL,
        credit_limit_pence INT DEFAULT 0, -- Stored in pence to avoid floating point issues
        payment_terms_days INT DEFAULT 0, -- Number of days for payment terms
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE (property_id, company_profile_id)
    );

CREATE INDEX idx_accounts_property ON sales_ledgers.accounts (property_id);
CREATE INDEX idx_accounts_company_profile ON sales_ledgers.accounts (company_profile_id);

-- Create group table (set master_folio_id to null for now)
CREATE TABLE IF NOT EXISTS
    operations.reservation_groups (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        master_folio_id UUID NULL, -- To be set later
        sequential SERIAL NOT NULL,
        code TEXT GENERATED ALWAYS AS (
          'GRP-' || LPAD(sequential::TEXT, 6, '0')
        ) STORED, -- e.g., GRP-000123
        name TEXT,
        notes TEXT,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE (property_id, code),
        UNIQUE (property_id, sequential)
    );

CREATE INDEX idx_reservation_groups_property ON operations.reservation_groups (property_id);

-- Create reservations table
CREATE TABLE IF NOT EXISTS
    operations.reservations (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        primary_guest_id UUID REFERENCES identity.guests (id) ON DELETE SET NULL,
        group_id UUID REFERENCES operations.reservation_groups (id) ON DELETE SET NULL,
        sequential SERIAL NOT NULL,
        code TEXT GENERATED ALWAYS AS (
          'RES-' || LPAD(sequential::TEXT, 6, '0')
        ) STORED, -- e.g., RES-000123
        source operations.reservation_source NOT NULL DEFAULT 'internal',
        travel_agent_id UUID REFERENCES identity.travel_agents (id) ON DELETE SET NULL,
        notes TEXT,
        status operations.reservation_status NOT NULL DEFAULT 'hold',
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL -- For soft deletes
    );

CREATE INDEX idx_reservations_property ON operations.reservations (property_id);
CREATE INDEX idx_reservations_primary_guest ON operations.reservations (primary_guest_id);
CREATE INDEX idx_reservations_group ON operations.reservations (group_id);
CREATE INDEX idx_reservations_travel_agent ON operations.reservations (travel_agent_id);
CREATE INDEX idx_reservations_status ON operations.reservations (status);
CREATE INDEX idx_reservations_source ON operations.reservations (source);



-- Create reservation items table
CREATE TABLE IF NOT EXISTS
    operations.reservation_items (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        reservation_id UUID REFERENCES operations.reservations (id) ON DELETE CASCADE,
        booked_room_type_id UUID REFERENCES inventory.room_types (id) ON DELETE SET NULL,
        assigned_room_id UUID REFERENCES inventory.rooms (id) ON DELETE SET NULL,
        rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE SET NULL, -- The rate plan for the majority of the stay,
        stay_period TSTZRANGE NOT NULL,
        base_rate_pence INT NOT NULL DEFAULT 0, -- The total stay base rate in pence Gets modified by booked_daily_rates for each day
        adults_count INT NOT NULL DEFAULT 2,
        children_count INT NOT NULL DEFAULT 0,
        status operations.reservation_item_status NOT NULL DEFAULT 'booked',
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL -- For soft deletes

        CONSTRAINT chk_stay_period_valid CHECK (upper(stay_period) > lower(stay_period)),
        CONSTRAINT no_overlapping_room_stays EXCLUDE USING GIST (
            assigned_room_id WITH =,
            stay_period WITH &&
        ) WHERE (deleted_at IS NULL AND assigned_room_id IS NOT NULL)
    );

CREATE INDEX idx_reservation_items_reservation ON operations.reservation_items (reservation_id);
CREATE INDEX idx_reservation_items_assigned_room ON operations.reservation_items (assigned_room_id);
CREATE INDEX idx_reservation_items_booked_room_type ON operations.reservation_items (booked_room_type_id);
CREATE INDEX idx_reservation_items_rate_plan ON operations.reservation_items (rate_plan_id);
CREATE INDEX idx_reservation_items_status ON operations.reservation_items (status);
CREATE INDEX idx_reservation_items_stay_period ON operations.reservation_items USING GIST (stay_period);

CREATE TABLE IF NOT EXISTS
    relations.reservation_item_guests (
        reservation_item_id UUID REFERENCES operations.reservation_items (id) ON DELETE CASCADE,
        guest_id UUID REFERENCES identity.guests (id) ON DELETE CASCADE,
        role operations.reservation_guest_role NOT NULL DEFAULT 'additional',
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        PRIMARY KEY (reservation_item_id, guest_id)
    );

CREATE INDEX idx_reservation_item_guests_guest ON relations.reservation_item_guests (guest_id);
CREATE INDEX idx_reservation_item_guests_reservation ON relations.reservation_item_guests (reservation_item_id);
CREATE INDEX idx_reservation_item_guests_role ON relations.reservation_item_guests (role);

-- Create booked daily rates table
CREATE TABLE IF NOT EXISTS
    pricing.booked_daily_rates (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        reservation_item_id UUID REFERENCES operations.reservation_items (id) ON DELETE CASCADE,
        calendar_date DATE NOT NULL,
        rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE SET NULL,
        base_price_pence INT NOT NULL DEFAULT 0, -- Daily price in pence
        adjustment JSONB CHECK (
            adjustment IS NULL
            OR (
                adjustment?'type'
                AND adjustment?'value'
                AND adjustment?'reason'
                AND adjustment->>'type' IN ('percentage', 'fixed')
                AND (adjustment->>'value')::INT<>0
                AND adjustment->>'reason' IS NOT NULL
            )
        ), -- e.g., {"type": "percentage", "value": 10} for 10% increase
        adjustment_approved BOOLEAN DEFAULT FALSE,
        adjustment_approved_by_user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL,
        final_price_pence INT NOT NULL GENERATED ALWAYS AS (
            CASE
                WHEN adjustment IS NULL THEN base_price_pence
                WHEN adjustment->>'type'='percentage' THEN base_price_pence+(
                    (base_price_pence*(adjustment->>'value')::NUMERIC)/100
                )::INT
                WHEN adjustment->>'type'='fixed' THEN base_price_pence+(adjustment->>'value')::INT
                ELSE base_price_pence
            END
        ) STORED,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE (reservation_item_id, calendar_date)
    );

CREATE INDEX idx_booked_daily_rates_calendar_date ON pricing.booked_daily_rates (calendar_date);
CREATE INDEX idx_booked_daily_rates_reservation_item ON pricing.booked_daily_rates (reservation_item_id);
CREATE INDEX idx_booked_daily_rates_rate_plan ON pricing.booked_daily_rates (rate_plan_id);
CREATE INDEX idx_booked_daily_rates_adjustment_approved ON pricing.booked_daily_rates (adjustment_approved);
CREATE INDEX idx_booked_daily_rates_adjustment_approved_by ON pricing.booked_daily_rates (adjustment_approved_by_user_id);
CREATE INDEX idx_booked_daily_rates_rate_plan_date ON pricing.booked_daily_rates (rate_plan_id, calendar_date);

-- Create folios table (add master_folio_id to groups)
CREATE TABLE IF NOT EXISTS
    finance.folios (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        reservation_id UUID REFERENCES operations.reservations (id) ON DELETE SET NULL,
        sales_ledger_id UUID REFERENCES sales_ledgers.accounts (id) ON DELETE SET NULL,
        folio_part finance.folio_part NOT NULL,
        balance_pence INT NOT NULL DEFAULT 0,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL -- For soft deletes
    );

CREATE INDEX idx_folios_property ON finance.folios (property_id);
CREATE INDEX idx_folios_reservation ON finance.folios (reservation_id);
CREATE INDEX idx_folios_sales_ledger ON finance.folios (sales_ledger_id);

-- Add master_folio_id to reservation groups
ALTER TABLE operations.reservation_groups
ADD COLUMN IF NOT EXISTS master_folio_id UUID REFERENCES finance.folios (id) ON DELETE SET NULL;


CREATE INDEX idx_reservation_groups_master_folio ON operations.reservation_groups (master_folio_id);

-- Create folio transcations table
CREATE TABLE IF NOT EXISTS
    finance.folio_transactions (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        folio_id UUID REFERENCES finance.folios (id) ON DELETE CASCADE,
        ledger_code_id UUID REFERENCES finance.ledger_codes (id) ON DELETE SET NULL,
        description TEXT,
        net_unit_price_pence INT NOT NULL DEFAULT 0, -- Can be positive (charge) or negative (credit)
        quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
        tax_rule_id UUID REFERENCES finance.tax_rules (id) ON DELETE SET NULL,
        total_net_price_pence INT NOT NULL GENERATED ALWAYS AS (net_unit_price_pence*quantity) STORED,
        tax_rate_snapshot NUMERIC(5, 2) NOT NULL, -- Snapshot of tax rate at time of transaction (from tax_rules table)
        tax_amount_pence INT GENERATED ALWAYS AS (CAST((net_unit_price_pence*quantity) * tax_rate_snapshot / 100 AS INT)) STORED,
        gross_amount_pence INT NOT NULL GENERATED ALWAYS AS (net_unit_price_pence * quantity + (CAST((net_unit_price_pence*quantity) * tax_rate_snapshot / 100 AS INT))) STORED,
        posted_at TIMESTAMPTZ DEFAULT NOW(), -- A tx can be created, but only posted when finalised
        posted_by_user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL CHECK(posted_at IS NOT NULL OR posted_by_user_id IS NULL),
        status finance.folio_transaction_status NOT NULL DEFAULT 'pending',
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL -- For soft deletes
    );

CREATE INDEX idx_folio_transactions_folio ON finance.folio_transactions (folio_id);
CREATE INDEX idx_folio_transactions_ledger_code ON finance.folio_transactions (ledger_code_id);
CREATE INDEX idx_folio_transactions_tax_rule ON finance.folio_transactions (tax_rule_id);
CREATE INDEX idx_folio_transactions_posted_by ON finance.folio_transactions (posted_by_user_id);
CREATE INDEX idx_folio_transactions_status ON finance.folio_transactions (status);


-- Create invoices table
CREATE TABLE IF NOT EXISTS
    finance.invoices (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        folio_id UUID REFERENCES finance.folios (id) ON DELETE SET NULL, -- Nullable for pro-forma invoices or sales ledger only invoices
        property_code TEXT NOT NULL DEFAULT 'RES' CHECK(LENGTH(property_code) = 3 OR LENGTH(property_code) = 4), -- Snapshot of property code at time of invoice
        fiscal_year INT NOT NULL,
        fiscal_sequential INT NOT NULL, -- Sequential number within the fiscal year
        invoice_number TEXT GENERATED ALWAYS AS (
            property_code||'-'||fiscal_year||'-'||LPAD(fiscal_sequential::TEXT, 6, '0')
        ) STORED, -- e.g., PROP-2024-000123
        billing_address TEXT NOT NULL,
        is_pro_forma BOOLEAN DEFAULT FALSE,
        issue_date TIMESTAMPTZ DEFAULT NOW(),
        due_date TIMESTAMPTZ DEFAULT NOW()+ INTERVAL '30 days' CHECK(due_date >= issue_date),
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE (property_id, fiscal_year, fiscal_sequential)
    );

CREATE INDEX idx_invoices_folio ON finance.invoices (folio_id);
CREATE INDEX idx_invoices_property ON finance.invoices (property_id);
CREATE INDEX idx_invoices_property_fiscal_year ON finance.invoices (property_id, fiscal_year);
CREATE INDEX idx_invoices_property_issue_date ON finance.invoices (property_id, issue_date);
CREATE INDEX idx_invoices_property_due_date ON finance.invoices (property_id, due_date);


-- Create sales ledger transactions table
CREATE TABLE IF NOT EXISTS sales_ledgers.transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ledger_account_id UUID REFERENCES sales_ledgers.accounts(id) ON DELETE CASCADE,
    source_invoice_id UUID REFERENCES finance.invoices(id) ON DELETE SET NULL,
    amount_pence INT NOT NULL, -- Positive for charges, negative for payments
    due_date TIMESTAMPTZ DEFAULT NOW() + INTERVAL '30 days' CHECK(due_date >= NOW()),
    is_fully_paid BOOLEAN GENERATED ALWAYS AS (amount_pence <= 0) STORED,
    posted_at TIMESTAMPTZ DEFAULT NOW(),
    posted_by_user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    type sales_ledgers.transaction_type NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL -- For soft deletes
);

CREATE INDEX idx_sales_ledger_transactions_account ON sales_ledgers.transactions (ledger_account_id);
CREATE INDEX idx_sales_ledger_transactions_invoice ON sales_ledgers.transactions (source_invoice_id);
CREATE INDEX idx_sales_ledger_transactions_posted_by ON sales_ledgers.transactions (posted_by_user_id);
CREATE INDEX idx_sales_ledger_transactions_type ON sales_ledgers.transactions (type);
CREATE INDEX idx_sales_ledger_transactions_due_date ON sales_ledgers.transactions (due_date);
CREATE INDEX idx_sales_ledger_transactions_posted_at ON sales_ledgers.transactions (posted_at);
CREATE INDEX idx_sales_ledger_transactions_fully_paid ON sales_ledgers.transactions (is_fully_paid);


-- Create checkout session  
CREATE TABLE IF NOT EXISTS operations.checkout_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID REFERENCES operations.properties(id) ON DELETE CASCADE,
    reservation_id UUID REFERENCES operations.reservations(id) ON DELETE CASCADE,
    payment_intent_id TEXT UNIQUE NOT NULL, -- TODO: Link to payment gateway intent
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '15 minutes',
    status operations.checkout_session_status NOT NULL DEFAULT 'pending',
    idempotency_key TEXT UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
    UNIQUE(property_id, reservation_id)
);

CREATE INDEX idx_checkout_sessions_property ON operations.checkout_sessions (property_id);
CREATE INDEX idx_checkout_sessions_reservation ON operations.checkout_sessions (reservation_id);
CREATE INDEX idx_checkout_sessions_expires_at ON operations.checkout_sessions (expires_at);
CREATE INDEX idx_checkout_sessions_status ON operations.checkout_sessions (status);
CREATE INDEX idx_checkout_sessions_payment_intent ON operations.checkout_sessions (payment_intent_id);

-- Create room inventory ledger table
CREATE TABLE IF NOT EXISTS inventory.room_inventory_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID REFERENCES inventory.rooms(id) ON DELETE CASCADE,
    reservation_id UUID REFERENCES operations.reservations(id) ON DELETE SET NULL, -- Nullable for non-reserved inventory changes
    checkout_session_id UUID REFERENCES operations.checkout_sessions(id) ON DELETE SET NULL, -- Nullable for non-checkout related changes
    calendar_date DATE NOT NULL,
    status inventory.inventory_status NOT NULL DEFAULT 'on_hold', -- e.g., 'available', 'sold', 'decommissioned'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
    UNIQUE (room_id, calendar_date),

    CONSTRAINT sold_requires_reservation CHECK (
        (status = 'sold' AND reservation_id IS NOT NULL)
        OR (status <> 'sold')
    ),

    CONSTRAINT on_hold_requires_checkout_session CHECK (
        (status = 'on_hold' AND checkout_session_id IS NOT NULL)
        OR (status <> 'on_hold')
    )
);

CREATE INDEX idx_inventory_dates_n_rooms_available ON inventory.room_inventory_ledger (calendar_date, room_id) WHERE status = 'available';
CREATE INDEX idx_availability_date ON inventory.room_inventory_ledger (calendar_date) WHERE status = 'available';
CREATE INDEX idx_room_inventory_ledger_room_date ON inventory.room_inventory_ledger (room_id, calendar_date);
CREATE INDEX idx_room_inventory_ledger_calendar_date ON inventory.room_inventory_ledger (calendar_date);
CREATE INDEX idx_room_inventory_ledger_status ON inventory.room_inventory_ledger (status);
CREATE INDEX idx_room_inventory_ledger_reservation ON inventory.room_inventory_ledger (reservation_id);
CREATE INDEX idx_room_inventory_ledger_checkout_session ON inventory.room_inventory_ledger (checkout_session_id);
CREATE INDEX idx_room_inventory_ledger_room ON inventory.room_inventory_ledger (room_id);



-- Create housekeeping logs table
CREATE TABLE IF NOT EXISTS inventory.housekeeping_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID REFERENCES operations.properties(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    room_id UUID REFERENCES inventory.rooms(id) ON DELETE CASCADE,
    status_to inventory.housekeeping_status NOT NULL,
    status_from inventory.housekeeping_status NOT NULL CHECK(status_to <> status_from),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL -- For soft deletes)
);

CREATE INDEX idx_housekeeping_logs_property ON inventory.housekeeping_logs (property_id);
CREATE INDEX idx_housekeeping_logs_user ON inventory.housekeeping_logs (user_id);
CREATE INDEX idx_housekeeping_logs_room ON inventory.housekeeping_logs (room_id);
-- DEPRECATED - Create rate adjustments table 
-- JSONB used in booked_daily_rates now

-- +goose Down
-- 1. Drop tables that depend on others (Leaf nodes first)
DROP TABLE IF EXISTS inventory.housekeeping_logs;
DROP TABLE IF EXISTS inventory.room_inventory_ledger;
DROP TABLE IF EXISTS operations.checkout_sessions;
DROP TABLE IF EXISTS sales_ledgers.transactions;
DROP TABLE IF EXISTS finance.invoices;
DROP TABLE IF EXISTS finance.folio_transactions;

-- 2. Break circular/complex dependencies
-- Remove the FK from reservation_groups to folios so we can drop folios safely
ALTER TABLE operations.reservation_groups DROP COLUMN IF EXISTS master_folio_id;

-- 3. Drop Finance & Pricing tables
DROP TABLE IF EXISTS finance.folios;
DROP TABLE IF EXISTS pricing.booked_daily_rates; -- <--- WAS MISSING (Caused your error)

-- 4. Drop Core Operation tables
DROP TABLE IF EXISTS operations.reservation_items;
DROP TABLE IF EXISTS relations.reservation_item_guests; -- <--- WAS MISSING
DROP TABLE IF EXISTS operations.reservations;
DROP TABLE IF EXISTS operations.reservation_groups; -- <--- WAS MISSING (You only dropped the column before)

-- 5. Drop Setup/Config tables
DROP TABLE IF EXISTS sales_ledgers.accounts;
DROP TABLE IF EXISTS finance.ledger_codes;
DROP TABLE IF EXISTS finance.tax_rules;