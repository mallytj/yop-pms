-- +goose Up

-- ========================================================
-- 1. FINANCE CONFIG
-- ========================================================

CREATE TABLE finance.tax_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 50),
    description TEXT CHECK (char_length(description) <= 250),
    tax_percentage NUMERIC(5, 2) NOT NULL CHECK (tax_percentage BETWEEN 0 AND 75),
    is_tax_inclusive BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (property_id, id)
);

ALTER TABLE finance.tax_rules ENABLE ROW LEVEL SECURITY;
CREATE POLICY tax_rules_isolation ON finance.tax_rules USING (property_id = current_setting('app.current_property_id')::uuid);
CREATE UNIQUE INDEX idx_tax_rules_name_act ON finance.tax_rules (property_id, name) WHERE (deleted_at IS NULL);

CREATE TABLE finance.ledger_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    code CITEXT NOT NULL CHECK (char_length(code) <= 50),
    description TEXT CHECK (char_length(description) <= 250),
    tax_rule_id UUID REFERENCES finance.tax_rules (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, tax_rule_id) REFERENCES finance.tax_rules (property_id, id),
    UNIQUE (property_id, id)
);

ALTER TABLE finance.ledger_codes ENABLE ROW LEVEL SECURITY;
CREATE POLICY ledger_codes_isolation ON finance.ledger_codes USING (property_id = current_setting('app.current_property_id')::uuid);
CREATE UNIQUE INDEX idx_ledger_codes_code_act ON finance.ledger_codes (property_id, code) WHERE (deleted_at IS NULL);

-- ========================================================
-- 2. SALES LEDGER ACCOUNTS
-- ========================================================

CREATE TABLE sales_ledgers.accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    company_profile_id UUID REFERENCES identity.company_profiles (id) ON DELETE SET NULL,
    name TEXT NOT NULL CHECK (char_length(name) <= 100),
    code CITEXT NOT NULL CHECK (char_length(code) <= 10),
    credit_limit_pence INT NOT NULL DEFAULT 0 CHECK (credit_limit_pence >= 0),
    payment_terms_days INT NOT NULL DEFAULT 0 CHECK (payment_terms_days >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, company_profile_id) REFERENCES identity.company_profiles (property_id, id),
    UNIQUE (property_id, id)
);

ALTER TABLE sales_ledgers.accounts ENABLE ROW LEVEL SECURITY;
CREATE POLICY accounts_isolation ON sales_ledgers.accounts USING (property_id = current_setting('app.current_property_id')::uuid);
CREATE UNIQUE INDEX idx_accounts_code_act ON sales_ledgers.accounts (property_id, code) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX idx_accounts_company_act ON sales_ledgers.accounts (property_id, company_profile_id) WHERE (deleted_at IS NULL);

-- ========================================================
-- 3. RESERVATIONS & GROUPS
-- ========================================================

CREATE TABLE operations.reservation_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    master_folio_id UUID, -- Circular ref; FK added after folios table
    sequential SERIAL NOT NULL,
    code TEXT GENERATED ALWAYS AS ('GRP-' || LPAD(sequential::TEXT, 5, '0')) STORED,
    name TEXT CHECK (char_length(name) <= 50),
    notes TEXT CHECK (char_length(notes) <= 2500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(property_id, id)
);

ALTER TABLE operations.reservation_groups ENABLE ROW LEVEL SECURITY;
CREATE POLICY res_groups_isolation ON operations.reservation_groups USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE TABLE operations.reservations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    primary_guest_id UUID NOT NULL REFERENCES identity.guests (id) ON DELETE RESTRICT,
    group_id UUID REFERENCES operations.reservation_groups (id) ON DELETE SET NULL,
    sequential SERIAL NOT NULL,
    code TEXT GENERATED ALWAYS AS ('RES-' || LPAD(sequential::TEXT, 6, '0')) STORED,
    source operations.reservation_source NOT NULL DEFAULT 'internal',
    travel_agent_id UUID REFERENCES identity.travel_agents (id) ON DELETE SET NULL,
    notes TEXT CHECK (char_length(notes) <= 2500),
    status operations.reservation_status NOT NULL DEFAULT 'hold',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, primary_guest_id) REFERENCES identity.guests (property_id, id),
    FOREIGN KEY (property_id, group_id) REFERENCES operations.reservation_groups (property_id, id),
    FOREIGN KEY (property_id, travel_agent_id) REFERENCES identity.travel_agents (property_id, id),
    UNIQUE (property_id, id)
);

ALTER TABLE operations.reservations ENABLE ROW LEVEL SECURITY;
CREATE POLICY reservations_isolation ON operations.reservations USING (property_id = current_setting('app.current_property_id')::uuid);

-- ========================================================
-- 4. RESERVATION ITEMS
-- ========================================================

CREATE TABLE operations.reservation_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    reservation_id UUID NOT NULL REFERENCES operations.reservations (id) ON DELETE RESTRICT,
    booked_room_type_id UUID NOT NULL REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
    assigned_room_id UUID REFERENCES inventory.rooms (id) ON DELETE SET NULL,
    guest_id UUID REFERENCES identity.guests (id) ON DELETE SET NULL,
    rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE SET NULL,
    stay_period TSTZRANGE NOT NULL CHECK (
        lower(stay_period) < upper(stay_period)
        AND lower(stay_period) IS NOT NULL
        AND upper(stay_period) IS NOT NULL
    ),
    base_rate_pence INT NOT NULL DEFAULT 0 CHECK (base_rate_pence >= 0),
    adults_count INT NOT NULL DEFAULT 2 CHECK (adults_count >= 1),
    children_count INT NOT NULL DEFAULT 0,
    status operations.reservation_item_status NOT NULL DEFAULT 'booked',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    -- REQ-024: DB level overlap prevention
    EXCLUDE USING GIST (assigned_room_id WITH =, stay_period WITH &&) WHERE (deleted_at IS NULL AND assigned_room_id IS NOT NULL),
    FOREIGN KEY (property_id, reservation_id) REFERENCES operations.reservations (property_id, id),
    FOREIGN KEY (property_id, booked_room_type_id) REFERENCES inventory.room_types (property_id, id),
    FOREIGN KEY (property_id, assigned_room_id) REFERENCES inventory.rooms (property_id, id),
    FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id),
    UNIQUE (property_id, id)
);

ALTER TABLE operations.reservation_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY res_items_isolation ON operations.reservation_items USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE TABLE relations.reservation_item_guests (
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    reservation_item_id UUID NOT NULL REFERENCES operations.reservation_items (id) ON DELETE RESTRICT,
    guest_id UUID NOT NULL REFERENCES identity.guests (id) ON DELETE RESTRICT,
    role operations.reservation_guest_role NOT NULL DEFAULT 'additional',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (reservation_item_id, guest_id),
    FOREIGN KEY (property_id, reservation_item_id) REFERENCES operations.reservation_items (property_id, id),
    FOREIGN KEY (property_id, guest_id) REFERENCES identity.guests (property_id, id)
);

ALTER TABLE relations.reservation_item_guests ENABLE ROW LEVEL SECURITY;
CREATE POLICY res_item_guests_isolation ON relations.reservation_item_guests USING (property_id = current_setting('app.current_property_id')::uuid);

-- ========================================================
-- 5. PRICING & FOLIOS
-- ========================================================

CREATE TABLE pricing.booked_daily_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    reservation_item_id UUID NOT NULL REFERENCES operations.reservation_items (id) ON DELETE RESTRICT,
    calendar_date DATE NOT NULL,
    rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE SET NULL,
    base_price_pence INT NOT NULL DEFAULT 0 CHECK (base_price_pence >= 0),
    adjustment JSONB CHECK (adjustment IS NULL OR (adjustment ? 'type' AND adjustment ? 'value' AND adjustment ? 'reason')),
    adjustment_approved BOOLEAN NOT NULL DEFAULT FALSE,
    adjustment_approved_by_user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL,
    final_price_pence INT NOT NULL, -- Calculated via trigger as per REQ-018 exception or Go
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (reservation_item_id, calendar_date),
    FOREIGN KEY (property_id, reservation_item_id) REFERENCES operations.reservation_items (property_id, id),
    FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id)
);

ALTER TABLE pricing.booked_daily_rates ENABLE ROW LEVEL SECURITY;
CREATE POLICY daily_rates_isolation ON pricing.booked_daily_rates USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE TABLE finance.folios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    reservation_id UUID REFERENCES operations.reservations (id) ON DELETE SET NULL,
    sales_ledger_id UUID REFERENCES sales_ledgers.accounts (id) ON DELETE SET NULL,
    folio_part finance.folio_part NOT NULL,
    balance_pence INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, reservation_id) REFERENCES operations.reservations (property_id, id),
    FOREIGN KEY (property_id, sales_ledger_id) REFERENCES sales_ledgers.accounts (property_id, id),
    UNIQUE (property_id, id)
);

ALTER TABLE finance.folios ENABLE ROW LEVEL SECURITY;
CREATE POLICY folios_isolation ON finance.folios USING (property_id = current_setting('app.current_property_id')::uuid);

-- Resolve circular dependency for Groups
ALTER TABLE operations.reservation_groups ADD CONSTRAINT fk_res_groups_folio FOREIGN KEY (property_id, master_folio_id) REFERENCES finance.folios (property_id, id);

CREATE TABLE finance.folio_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    folio_id UUID NOT NULL REFERENCES finance.folios (id) ON DELETE RESTRICT,
    ledger_code_id UUID REFERENCES finance.ledger_codes (id) ON DELETE SET NULL,
    description TEXT,
    net_unit_price_pence INT NOT NULL DEFAULT 0,
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    tax_rule_id UUID REFERENCES finance.tax_rules (id) ON DELETE SET NULL,
    tax_rate_snapshot NUMERIC(5, 2) NOT NULL,
    status finance.folio_transaction_status NOT NULL DEFAULT 'pending',
    posted_at TIMESTAMPTZ,
    posted_by_user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, folio_id) REFERENCES finance.folios (property_id, id),
    FOREIGN KEY (property_id, ledger_code_id) REFERENCES finance.ledger_codes (property_id, id),
    FOREIGN KEY (property_id, tax_rule_id) REFERENCES finance.tax_rules (property_id, id)
);

ALTER TABLE finance.folio_transactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY folio_tx_isolation ON finance.folio_transactions USING (property_id = current_setting('app.current_property_id')::uuid);

-- ========================================================
-- 6. INVOICING & CHECKOUT
-- ========================================================

CREATE TABLE finance.invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    folio_id UUID REFERENCES finance.folios (id) ON DELETE SET NULL,
    property_code CITEXT NOT NULL DEFAULT 'RES' CHECK (char_length(property_code) BETWEEN 3 AND 4),
    fiscal_year INT NOT NULL,
    fiscal_sequential INT NOT NULL,
    invoice_number TEXT GENERATED ALWAYS AS (property_code || '-' || fiscal_year || '-' || LPAD(fiscal_sequential::TEXT, 6, '0')) STORED,
    billing_address TEXT NOT NULL,
    is_pro_forma BOOLEAN NOT NULL DEFAULT FALSE,
    issue_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_date TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, folio_id) REFERENCES finance.folios (property_id, id),
    UNIQUE (property_id, fiscal_year, fiscal_sequential),
    UNIQUE (property_id, id)
);

ALTER TABLE finance.invoices ENABLE ROW LEVEL SECURITY;
CREATE POLICY invoices_isolation ON finance.invoices USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE TABLE sales_ledgers.transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    ledger_account_id UUID NOT NULL REFERENCES sales_ledgers.accounts (id) ON DELETE RESTRICT,
    source_invoice_id UUID REFERENCES finance.invoices (id) ON DELETE SET NULL,
    amount_pence INT NOT NULL,
    due_date TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    posted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    posted_by_user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL,
    type sales_ledgers.transaction_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, ledger_account_id) REFERENCES sales_ledgers.accounts (property_id, id),
    FOREIGN KEY (property_id, source_invoice_id) REFERENCES finance.invoices (property_id, id)
);

ALTER TABLE sales_ledgers.transactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY sl_tx_isolation ON sales_ledgers.transactions USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE TABLE operations.checkout_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    reservation_id UUID NOT NULL REFERENCES operations.reservations (id) ON DELETE RESTRICT,
    payment_intent_id TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '15 minutes'),
    status operations.checkout_session_status NOT NULL DEFAULT 'pending',
    idempotency_key TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, reservation_id) REFERENCES operations.reservations (property_id, id),
    UNIQUE (property_id, id)
);

ALTER TABLE operations.checkout_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY checkout_isolation ON operations.checkout_sessions USING (property_id = current_setting('app.current_property_id')::uuid);

-- ========================================================
-- 7. INVENTORY LEDGER & LOGS
-- ========================================================

CREATE TABLE inventory.room_inventory_ledger (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_id UUID NOT NULL REFERENCES inventory.rooms (id) ON DELETE RESTRICT,
    reservation_id UUID REFERENCES operations.reservations (id) ON DELETE SET NULL,
    checkout_session_id UUID REFERENCES operations.checkout_sessions (id) ON DELETE SET NULL,
    calendar_date DATE NOT NULL,
    status inventory.inventory_status NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (room_id, calendar_date),
    FOREIGN KEY (property_id, room_id) REFERENCES inventory.rooms (property_id, id),
    FOREIGN KEY (property_id, reservation_id) REFERENCES operations.reservations (property_id, id),
    FOREIGN KEY (property_id, checkout_session_id) REFERENCES operations.checkout_sessions (property_id, id),
    CHECK ((status = 'sold' AND reservation_id IS NOT NULL) OR (status <> 'sold')),
    CHECK ((status = 'on_hold' AND checkout_session_id IS NOT NULL) OR (status <> 'on_hold'))
);

ALTER TABLE inventory.room_inventory_ledger ENABLE ROW LEVEL SECURITY;
CREATE POLICY inv_ledger_isolation ON inventory.room_inventory_ledger USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE TABLE inventory.housekeeping_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL,
    room_id UUID NOT NULL REFERENCES inventory.rooms (id) ON DELETE RESTRICT,
    status_to inventory.housekeeping_status NOT NULL,
    status_from inventory.housekeeping_status NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, room_id) REFERENCES inventory.rooms (property_id, id),
    CHECK (status_to <> status_from)
);

ALTER TABLE inventory.housekeeping_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY housekeeping_isolation ON inventory.housekeeping_logs USING (property_id = current_setting('app.current_property_id')::uuid);

-- +goose Down
DROP TABLE IF EXISTS inventory.housekeeping_logs;
DROP TABLE IF EXISTS inventory.room_inventory_ledger;
DROP TABLE IF EXISTS operations.checkout_sessions;
DROP TABLE IF EXISTS sales_ledgers.transactions;
DROP TABLE IF EXISTS finance.folio_transactions;
DROP TABLE IF EXISTS pricing.booked_daily_rates;
DROP TABLE IF EXISTS relations.reservation_item_guests;
DROP TABLE IF EXISTS finance.invoices;
ALTER TABLE IF EXISTS operations.reservation_groups DROP CONSTRAINT IF EXISTS fk_res_groups_folio;
DROP TABLE IF EXISTS finance.folios;
DROP TABLE IF EXISTS operations.reservation_items;
DROP TABLE IF EXISTS operations.reservations;
DROP TABLE IF EXISTS operations.reservation_groups;
DROP TABLE IF EXISTS sales_ledgers.accounts;
DROP TABLE IF EXISTS finance.ledger_codes;
DROP TABLE IF EXISTS finance.tax_rules;