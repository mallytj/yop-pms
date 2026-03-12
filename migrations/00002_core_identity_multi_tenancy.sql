-- +goose Up

-- ========================================================
-- 1. LICENCES
-- ========================================================
CREATE TABLE operations.licences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    licence_key CITEXT NOT NULL CHECK (licence_key ~ '^YOP-\d{5}$'),
    organisation_name TEXT NOT NULL CHECK (char_length(organisation_name) <= 50),
    contact_email CITEXT NOT NULL,
    licence_notes TEXT CHECK (char_length(licence_notes) <= 1500),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- REQ-016: Partial uniqueness for soft deletes
CREATE UNIQUE INDEX idx_licence_key_unique_active ON operations.licences (licence_key) WHERE (deleted_at IS NULL);
CREATE INDEX idx_licence_active ON operations.licences (is_active);

-- ========================================================
-- 2. PROPERTIES
-- ========================================================
CREATE TABLE operations.properties (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    licence_id UUID NOT NULL REFERENCES operations.licences (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 50),
    address TEXT NOT NULL CHECK (char_length(address) <= 250),
    timezone TEXT NOT NULL,
    property_notes TEXT CHECK (char_length(property_notes) <= 1500),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- REQ-020: Enable RLS for Licence isolation
ALTER TABLE operations.properties ENABLE ROW LEVEL SECURITY;
CREATE POLICY property_licence_isolation_policy ON operations.properties 
    USING (licence_id = current_setting('app.current_licence_id')::uuid);

CREATE TRIGGER trg_check_licence_is_active 
BEFORE INSERT OR UPDATE ON operations.properties 
FOR EACH ROW EXECUTE FUNCTION operations.fn_chk_licence_is_active();

CREATE INDEX idx_properties_licence ON operations.properties (licence_id);
CREATE UNIQUE INDEX idx_properties_addr_active ON operations.properties (licence_id, address) WHERE (deleted_at IS NULL);

-- ========================================================
-- 3. USERS
-- ========================================================
CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    licence_id UUID NOT NULL REFERENCES operations.licences (id) ON DELETE RESTRICT,
    username CITEXT NOT NULL CHECK (char_length(username) <= 20 AND username ~ '^[a-zA-Z0-9_]+$'),
    email CITEXT NOT NULL,
    password_hash TEXT NOT NULL CHECK (length(password_hash) >= 60 AND password_hash LIKE '$2%'),
    first_name TEXT NOT NULL CHECK (char_length(first_name) <= 50),
    last_name TEXT NOT NULL CHECK (char_length(last_name) <= 50),
    role auth.user_role NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
CREATE POLICY user_licence_isolation_policy ON auth.users 
    USING (licence_id = current_setting('app.current_licence_id')::uuid);

CREATE TRIGGER trg_check_user_licence_is_active BEFORE INSERT OR UPDATE ON auth.users 
FOR EACH ROW EXECUTE FUNCTION operations.fn_chk_licence_is_active();

CREATE UNIQUE INDEX idx_users_username_active ON auth.users (username) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX idx_users_email_active ON auth.users (email) WHERE (deleted_at IS NULL);

-- ========================================================
-- 4. GUESTS
-- ========================================================
CREATE TABLE identity.guests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    first_name TEXT NOT NULL CHECK (char_length(first_name) <= 50),
    last_name TEXT NOT NULL CHECK (char_length(last_name) <= 50),
    email CITEXT,
    phone_number TEXT,
    preferences JSONB DEFAULT '{}'::jsonb,
    notes TEXT,
    marketing_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    is_anonymised BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (property_id, id)
);

ALTER TABLE identity.guests ENABLE ROW LEVEL SECURITY;
CREATE POLICY guest_property_isolation_policy ON identity.guests 
    USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_guests_property ON identity.guests (property_id);
CREATE INDEX idx_guests_search ON identity.guests (property_id, last_name, first_name);

-- ========================================================
-- 5. AUDIT LOGS
-- ========================================================
CREATE TABLE auth.audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL,
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE CASCADE,
    action auth.audit_log_action NOT NULL,
    entity auth.audit_log_entity NOT NULL,
    entity_id UUID NOT NULL,
    changes JSONB NOT NULL CHECK (jsonb_typeof(changes) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE auth.audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_logs_isolation_policy ON auth.audit_logs 
    USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_audit_logs_property_entity ON auth.audit_logs (property_id, entity, entity_id);

-- ========================================================
-- 6. AMENITIES
-- ========================================================
CREATE TABLE operations.amenities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 100),
    short_code TEXT CHECK (short_code ~ '^[A-Z0-9_/]{2,5}$'),
    description TEXT CHECK (char_length(description) <= 250),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (property_id, id)
);

ALTER TABLE operations.amenities ENABLE ROW LEVEL SECURITY;
CREATE POLICY amenities_isolation_policy ON operations.amenities 
    USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE UNIQUE INDEX idx_amenities_name_active ON operations.amenities (property_id, name) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX idx_amenities_code_active ON operations.amenities (property_id, short_code) WHERE (deleted_at IS NULL);

-- ========================================================
-- 7. JOIN TABLES & RELATIONS
-- ========================================================
CREATE TABLE relations.property_amenities (
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    amenity_id UUID NOT NULL REFERENCES operations.amenities (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (property_id, amenity_id)
);

-- Travel Agents
CREATE TABLE identity.travel_agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 100),
    contact_email CITEXT,
    contact_phone TEXT,
    agency_notes TEXT CHECK (char_length(agency_notes) <= 1000),
    iata_code TEXT,
    commission_percent NUMERIC(5, 2) NOT NULL DEFAULT 0.00 CHECK (commission_percent BETWEEN 0 AND 75),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (property_id, id)
);

ALTER TABLE identity.travel_agents ENABLE ROW LEVEL SECURITY;
CREATE POLICY travel_agents_isolation_policy ON identity.travel_agents 
    USING (property_id = current_setting('app.current_property_id')::uuid);

-- Identity Docs
CREATE TABLE identity.identity_docs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    guest_id UUID NOT NULL REFERENCES identity.guests (id) ON DELETE RESTRICT,
    doc_type identity.identity_doc_type NOT NULL,
    encrypted_doc_number TEXT NOT NULL,
    issuing_country TEXT,
    expiry_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

ALTER TABLE identity.identity_docs ENABLE ROW LEVEL SECURITY;
CREATE POLICY identity_docs_isolation_policy ON identity.identity_docs 
    USING (property_id = current_setting('app.current_property_id')::uuid);

-- +goose Down
DROP TABLE IF EXISTS identity.identity_docs;
DROP TABLE IF EXISTS identity.travel_agents;
DROP TABLE IF EXISTS relations.property_amenities;
DROP TABLE IF EXISTS operations.amenities;
DROP TABLE IF EXISTS auth.audit_logs;
DROP TABLE IF EXISTS identity.guests;
DROP TABLE IF EXISTS auth.users;
DROP TABLE IF EXISTS operations.properties;
DROP TABLE IF EXISTS operations.licences;