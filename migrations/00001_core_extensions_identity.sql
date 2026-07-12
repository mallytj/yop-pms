-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gist";
CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Custom Domain
CREATE DOMAIN EMAIL as CITEXT
  CHECK ( value ~ '^[a-zA-Z0-9.!#$%&''*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$' );

-- Schemas
CREATE SCHEMA IF NOT EXISTS operations;
CREATE SCHEMA IF NOT EXISTS inventory;
CREATE SCHEMA IF NOT EXISTS pricing;
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS relations;

-- Base enums
CREATE TYPE auth.user_role AS ENUM('admin', 'manager', 'staff');
CREATE TYPE operations.reservation_source AS ENUM('website', 'internal', 'ota');
CREATE TYPE operations.reservation_status AS ENUM(
    'hold',
    'confirmed',
    'checked_in',
    'checked_out',
    'cancelled',
    'archived'
);
CREATE TYPE operations.reservation_item_status AS ENUM(
    'booked',
    'checked_in',
    'checked_out',
    'no_show',
    'cancelled',
    'archived',
    'overstay'
);
CREATE TYPE inventory.housekeeping_status AS ENUM(
    'clean',
    'dirty',
    'in_progress',
    'out_of_service',
    'linen_change',
    'inspected'
);
CREATE TYPE inventory.occupancy_status AS ENUM(
    'occupied',
    'vacant',
    'reserved',
    'out_of_service',
    'checked_out'
);
CREATE TYPE inventory.inventory_status AS ENUM(
    'available', 
    'sold', 
    'decommissioned', 
    'on_hold', 
    'maintenance'
);
CREATE TYPE inventory.maintenance_block_type AS ENUM(
    'deep_clean',
    'repair',
    'inspection',
    'out_of_service'
);

-- Core identities
CREATE TABLE operations.licences (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    licence_key CITEXT NOT NULL CHECK (licence_key ~ '^YOP-\d{5}$'),
    organisation_name TEXT NOT NULL CHECK (char_length(organisation_name) <= 50),
    contact_email EMAIL NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

ALTER TABLE operations.licences ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.licences FORCE ROW LEVEL SECURITY;
CREATE UNIQUE INDEX idx_active_licence_unique_key ON operations.licences (licence_key) WHERE (deleted_at IS NULL);
CREATE INDEX idx_licence_active ON operations.licences (is_active);

CREATE TABLE operations.properties (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    licence_id UUID NOT NULL REFERENCES operations.licences (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 50),
    address TEXT NOT NULL CHECK (char_length(address) <= 250),
    timezone TEXT NOT NULL CHECK (char_length(timezone) <= 100), -- Sanity check
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

ALTER TABLE operations.properties ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.properties FORCE ROW LEVEL SECURITY;
CREATE POLICY property_licence_isolation_policy ON operations.properties 
    USING (licence_id = current_setting('app.current_licence_id')::uuid, true);

CREATE INDEX idx_properties_licence ON operations.properties (licence_id);
CREATE UNIQUE INDEX idx_properties_addr_active ON operations.properties (licence_id, address) WHERE (deleted_at IS NULL);

CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    licence_id UUID NOT NULL REFERENCES operations.licences (id) ON DELETE RESTRICT,
    username CITEXT NOT NULL CHECK (char_length(username) <= 30 AND username ~ '^[a-zA-Z0-9_]+$'),
    email EMAIL NOT NULL,
    -- Check to see if it is in the hashing format
    password_hash TEXT NOT NULL CHECK (password_hash LIKE '$2%'),
    first_name TEXT NOT NULL CHECK (char_length(first_name) <= 75),
    last_name TEXT NOT NULL CHECK (char_length(last_name) <= 75),
    role auth.user_role NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.users FORCE ROW LEVEL SECURITY;
CREATE POLICY user_licence_isolation_policy ON auth.users 
    USING (licence_id = current_setting('app.current_licence_id')::uuid);

CREATE INDEX idx_users_licence ON auth.users (licence_id);
CREATE UNIQUE INDEX idx_active_user_unique_username ON auth.users (username) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX idx_active_user_unique_email ON auth.users (email) WHERE (deleted_at IS NULL);

CREATE TABLE identity.guests (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    first_name TEXT NOT NULL CHECK (char_length(first_name) <= 50),
    last_name TEXT NOT NULL CHECK (char_length(last_name) <= 50),
    email CITEXT CHECK (char_length(email) <= 100),
    phone_number TEXT CHECK (char_length(phone_number) <= 20),
    preferences JSONB DEFAULT '{}'::jsonb,
    notes TEXT CHECK (char_length(notes) <= 1500),
    marketing_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    is_anonymised BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (property_id, id)
);

ALTER TABLE identity.guests ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity.guests FORCE ROW LEVEL SECURITY;
CREATE POLICY guest_property_isolation_policy ON identity.guests 
    USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_guests_property ON identity.guests (property_id);
CREATE INDEX idx_guests_search ON identity.guests (property_id, last_name, first_name);
CREATE INDEX idx_guests_email_search ON identity.guests (property_id, email) WHERE (email IS NOT NULL AND deleted_at IS NULL);
CREATE INDEX idx_guests_phone_search ON identity.guests (property_id, phone_number) WHERE (phone_number IS NOT NULL AND deleted_at IS NULL);

-- Helps with autocomplete search for guests by first name, last name, and email
-- Example usage
-- SELECT * FROM identity.guests
-- WHERE property_id = :property_id
--   AND deleted_at IS NULL
--   -- Backend splits "Smith John" into two terms:
--   AND (COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(email, '')) ILIKE :term_1 -- '%Smith%'
--   AND (COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(email, '')) ILIKE :term_2; -- '%John%'
CREATE INDEX idx_guests_autocomplete ON identity.guests 
USING GIN (
    property_id, 
    (COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(email, '')) gin_trgm_ops
)
WHERE (deleted_at IS NULL);

CREATE INDEX idx_guests_composite_search_trgm 
ON identity.guests 
USING GIN (property_id, (first_name || ' ' || last_name || ' ' || COALESCE(email, '')) gin_trgm_ops)
WHERE (deleted_at IS NULL);

-- +goose Down
DROP TABLE IF EXISTS identity.guests CASCADE;
DROP TABLE IF EXISTS auth.users CASCADE;
DROP TABLE IF EXISTS operations.properties CASCADE;
DROP TABLE IF EXISTS operations.licences CASCADE;

DROP TYPE IF EXISTS inventory.maintenance_block_type CASCADE;
DROP TYPE IF EXISTS inventory.inventory_status CASCADE;
DROP TYPE IF EXISTS inventory.occupancy_status CASCADE;
DROP TYPE IF EXISTS auth.user_role CASCADE;
DROP TYPE IF EXISTS operations.reservation_source CASCADE;
DROP TYPE IF EXISTS operations.reservation_status CASCADE;
DROP TYPE IF EXISTS operations.reservation_item_status CASCADE;
DROP TYPE IF EXISTS inventory.housekeeping_status CASCADE;

DROP SCHEMA IF EXISTS relations CASCADE;
DROP SCHEMA IF EXISTS auth CASCADE;
DROP SCHEMA IF EXISTS identity CASCADE;
DROP SCHEMA IF EXISTS sales_ledgers CASCADE;
DROP SCHEMA IF EXISTS finance CASCADE;
DROP SCHEMA IF EXISTS operations CASCADE;
DROP SCHEMA IF EXISTS inventory CASCADE;
DROP SCHEMA IF EXISTS pricing CASCADE;

DROP EXTENSION IF EXISTS "btree_gin" CASCADE;
DROP EXTENSION IF EXISTS "pg_stat_statements" CASCADE;
DROP EXTENSION IF EXISTS "pg_trgm" CASCADE;
DROP EXTENSION IF EXISTS "citext" CASCADE;
DROP EXTENSION IF EXISTS "btree_gist" CASCADE;
DROP EXTENSION IF EXISTS "pgcrypto" CASCADE;
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
