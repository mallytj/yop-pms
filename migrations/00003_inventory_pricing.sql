-- +goose Up

-- ========================================================
-- 1. ROOM TYPES
-- ========================================================
CREATE TABLE inventory.room_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 75),
    code CITEXT NOT NULL CHECK (char_length(code) <= 10),
    std_occupancy INT NOT NULL DEFAULT 2 CHECK (std_occupancy > 0),
    min_occupancy INT NOT NULL DEFAULT 1,
    max_occupancy INT NOT NULL DEFAULT 2,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (min_occupancy > 0 AND min_occupancy <= std_occupancy AND max_occupancy >= std_occupancy),
    UNIQUE (property_id, id)
);

ALTER TABLE inventory.room_types ENABLE ROW LEVEL SECURITY;
CREATE POLICY room_types_isolation ON inventory.room_types USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE UNIQUE INDEX idx_room_types_code_act ON inventory.room_types (property_id, code) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX idx_room_types_name_act ON inventory.room_types (property_id, name) WHERE (deleted_at IS NULL);

-- ========================================================
-- 2. ROOMS
-- ========================================================
CREATE TABLE inventory.rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_type_id UUID REFERENCES inventory.room_types (id) ON DELETE SET NULL,
    name TEXT NOT NULL CHECK (char_length(name) <= 75),
    housekeeping_status inventory.housekeeping_status NOT NULL DEFAULT 'clean',
    occupancy_status inventory.occupancy_status NOT NULL DEFAULT 'vacant',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id),
    UNIQUE (property_id, id)
);

ALTER TABLE inventory.rooms ENABLE ROW LEVEL SECURITY;
CREATE POLICY rooms_isolation ON inventory.rooms USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE UNIQUE INDEX idx_rooms_name_act ON inventory.rooms (property_id, name) WHERE (deleted_at IS NULL);

-- ========================================================
-- 3. AMENITY RELATIONS
-- ========================================================
CREATE TABLE relations.room_type_amenities (
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_type_id UUID NOT NULL REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
    amenity_id UUID NOT NULL REFERENCES operations.amenities (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_type_id, amenity_id),
    FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id),
    FOREIGN KEY (property_id, amenity_id) REFERENCES operations.amenities (property_id, id)
);

ALTER TABLE relations.room_type_amenities ENABLE ROW LEVEL SECURITY;
CREATE POLICY rt_amenities_isolation ON relations.room_type_amenities USING (property_id = current_setting('app.current_property_id')::uuid);

-- ========================================================
-- 4. MAINTENANCE BLOCKS
-- ========================================================
CREATE TABLE inventory.maintenance_blocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_id UUID NOT NULL REFERENCES inventory.rooms (id) ON DELETE RESTRICT,
    block_period TSTZRANGE NOT NULL CHECK (upper(block_period) > lower(block_period)),
    reason TEXT NOT NULL CHECK (char_length(reason) <= 150),
    type inventory.maintenance_block_type NOT NULL,
    created_by_user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, room_id) REFERENCES inventory.rooms (property_id, id),
    -- Prevent overlapping blocks for same room
    EXCLUDE USING GIST (room_id WITH =, block_period WITH &&) WHERE (deleted_at IS NULL)
);

ALTER TABLE inventory.maintenance_blocks ENABLE ROW LEVEL SECURITY;
CREATE POLICY maint_isolation ON inventory.maintenance_blocks USING (property_id = current_setting('app.current_property_id')::uuid);

-- ========================================================
-- 5. RATE PLANS & PRICING
-- ========================================================
CREATE TABLE pricing.rate_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 50),
    code CITEXT NOT NULL CHECK (char_length(code) <= 10),
    description TEXT CHECK (char_length(description) <= 300),
    parent_rate_plan_id UUID,
    derivation_rule JSONB CHECK (
        derivation_rule IS NULL OR (
            derivation_rule ? 'type' AND derivation_rule ? 'value' AND derivation_rule ->> 'type' IN ('percentage', 'fixed')
        )
    ),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    currency_code CITEXT NOT NULL DEFAULT 'GBP' CHECK (char_length(currency_code) = 3),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, parent_rate_plan_id) REFERENCES pricing.rate_plans (property_id, id),
    CHECK ((parent_rate_plan_id IS NULL AND derivation_rule IS NULL) OR (parent_rate_plan_id IS NOT NULL AND derivation_rule IS NOT NULL)),
    UNIQUE (property_id, id)
);

ALTER TABLE pricing.rate_plans ENABLE ROW LEVEL SECURITY;
CREATE POLICY rate_plans_isolation ON pricing.rate_plans USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE UNIQUE INDEX idx_rate_plans_code_act ON pricing.rate_plans (property_id, code) WHERE (deleted_at IS NULL);

-- ========================================================
-- 6. DAILY PRICE GRID
-- ========================================================
CREATE TABLE pricing.daily_price_grid (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_type_id UUID NOT NULL REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
    rate_plan_id UUID NOT NULL REFERENCES pricing.rate_plans (id) ON DELETE RESTRICT,
    calendar_date DATE NOT NULL,
    base_price_pence INTEGER NOT NULL DEFAULT 0,
    min_los_restriction INT NOT NULL DEFAULT 1 CHECK (min_los_restriction > 0),
    max_los_restriction INT NOT NULL DEFAULT 365,
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id),
    FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id),
    CHECK (max_los_restriction >= min_los_restriction)
);

ALTER TABLE pricing.daily_price_grid ENABLE ROW LEVEL SECURITY;
CREATE POLICY prices_isolation ON pricing.daily_price_grid USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE UNIQUE INDEX idx_price_lookup ON pricing.daily_price_grid (property_id, room_type_id, rate_plan_id, calendar_date) WHERE (deleted_at IS NULL);
-- ========================================================
-- 2. PRICING OVERRIDES (Base & Seasonal)
-- ========================================================

CREATE TABLE pricing.base_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_type_id UUID NOT NULL REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
    rate_plan_id UUID NOT NULL REFERENCES pricing.rate_plans (id) ON DELETE RESTRICT,
    day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    base_price_pence INTEGER NOT NULL DEFAULT 0,
    min_los_restriction INT CHECK (min_los_restriction > 0),
    max_los_restriction INT CHECK (max_los_restriction > min_los_restriction),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    UNIQUE (property_id, id),
    UNIQUE (property_id, room_type_id, rate_plan_id, day_of_week),
    FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id),
    FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id)
);

ALTER TABLE pricing.base_rates ENABLE ROW LEVEL SECURITY;
CREATE POLICY base_rates_isolation ON pricing.base_rates USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE TABLE pricing.seasonal_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_type_id UUID NOT NULL REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
    rate_plan_id UUID NOT NULL REFERENCES pricing.rate_plans (id) ON DELETE RESTRICT,
    override_period TSTZRANGE NOT NULL CHECK (lower(override_period) < upper(override_period)),
    day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    base_price_pence INTEGER NOT NULL DEFAULT 0,
    min_los_restriction INT CHECK (min_los_restriction > 0),
    max_los_restriction INT CHECK (max_los_restriction > min_los_restriction),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    UNIQUE (property_id, id),
    FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id),
    FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id),
    -- REQ-024: Prevent overlapping seasonal overrides for the same type/plan/day
    EXCLUDE USING GIST (
        property_id WITH =, 
        room_type_id WITH =, 
        rate_plan_id WITH =, 
        day_of_week WITH =, 
        override_period WITH &&
    ) WHERE (deleted_at IS NULL)
);

ALTER TABLE pricing.seasonal_rates ENABLE ROW LEVEL SECURITY;
CREATE POLICY seasonal_rates_isolation ON pricing.seasonal_rates USING (property_id = current_setting('app.current_property_id')::uuid);
-- ========================================================
-- 7. COMPANY PROFILES (identity schema)
-- ========================================================
CREATE TABLE identity.company_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    negotiated_rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE SET NULL,
    tax_id CITEXT, -- e.g., VAT number
    company_name TEXT NOT NULL CHECK (char_length(company_name) BETWEEN 2 AND 50),
    contact_email CITEXT,
    contact_phone TEXT,
    billing_address TEXT CHECK (char_length(billing_address) <= 300),
    company_notes TEXT CHECK (char_length(company_notes) <= 1500),
    has_credit_facility BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- REQUIRED for composite FK in 004_sales_ledgers
    UNIQUE (property_id, id),
    -- Ensure negotiated rate belongs to the same property
    FOREIGN KEY (property_id, negotiated_rate_plan_id) REFERENCES pricing.rate_plans (property_id, id)
);

ALTER TABLE identity.company_profiles ENABLE ROW LEVEL SECURITY;
CREATE POLICY company_profiles_isolation ON identity.company_profiles 
    USING (property_id = current_setting('app.current_property_id')::uuid);

-- Unique per property, ignoring soft-deleted companies
CREATE UNIQUE INDEX idx_company_name_active ON identity.company_profiles (property_id, company_name) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX idx_company_tax_active ON identity.company_profiles (property_id, tax_id) WHERE (deleted_at IS NULL AND tax_id IS NOT NULL);

-- +goose Down
DROP TABLE IF EXISTS identity.company_profiles;
DROP TABLE IF EXISTS pricing.seasonal_rates;
DROP TABLE IF EXISTS pricing.base_rates;
DROP TABLE IF EXISTS pricing.daily_price_grid;
DROP TABLE IF EXISTS pricing.rate_plans;
DROP TABLE IF EXISTS inventory.maintenance_blocks;
DROP TABLE IF EXISTS relations.room_amenities;
DROP TABLE IF EXISTS relations.room_type_amenities;
DROP TABLE IF EXISTS inventory.rooms;
DROP TABLE IF EXISTS inventory.room_types;