-- +goose Up
-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gist";
CREATE EXTENSION IF NOT EXISTS "citext";
-- Schemas
CREATE SCHEMA IF NOT EXISTS operations;
CREATE SCHEMA IF NOT EXISTS inventory;
CREATE SCHEMA IF NOT EXISTS pricing;
CREATE SCHEMA IF NOT EXISTS finance;
CREATE SCHEMA IF NOT EXISTS sales_ledgers;
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS relations;
-- Enums
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
CREATE TYPE finance.folio_part AS ENUM('A', 'B', 'C');
CREATE TYPE finance.folio_transaction_status AS ENUM(
    'pending',
    'posted',
    'voided',
    'reversed',
    'transferred'
);
CREATE TYPE sales_ledgers.transaction_type AS ENUM(
    'charge',
    'payment',
    'adjustment',
    'invoice_credit',
    'refund'
);
CREATE TYPE identity.identity_doc_type AS ENUM('passport', 'id_card', 'driver_license', 'other');
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION operations.fn_chk_licence_is_active () RETURNS TRIGGER AS $$ BEGIN IF NOT EXISTS (
        SELECT 1
        FROM operations.licences
        WHERE id = NEW.licence_id
            AND is_active = TRUE
    ) THEN RAISE EXCEPTION 'PR-001: Cannot assign property to an inactive or non-existent licence (ID: %)',
    NEW.licence_id;
END IF;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS operations.fn_chk_licence_is_active();
DROP TYPE IF EXISTS operations.checkout_session_status;
DROP TYPE IF EXISTS operations.reservation_guest_role;
DROP TYPE IF EXISTS operations.reservation_item_status;
DROP TYPE IF EXISTS operations.reservation_status;
DROP TYPE IF EXISTS operations.reservation_source;
DROP TYPE IF EXISTS inventory.maintenance_block_type;
DROP TYPE IF EXISTS inventory.inventory_status;
DROP TYPE IF EXISTS inventory.occupancy_status;
DROP TYPE IF EXISTS inventory.housekeeping_status;
DROP TYPE IF EXISTS finance.folio_transaction_status;
DROP TYPE IF EXISTS finance.folio_part;
DROP TYPE IF EXISTS sales_ledgers.transaction_type;
DROP TYPE IF EXISTS identity.identity_doc_type;
DROP TYPE IF EXISTS auth.audit_log_action;
DROP TYPE IF EXISTS auth.audit_log_entity;
DROP TYPE IF EXISTS auth.user_role;
DROP SCHEMA IF EXISTS operations;
DROP SCHEMA IF EXISTS inventory;
DROP SCHEMA IF EXISTS pricing;
DROP SCHEMA IF EXISTS finance;
DROP SCHEMA IF EXISTS sales_ledgers;
DROP SCHEMA IF EXISTS identity;
DROP SCHEMA IF EXISTS auth;
DROP SCHEMA IF EXISTS relations;
DROP EXTENSION IF EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "btree_gist";
DROP EXTENSION IF EXISTS "citext";
-- +goose StatementEnd