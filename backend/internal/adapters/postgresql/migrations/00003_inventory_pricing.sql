-- +goose Up
-- Room types
CREATE TABLE IF NOT EXISTS
    inventory.room_types (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
        name TEXT NOT NULL CHECK (char_length(name)<=75),
        code TEXT NOT NULL CHECK (char_length(code)<=7),
        std_occupancy INT NOT NULL DEFAULT 2 CHECK (std_occupancy>0),
        min_occupancy INT NOT NULL DEFAULT 1 CHECK (
            min_occupancy>0
            AND min_occupancy<=std_occupancy
        ),
        max_occupancy INT NOT NULL DEFAULT 2 CHECK (
            max_occupancy>0
            AND max_occupancy>=std_occupancy
        ),
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now(),
        deleted_at TIMESTAMPTZ,
        UNIQUE (property_id, code),
        UNIQUE (property_id, name),
        UNIQUE (property_id, id)
    );

CREATE INDEX idx_room_types_property ON inventory.room_types (property_id);

CREATE TABLE IF NOT EXISTS
    relations.room_type_amenities (
        property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
        room_type_id UUID REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
        amenity_id UUID REFERENCES operations.amenities (id) ON DELETE RESTRICT,
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now(),
        deleted_at TIMESTAMPTZ DEFAULT NULL,
        -- Enforce that the room type belongs to the same property
        FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id) ON DELETE RESTRICT,
        -- Enforce that the amenity belongs to the same property as the room type
        FOREIGN KEY (property_id, amenity_id) REFERENCES operations.amenities (property_id, id) ON DELETE RESTRICT,
        PRIMARY KEY (room_type_id, amenity_id)
    );

CREATE INDEX idx_room_type_amenities_room_type ON relations.room_type_amenities (room_type_id);

CREATE INDEX idx_room_type_amenities_amenity ON relations.room_type_amenities (amenity_id);

-- Rooms
CREATE TABLE IF NOT EXISTS
    inventory.rooms (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
        room_type_id UUID REFERENCES inventory.room_types (id) ON DELETE SET NULL,
        name TEXT NOT NULL CHECK (char_length(name)<=75),
        housekeeping_status inventory.housekeeping_status NOT NULL DEFAULT 'clean',
        occupancy_status inventory.occupancy_status NOT NULL DEFAULT 'vacant',
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now(),
        deleted_at TIMESTAMPTZ,
        UNIQUE (property_id, id),
        UNIQUE (property_id, name),
        -- Enforce that the room type belongs to the same property
        FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id) ON DELETE SET NULL
    );

CREATE INDEX idx_rooms_room_type ON inventory.rooms (room_type_id);

CREATE INDEX idx_housekeeping_status ON inventory.rooms (housekeeping_status);

CREATE INDEX idx_occupancy_status ON inventory.rooms (occupancy_status);

CREATE INDEX idx_rooms_property ON inventory.rooms (property_id);

CREATE TABLE IF NOT EXISTS
    relations.room_amenities (
        property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
        room_id UUID REFERENCES inventory.rooms (id) ON DELETE RESTRICT,
        amenity_id UUID REFERENCES operations.amenities (id) ON DELETE RESTRICT,
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now(),
        deleted_at TIMESTAMPTZ DEFAULT NULL,
        -- Enforce that the room belongs to the same property
        FOREIGN KEY (property_id, room_id) REFERENCES inventory.rooms (property_id, id) ON DELETE RESTRICT,
        -- Enforce that the amenity belongs to the same property as the room
        FOREIGN KEY (property_id, amenity_id) REFERENCES operations.amenities (property_id, id) ON DELETE RESTRICT,
        PRIMARY KEY (room_id, amenity_id)
    );

CREATE INDEX idx_property_room_amenities_room ON relations.room_amenities (property_id, room_id);

CREATE INDEX idx_property_room_amenities_amenity ON relations.room_amenities (property_id, amenity_id);

-- Create maintenance blocks table
CREATE TABLE IF NOT EXISTS
    inventory.maintenance_blocks (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        room_id UUID REFERENCES inventory.rooms (id) ON DELETE RESTRICT NOT NULL,
        block_period TSTZRANGE NOT NULL CHECK (upper(block_period)>lower(block_period)),
        reason TEXT CHECK (char_length(reason)<=150) NOT NULL,
        type inventory.maintenance_block_type NOT NULL,
        created_by_user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL NOT NULL,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE (room_id, block_period),
        EXCLUDE USING GIST (
            room_id
            WITH
=,
                block_period
            WITH
&&
        )
    );

CREATE INDEX idx_maintenance_blocks_room_period ON inventory.maintenance_blocks (room_id, block_period);

CREATE INDEX idx_maintenance_blocks_period ON inventory.maintenance_blocks (block_period);

CREATE INDEX idx_maintenance_blocks_type ON inventory.maintenance_blocks (
    type
);

CREATE INDEX idx_maintenance_blocks_created_by ON inventory.maintenance_blocks (created_by_user_id);

CREATE INDEX idx_maintenance_blocks_room ON inventory.maintenance_blocks (room_id);

CREATE TABLE IF NOT EXISTS
    pricing.rate_plans (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
        name TEXT NOT NULL CHECK (char_length(name)<=30),
        code TEXT NOT NULL CHECK (char_length(code)<=7),
        description TEXT CHECK (char_length(description)<=300),
        parent_rate_plan_id UUID NULL,
        derivation_rule JSONB CHECK (
            derivation_rule?'type'
            AND derivation_rule?'value'
            AND derivation_rule->>'type' IN ('percentage', 'fixed')
            AND (derivation_rule->>'value')::NUMERIC<>0
        ), -- e.g., {"type": "percentage", "value": 10} for 10% above parent
        is_active BOOLEAN DEFAULT TRUE,
        currency_code TEXT NOT NULL DEFAULT 'GBP' CHECK (LENGTH(currency_code)=3), -- ISO 4217 currency code
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        CHECK (
            (
                parent_rate_plan_id IS NULL
                AND derivation_rule IS NULL
            )
            OR (
                parent_rate_plan_id IS NOT NULL
                AND derivation_rule IS NOT NULL
            )
        ), -- Both parent_rate_plan_id and derivation_rule must be either set or null
        -- Ensure that the parent rate plan belongs to the same property
        FOREIGN KEY (property_id, parent_rate_plan_id) REFERENCES pricing.rate_plans (property_id, id) ON DELETE SET NULL,
        UNIQUE (property_id, id),
        UNIQUE (property_id, code),
        UNIQUE (property_id, name)
    );

-- Add self-referencing foreign key for parent rate plans
ALTER TABLE pricing.rate_plans
ADD CONSTRAINT fk_parent_rate_plan FOREIGN KEY (parent_rate_plan_id) REFERENCES pricing.rate_plans (id) ON DELETE SET NULL;

CREATE INDEX idx_rate_plans_property ON pricing.rate_plans (property_id);

CREATE INDEX idx_property_rate_plans_parent_rate_plan ON pricing.rate_plans (property_id, parent_rate_plan_id);

CREATE INDEX idx_property_rate_plans_active ON pricing.rate_plans (property_id, is_active);

-- Company profiles
CREATE TABLE IF NOT EXISTS
    identity.company_profiles (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT NOT NULL,
        tax_id TEXT, -- e.g., VAT number
        negotiated_rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE SET NULL,
        company_name TEXT NOT NULL CHECK (
            char_length(company_name)<=50
            AND char_length(company_name)>=2
        ),
        contact_email TEXT,
        contact_phone TEXT,
        billing_address TEXT CHECK (char_length(billing_address)<=300),
        company_notes TEXT CHECK (char_length(company_notes)<=1500),
        has_credit_facility BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes,
        FOREIGN KEY (property_id, negotiated_rate_plan_id) REFERENCES pricing.rate_plans (property_id, id) ON DELETE SET NULL,
        UNIQUE (property_id, company_name),
        UNIQUE (tax_id, property_id),
        UNIQUE (property_id, id)
    );

CREATE INDEX idx_company_profiles_property ON identity.company_profiles (property_id);

CREATE INDEX idx_company_profiles_negotiated_rate_plan ON identity.company_profiles (property_id, negotiated_rate_plan_id);

-- Sets default date style to YYYY-MM-DD for consistency
-- across all date inputs and outputs
SET
    datestyle='ISO, YMD';

CREATE TABLE IF NOT EXISTS
    pricing.daily_price_grid (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
        room_type_id UUID REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
        rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE RESTRICT,
        calendar_date DATE NOT NULL CHECK (calendar_date>=NOW()),
        base_price_pence INTEGER NOT NULL DEFAULT 0, -- Store prices in pence to avoid floating point issues
        min_los_restriction INT DEFAULT 1 CHECK (min_los_restriction>0), -- Minimum length of stay restriction
        max_los_restriction INT DEFAULT 365 CHECK (
            max_los_restriction>0
            AND max_los_restriction>min_los_restriction
        ), -- Maximum length of stay restriction
        is_available BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        -- Ensure that the room type and rate plan belong to the same property
        FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id) ON DELETE RESTRICT,
        FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id) ON DELETE RESTRICT,
        UNIQUE (property_id, id),
        UNIQUE (
            property_id,
            room_type_id,
            rate_plan_id,
            calendar_date
        )
    );

CREATE INDEX idx_property_daily_price_grid_room_type ON pricing.daily_price_grid (property_id, room_type_id);

CREATE INDEX idx_property_daily_price_grid_rate_plan ON pricing.daily_price_grid (property_id, rate_plan_id);

CREATE INDEX idx_property_daily_price_grid_date ON pricing.daily_price_grid (property_id, calendar_date);

CREATE INDEX idx_property_daily_price_grid_available ON pricing.daily_price_grid (property_id, is_available);

CREATE INDEX idx_property_daily_price_grid_available_room_type_date ON pricing.daily_price_grid (property_id, room_type_id, calendar_date)
WHERE
    is_available=true;

CREATE INDEX idx_property_daily_price_grid_available_rate_plan_date ON pricing.daily_price_grid (property_id, rate_plan_id, calendar_date)
WHERE
    is_available=true;

-- +goose Down
DROP TABLE IF EXISTS pricing.daily_price_grid;

DROP TABLE IF EXISTS identity.company_profiles;

DROP TABLE IF EXISTS pricing.rate_plans;

DROP TABLE IF EXISTS inventory.maintenance_blocks;

DROP TABLE IF EXISTS relations.room_amenities;

DROP TABLE IF EXISTS relations.room_type_amenities;

DROP TABLE IF EXISTS inventory.rooms;

DROP TABLE IF EXISTS inventory.room_types;