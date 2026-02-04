-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Schemas
CREATE SCHEMA IF NOT EXISTS operations;

CREATE SCHEMA IF NOT EXISTS inventory;

CREATE SCHEMA IF NOT EXISTS pricing;

CREATE SCHEMA IF NOT EXISTS finance;

CREATE SCHEMA IF NOT EXISTS sales_ledgers;

CREATE SCHEMA IF NOT EXISTS identity;

CREATE SCHEMA IF NOT EXISTS auth;

CREATE SCHEMA IF NOT EXISTS relations;

-- =========================
-- ENUMS
-- =========================
-- Auth
CREATE TYPE auth.user_role AS ENUM('admin', 'manager', 'staff');

CREATE TYPE auth.audit_log_entity AS ENUM(
  'user',
  'property',
  'reservation',
  'room',
  'guest',
  'rate_plan',
  'folio',
  'transaction'
);

-- Operations
CREATE TYPE auth.audit_log_action AS ENUM(
  'create',
  'update',
  'delete',
  'login',
  'logout',
  'post_transaction'
);

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
  'archived'
);

CREATE TYPE operations.reservation_guest_role AS ENUM(
  'primary',
  'additional',
  'vip',
  'booker_not_staying'
);

CREATE TYPE operations.checkout_session_status AS ENUM('pending', 'completed', 'expired', 'cancelled');

-- Inventory
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

CREATE TYPE inventory.inventory_status AS ENUM('available', 'sold', 'decommissioned', 'on_hold');

CREATE TYPE inventory.maintenance_block_type AS ENUM(
  'deep_clean',
  'repair',
  'inspection',
  'out_of_service'
);

-- Finance
CREATE TYPE finance.folio_part AS ENUM('A', 'B', 'C');

CREATE TYPE finance.folio_transaction_status AS ENUM(
  'pending',
  'posted',
  'voided',
  'reversed',
  'transferred'
);

-- Sales ledger
CREATE TYPE sales_ledgers.transaction_type AS ENUM(
  'charge',
  'payment',
  'adjustment',
  'invoice_credit',
  'refund'
);

-- Identity
CREATE TYPE identity.identity_doc_type AS ENUM('passport', 'id_card', 'driver_license', 'other');

-- Function for checking if a licence is active
-- +goose StatementBegin
CREATE
OR REPLACE FUNCTION operations.check_licence_is_active () RETURNS TRIGGER AS $$
BEGIN
  -- Check if the linked licence exists and is active
  -- We perform a direct lookup on the parent table
  IF NOT EXISTS (
    SELECT 1 
    FROM operations.licences 
    WHERE id = NEW.licence_id 
      AND is_active = TRUE
  ) THEN
    RAISE EXCEPTION 'Cannot assign property to an inactive or non-existent licence (ID: %)', NEW.licence_id;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd
-- Function for checking the occupanncy bounds are valid
-- +goose StatementBegin
CREATE
OR REPLACE FUNCTION operations.fn_validate_room_occupancy () RETURNS TRIGGER AS $$
DECLARE
    v_min_occ INT;
    v_max_occ INT;
BEGIN
    -- Look up the occupancy rules for the specific room type
    SELECT min_occupancy, max_occupancy 
    INTO v_min_occ, v_max_occ
    FROM inventory.room_types 
    WHERE id = NEW.booked_room_type_id;

    -- Check Max
    IF (NEW.adults_count + NEW.children_count) > v_max_occ THEN
        RAISE EXCEPTION 'Total occupancy (%) exceeds room type maximum (%)', 
            (NEW.adults_count + NEW.children_count), v_max_occ;
    END IF;

    -- Check Min
    IF (NEW.adults_count + NEW.children_count) < v_min_occ THEN
        RAISE EXCEPTION 'Total occupancy (%) is below room type minimum (%)', 
            (NEW.adults_count + NEW.children_count), v_min_occ;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd
-- Function for calculating final price from base price and adjustment
-- +goose StatementBegin
CREATE
OR REPLACE FUNCTION pricing.fn_calculate_final_price () RETURNS TRIGGER AS $$
DECLARE 
    v_final_price INT;
BEGIN
    IF NEW.adjustment IS NULL OR NEW.adjustment = '{}' THEN
        v_final_price := NEW.base_price_pence;
    ELSIF NEW.adjustment->>'type' = 'percentage' THEN
        v_final_price := NEW.base_price_pence + (
            (NEW.base_price_pence * (NEW.adjustment->>'value')::INT) / 100
        )::INT;
    ELSIF NEW.adjustment->>'type' = 'fixed' THEN
        v_final_price := NEW.base_price_pence + (NEW.adjustment->>'value')::INT;
    ELSE
        v_final_price := NEW.base_price_pence;
    END IF;
    IF v_final_price < 0 THEN
        NEW.final_price_pence := 0;
        RAISE EXCEPTION 'Final price cannot be negative';
    END IF;
    NEW.final_price_pence := v_final_price;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd
-- +goose Down
DROP FUNCTION IF EXISTS operations.check_licence_is_active ();

DROP FUNCTION IF EXISTS operations.fn_validate_room_occupancy ();

DROP FUNCTION IF EXISTS pricing.fn_calculate_final_price (INT, JSONB);

DROP SCHEMA IF EXISTS relations CASCADE;

DROP SCHEMA IF EXISTS inventory CASCADE;

DROP SCHEMA IF EXISTS pricing CASCADE;

DROP SCHEMA IF EXISTS finance CASCADE;

DROP SCHEMA IF EXISTS sales_ledgers CASCADE;

DROP SCHEMA IF EXISTS identity CASCADE;

DROP SCHEMA IF EXISTS auth CASCADE;

DROP SCHEMA IF EXISTS operations CASCADE;

DROP EXTENSION IF EXISTS btree_gist;

DROP EXTENSION IF EXISTS "pgcrypto";

DROP EXTENSION IF EXISTS "uuid-ossp";

DROP TYPE IF EXISTS auth.audit_log_entity;

DROP TYPE IF EXISTS sales_ledgers.transaction_type;

DROP TYPE IF EXISTS finance.folio_transaction_status;

DROP TYPE IF EXISTS finance.folio_part;

DROP TYPE IF EXISTS operations.checkout_session_status;

DROP TYPE IF EXISTS operations.reservation_item_status;

DROP TYPE IF EXISTS operations.reservation_status;

DROP TYPE IF EXISTS operations.reservation_source;

DROP TYPE IF EXISTS auth.audit_action;

DROP TYPE IF EXISTS auth.user_role;

DROP TYPE IF EXISTS inventory.maintenance_block_type;

DROP TYPE IF EXISTS inventory.inventory_status;

DROP TYPE IF EXISTS inventory.occupancy_status;

DROP TYPE IF EXISTS inventory.housekeeping_status;

DROP TYPE IF EXISTS operations.reservation_guest_role;

DROP TYPE IF EXISTS sales_ledgers.transaction_type;

DROP TYPE IF EXISTS identity.identity_doc_type;