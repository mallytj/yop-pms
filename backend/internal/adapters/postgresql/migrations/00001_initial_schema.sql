-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS licences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    licence_key TEXT UNIQUE NOT NULL,
    organisation_name TEXT NOT NULL,
    contact_email TEXT NOT NULL,
    licence_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS properties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    licence_id UUID REFERENCES licences(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    property_notes TEXT,
    timezone TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS property_amenities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID REFERENCES properties(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    short_code TEXT UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    licence_id UUID REFERENCES licences(id) ON DELETE SET NULL,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    role TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    entity TEXT NOT NULL,
    entity_id UUID,
    changes JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
    

CREATE TABLE IF NOT EXISTS guests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    id_document_data JSONB,
    marketing_opt_in BOOLEAN DEFAULT FALSE,
    guest_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS guest_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guest_id UUID REFERENCES guests(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    preference TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_name TEXT NOT NULL,
    group_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS room_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID REFERENCES properties(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    short_code TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    is_admin BOOLEAN DEFAULT FALSE,
    max_occupancy INT NOT NULL,
    base_inventory_count INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (property_id, short_code)
);

CREATE TABLE IF NOT EXISTS room_type_amenities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_type_id UUID REFERENCES room_types(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    short_code TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (room_type_id, short_code)
);

CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID REFERENCES properties(id) ON DELETE CASCADE,
    room_type_id UUID REFERENCES room_types(id) ON DELETE CASCADE,
    room_number TEXT NOT NULL,
    floor INT,
    status TEXT NOT NULL,
    room_notes TEXT,
    is_twinnable BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (property_id, room_number)
);

CREATE TABLE IF NOT EXISTS room_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    short_code TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (room_id, short_code)
);

CREATE TABLE IF NOT EXISTS daily_availability (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_date DATE NOT NULL,
    room_type_id UUID REFERENCES room_types(id) ON DELETE CASCADE,
    total_rooms INT NOT NULL,
    sold_count INT NOT NULL,
    decom_count INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (inventory_date, room_type_id)
);

CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID REFERENCES properties(id) ON DELETE CASCADE,
    primary_guest_id UUID REFERENCES guests(id) ON DELETE SET NULL,
    group_id UUID REFERENCES groups(id) ON DELETE SET NULL,
    booking_reference TEXT NOT NULL,
    status TEXT NOT NULL,
    source TEXT,
    reservation_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (property_id, booking_reference)
);

CREATE TABLE IF NOT EXISTS reservation_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id UUID REFERENCES reservations(id) ON DELETE CASCADE,
    assigned_room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    check_in_date DATE NOT NULL,
    check_out_date DATE NOT NULL,
    quoted_rate NUMERIC(10, 2) NOT NULL,
    reservation_room_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS do_not_moves (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_room_id UUID REFERENCES reservation_rooms(id) ON DELETE CASCADE,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS rate_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID REFERENCES properties(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    cancellation_policy TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    is_bookable BOOLEAN DEFAULT TRUE,
    description TEXT,
    short_code TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (property_id, short_code)
);

CREATE TABLE IF NOT EXISTS daily_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stay_date DATE NOT NULL,
    reservation_room_id UUID REFERENCES reservation_rooms(id) ON DELETE CASCADE,
    rate_plan_id UUID REFERENCES rate_plans(id) ON DELETE CASCADE,
    gross_price NUMERIC(10, 2) NOT NULL,
    net_price NUMERIC(10, 2) NOT NULL,
    currency TEXT NOT NULL,
    rate_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rate_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    daily_rate_id UUID REFERENCES daily_rates(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    amount NUMERIC(10,2) NOT NULL,
    is_approved BOOLEAN DEFAULT FALSE,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS housekeeping_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    status_to TEXT NOT NULL,
    status_from TEXT NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS housekeeping_logs;
DROP TABLE IF EXISTS rate_adjustments;
DROP TABLE IF EXISTS daily_rates;
DROP TABLE IF EXISTS rate_plans;
DROP TABLE IF EXISTS do_not_moves;
DROP TABLE IF EXISTS reservation_rooms;
DROP TABLE IF EXISTS reservations;
DROP TABLE IF EXISTS daily_availability;
DROP TABLE IF EXISTS room_features;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS room_type_amenities;
DROP TABLE IF EXISTS room_types;
DROP TABLE IF EXISTS property_amenities;
DROP TABLE IF EXISTS properties;
DROP TABLE IF EXISTS licences;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS guest_preferences;
DROP TABLE IF EXISTS guests;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
