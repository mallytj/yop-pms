-- +goose Up
-- Licences
CREATE TABLE operations.licences (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  licence_key TEXT UNIQUE NOT NULL,
  organisation_name TEXT NOT NULL,
  contact_email TEXT NOT NULL,
  licence_notes TEXT,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_licence_key ON operations.licences(licence_key);
CREATE INDEX idx_licence_active ON operations.licences(is_active);

-- Properties
CREATE TABLE operations.properties (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  licence_id UUID REFERENCES operations.licences(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  address TEXT NOT NULL,
  timezone TEXT NOT NULL,
  property_notes TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  is_active BOOLEAN DEFAULT TRUE,
  UNIQUE (licence_id, address)
);

CREATE INDEX idx_properties_name ON operations.properties(name);
CREATE INDEX idx_properties_licence ON operations.properties(licence_id);
CREATE INDEX idx_properties_active ON operations.properties(is_active);

-- Users
CREATE TABLE auth.users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  licence_id UUID REFERENCES operations.licences(id) ON DELETE SET NULL,
  username TEXT UNIQUE NOT NULL,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  role auth.user_role NOT NULL,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_licence ON auth.users(licence_id);
CREATE INDEX idx_users_role ON auth.users(role);
CREATE INDEX idx_users_active ON auth.users(is_active);
CREATE INDEX idx_users_name ON auth.users(last_name, first_name);

-- Guests
CREATE TABLE identity.guests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  property_id UUID REFERENCES operations.properties(id) ON DELETE CASCADE,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  email TEXT,
  phone_number TEXT,
  preferences JSONB,
  notes TEXT,
  marketing_opt_in BOOLEAN DEFAULT FALSE,
  is_anonymised BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_guests_name ON identity.guests(last_name, first_name);
CREATE INDEX idx_guests_email ON identity.guests(email);
CREATE INDEX idx_guests_phone ON identity.guests(phone_number);
CREATE INDEX idx_guests_marketing_opt_in ON identity.guests(marketing_opt_in);
CREATE INDEX idx_guests_anonymised ON identity.guests(is_anonymised);
CREATE INDEX idx_guests_property ON identity.guests(property_id);

CREATE TABLE IF NOT EXISTS
  auth.audit_logs (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID REFERENCES auth.users (id) ON DELETE SET NULL,
      action auth.audit_log_action NOT NULL,
      entity auth.audit_log_entity NOT NULL,
      entity_id UUID,
      changes JSONB CHECK (
          changes IS NOT NULL AND jsonb_typeof(changes) = 'object' AND changes <> '{}'::JSONB 
          AND (
              (CHANGES ? 'field' AND changes->>'field' IS NOT NULL)
              OR
              (changes ? 'before' AND changes->'before' IS NOT NULL)
              OR
              (changes ? 'after' AND changes->'after' IS NOT NULL)
          )
      ),
      created_at TIMESTAMPTZ DEFAULT NOW(),
      updated_at TIMESTAMPTZ DEFAULT NOW(),
      deleted_at TIMESTAMPTZ DEFAULT NULL
  );

CREATE INDEX idx_audit_logs_user ON auth.audit_logs(user_id);

CREATE INDEX idx_audit_logs_entity ON auth.audit_logs(entity, entity_id);

CREATE INDEX idx_audit_logs_action ON auth.audit_logs(action);

-- Amenities
CREATE TABLE IF NOT EXISTS
    operations.amenities (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE, -- The owner of the amenities, separate to a property's amenities
        name TEXT NOT NULL,
        short_code TEXT,
        description TEXT,
        is_active BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE (property_id, short_code),
        UNIQUE (property_id, name)
    );

CREATE INDEX idx_amenities_property ON operations.amenities(property_id);
CREATE INDEX idx_amenities_active ON operations.amenities(is_active);

-- Join table for property amenities
CREATE TABLE IF NOT EXISTS
    relations.property_amenities (
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        amenity_id UUID REFERENCES operations.amenities (id) ON DELETE CASCADE,
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now(),
        deleted_at TIMESTAMPTZ DEFAULT NULL,
        PRIMARY KEY (property_id, amenity_id)
    );

-- Travel agents
CREATE TABLE IF NOT EXISTS
    identity.travel_agents (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        property_id UUID REFERENCES operations.properties (id) ON DELETE CASCADE,
        name TEXT NOT NULL,
        contact_email TEXT,
        contact_phone TEXT,
        agency_notes TEXT,
        iata_code TEXT,
        commission_percent NUMERIC(5, 2) DEFAULT 0.00 CHECK(commission_percent >= 0.00 AND commission_percent <= 75.00), -- e.g., 10.00 for 10%
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL, -- For soft deletes
        UNIQUE(property_id, name)
    );

CREATE INDEX idx_travel_agents_property ON identity.travel_agents(property_id);

-- Identity Docs
CREATE TABLE IF NOT EXISTS
    identity.identity_docs (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        guest_id UUID REFERENCES identity.guests (id) ON DELETE CASCADE,
        doc_type identity.identity_doc_type NOT NULL,
        encrypted_doc_number TEXT NOT NULL CHECK (LENGTH(encrypted_doc_number) > 0), -- Encrypted for security
        issuing_country TEXT, -- ISO country code
        expiry_date DATE, -- Nullable for documents without expiry
        doc_image_url TEXT, -- URL to the stored document image
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        deleted_at TIMESTAMPTZ DEFAULT NULL -- For soft deletes
    );

CREATE INDEX idx_identity_docs_guest ON identity.identity_docs(guest_id);

-- +goose Down
DROP TABLE IF EXISTS identity.travel_agents;
DROP TABLE IF EXISTS relations.property_amenities;
DROP TABLE IF EXISTS operations.amenities;
DROP TABLE IF EXISTS auth.audit_logs;
DROP TABLE IF EXISTS auth.users;
DROP TABLE IF EXISTS identity.identity_docs;
DROP TABLE IF EXISTS identity.guests;
DROP TABLE IF EXISTS operations.properties;
DROP TABLE IF EXISTS operations.licences;
