-- +goose Up
-- +goose NO TRANSACTION
--
-- Reservation API preparation migration (M1-M5 from docs/requirements/reservations.md §13)
--
-- NO TRANSACTION required because ALTER TYPE ADD VALUE for `inventory.inventory_status`
-- cannot reference the new enum value within the same transaction that added it
-- (PG 12+ restriction). Safe on greenfield schema with no live data.
--
-- M1: Per-property reservation + group code sequences
-- M2: Optimistic locking column on reservation_items
-- M3: `maintenance` ledger status + maintenance_block_id FK
-- M4: Property settings table
-- M5: OTA inbound dedup table

-- ============================================================
-- M1: Per-property reservation_sequences + triggers
-- ============================================================

CREATE TABLE operations.reservation_sequences (
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('reservation', 'group')),
    next_value BIGINT NOT NULL DEFAULT 1 CHECK (next_value > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (property_id, entity_type)
);

ALTER TABLE operations.reservation_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.reservation_sequences FORCE ROW LEVEL SECURITY;
CREATE POLICY res_seq_isolation ON operations.reservation_sequences
    USING (property_id = current_setting('app.current_property_id')::uuid);

-- Drop existing GENERATED code columns + SERIAL sequential columns (CASCADE removes dependent indexes/constraints)
ALTER TABLE operations.reservations DROP COLUMN code CASCADE;
ALTER TABLE operations.reservations DROP COLUMN sequential CASCADE;

ALTER TABLE operations.reservation_groups DROP COLUMN code CASCADE;
ALTER TABLE operations.reservation_groups DROP COLUMN sequential CASCADE;

-- Re-add as plain columns; trigger populates BEFORE INSERT
ALTER TABLE operations.reservations ADD COLUMN sequential BIGINT;
ALTER TABLE operations.reservations ADD COLUMN code TEXT;
ALTER TABLE operations.reservations ADD CONSTRAINT chk_reservations_code CHECK (code ~ '^RES-[0-9]{6}$');

ALTER TABLE operations.reservation_groups ADD COLUMN sequential BIGINT;
ALTER TABLE operations.reservation_groups ADD COLUMN code TEXT;
ALTER TABLE operations.reservation_groups ADD CONSTRAINT chk_reservation_groups_code CHECK (code ~ '^GRP-[0-9]{5}$');

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION operations.fn_assign_reservation_code() RETURNS TRIGGER AS $$
DECLARE
    next_seq BIGINT;
BEGIN
    INSERT INTO operations.reservation_sequences (property_id, entity_type, next_value)
        VALUES (NEW.property_id, 'reservation', 2)
        ON CONFLICT (property_id, entity_type) DO UPDATE
            SET next_value = operations.reservation_sequences.next_value + 1,
                updated_at = NOW()
        RETURNING next_value - 1 INTO next_seq;

    NEW.sequential := next_seq;
    NEW.code := 'RES-' || LPAD(next_seq::TEXT, 6, '0');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION operations.fn_assign_group_code() RETURNS TRIGGER AS $$
DECLARE
    next_seq BIGINT;
BEGIN
    INSERT INTO operations.reservation_sequences (property_id, entity_type, next_value)
        VALUES (NEW.property_id, 'group', 2)
        ON CONFLICT (property_id, entity_type) DO UPDATE
            SET next_value = operations.reservation_sequences.next_value + 1,
                updated_at = NOW()
        RETURNING next_value - 1 INTO next_seq;

    NEW.sequential := next_seq;
    NEW.code := 'GRP-' || LPAD(next_seq::TEXT, 5, '0');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_assign_reservation_code
    BEFORE INSERT ON operations.reservations
    FOR EACH ROW EXECUTE FUNCTION operations.fn_assign_reservation_code();

CREATE TRIGGER trg_assign_group_code
    BEFORE INSERT ON operations.reservation_groups
    FOR EACH ROW EXECUTE FUNCTION operations.fn_assign_group_code();

-- Enforce per-property uniqueness on the freshly populated codes
CREATE UNIQUE INDEX idx_reservations_code_per_property
    ON operations.reservations (property_id, code)
    WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX idx_reservation_groups_code_per_property
    ON operations.reservation_groups (property_id, code)
    WHERE (deleted_at IS NULL);

-- After backfill (none on greenfield), enforce NOT NULL
ALTER TABLE operations.reservations ALTER COLUMN sequential SET NOT NULL;
ALTER TABLE operations.reservations ALTER COLUMN code SET NOT NULL;
ALTER TABLE operations.reservation_groups ALTER COLUMN sequential SET NOT NULL;
ALTER TABLE operations.reservation_groups ALTER COLUMN code SET NOT NULL;

-- ============================================================
-- M2: Optimistic locking on reservation_items
-- ============================================================

ALTER TABLE operations.reservation_items
    ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- ============================================================
-- M3: maintenance ledger status + maintenance_block_id FK
-- ============================================================

ALTER TYPE inventory.inventory_status ADD VALUE IF NOT EXISTS 'maintenance';

ALTER TABLE inventory.room_inventory_ledger
    ADD COLUMN maintenance_block_id UUID;

ALTER TABLE inventory.room_inventory_ledger
    ADD CONSTRAINT fk_inv_ledger_maint_block
    FOREIGN KEY (maintenance_block_id)
    REFERENCES inventory.maintenance_blocks (id)
    ON DELETE SET NULL;

CREATE INDEX idx_inv_ledger_maintenance_block
    ON inventory.room_inventory_ledger (maintenance_block_id)
    WHERE (maintenance_block_id IS NOT NULL);

-- Drop existing unnamed CHECK constraints on the ledger that reference status
-- +goose StatementBegin
DO $$
DECLARE
    cname TEXT;
BEGIN
    FOR cname IN
        SELECT conname FROM pg_constraint
        WHERE conrelid = 'inventory.room_inventory_ledger'::regclass
            AND contype = 'c'
            AND (pg_get_constraintdef(oid) LIKE '%status = ''sold''%'
                 OR pg_get_constraintdef(oid) LIKE '%status = ''on_hold''%')
    LOOP
        EXECUTE format('ALTER TABLE inventory.room_inventory_ledger DROP CONSTRAINT %I', cname);
    END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE inventory.room_inventory_ledger
    ADD CONSTRAINT inv_ledger_status_consistency CHECK (
        (status = 'sold' AND reservation_id IS NOT NULL) OR
        (status = 'on_hold' AND checkout_session_id IS NOT NULL) OR
        (status = 'maintenance' AND maintenance_block_id IS NOT NULL) OR
        status IN ('available', 'decommissioned')
    );

-- ============================================================
-- M4: Property settings
-- ============================================================

CREATE TABLE operations.property_settings (
    property_id UUID PRIMARY KEY REFERENCES operations.properties (id) ON DELETE RESTRICT,
    website_hold_ttl_seconds INT NOT NULL DEFAULT 900 CHECK (website_hold_ttl_seconds > 0),
    internal_hold_ttl_seconds INT NOT NULL DEFAULT 86400 CHECK (internal_hold_ttl_seconds > 0),
    reservation_archive_after_days INT NOT NULL DEFAULT 365 CHECK (reservation_archive_after_days > 0),
    housekeeping_buffer_minutes INT NOT NULL DEFAULT 0 CHECK (housekeeping_buffer_minutes >= 0),
    no_show_grace_minutes INT NOT NULL DEFAULT 360 CHECK (no_show_grace_minutes >= 0),
    max_stay_length_days INT NOT NULL DEFAULT 90 CHECK (max_stay_length_days > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE operations.property_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.property_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY property_settings_isolation ON operations.property_settings
    USING (property_id = current_setting('app.current_property_id')::uuid);

-- ============================================================
-- M5: OTA inbound dedup table 
-- NOTE: Deferred to future PR
-- ============================================================

-- CREATE TABLE operations.ota_inbound_messages (
--     channel_id TEXT NOT NULL CHECK (char_length(channel_id) <= 50),
--     channel_message_id TEXT NOT NULL CHECK (char_length(channel_message_id) <= 255),
--     property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
--     processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     response_jsonb JSONB,
--     PRIMARY KEY (channel_id, channel_message_id)
-- );
--
-- CREATE INDEX idx_ota_inbound_property ON operations.ota_inbound_messages (property_id);
-- CREATE INDEX idx_ota_inbound_processed ON operations.ota_inbound_messages (processed_at);

-- +goose Down
-- +goose NO TRANSACTION

DROP TABLE IF EXISTS operations.ota_inbound_messages;
DROP TABLE IF EXISTS operations.property_settings;

-- M3 rollback
-- +goose StatementBegin
DO $$
DECLARE
    cname TEXT;
BEGIN
    FOR cname IN
        SELECT conname FROM pg_constraint
        WHERE conrelid = 'inventory.room_inventory_ledger'::regclass
            AND conname = 'inv_ledger_status_consistency'
    LOOP
        EXECUTE format('ALTER TABLE inventory.room_inventory_ledger DROP CONSTRAINT %I', cname);
    END LOOP;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS inventory.idx_inv_ledger_maintenance_block;
ALTER TABLE inventory.room_inventory_ledger DROP CONSTRAINT IF EXISTS fk_inv_ledger_maint_block;
ALTER TABLE inventory.room_inventory_ledger DROP COLUMN IF EXISTS maintenance_block_id;

-- Restore original CHECK constraints
ALTER TABLE inventory.room_inventory_ledger
    ADD CHECK ((status = 'sold' AND reservation_id IS NOT NULL) OR (status <> 'sold'));
ALTER TABLE inventory.room_inventory_ledger
    ADD CHECK ((status = 'on_hold' AND checkout_session_id IS NOT NULL) OR (status <> 'on_hold'));

-- NOTE: enum value 'maintenance' cannot be removed cleanly; rollback past M3 requires manual intervention

-- M2 rollback
ALTER TABLE operations.reservation_items DROP COLUMN IF EXISTS version;

-- M1 rollback
DROP TRIGGER IF EXISTS trg_assign_reservation_code ON operations.reservations;
DROP TRIGGER IF EXISTS trg_assign_group_code ON operations.reservation_groups;
DROP FUNCTION IF EXISTS operations.fn_assign_reservation_code();
DROP FUNCTION IF EXISTS operations.fn_assign_group_code();

DROP INDEX IF EXISTS operations.idx_reservations_code_per_property;
DROP INDEX IF EXISTS operations.idx_reservation_groups_code_per_property;

ALTER TABLE operations.reservations DROP COLUMN IF EXISTS code;
ALTER TABLE operations.reservations DROP COLUMN IF EXISTS sequential;
ALTER TABLE operations.reservation_groups DROP COLUMN IF EXISTS code;
ALTER TABLE operations.reservation_groups DROP COLUMN IF EXISTS sequential;

ALTER TABLE operations.reservations ADD COLUMN sequential SERIAL NOT NULL;
ALTER TABLE operations.reservations
    ADD COLUMN code TEXT GENERATED ALWAYS AS ('RES-' || LPAD(sequential::TEXT, 6, '0')) STORED;

ALTER TABLE operations.reservation_groups ADD COLUMN sequential SERIAL NOT NULL;
ALTER TABLE operations.reservation_groups
    ADD COLUMN code TEXT GENERATED ALWAYS AS ('GRP-' || LPAD(sequential::TEXT, 5, '0')) STORED;

DROP TABLE IF EXISTS operations.reservation_sequences;
