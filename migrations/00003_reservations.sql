-- +goose Up

CREATE TYPE operations.reservation_sequence_entity AS ENUM('reservation');
CREATE TABLE operations.reservation_sequences (
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    entity_type operations.reservation_sequence_entity NOT NULL DEFAULT 'reservation',
    next_value BIGINT NOT NULL DEFAULT 1 CHECK (next_value > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (property_id, entity_type)
);

ALTER TABLE operations.reservation_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.reservation_sequences FORCE ROW LEVEL SECURITY;
CREATE POLICY res_seq_isolation ON operations.reservation_sequences
    USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE TABLE operations.reservations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    primary_guest_id UUID NOT NULL REFERENCES identity.guests (id) ON DELETE RESTRICT,
    sequential BIGINT NOT NULL,
    code TEXT NOT NULL, 
    source operations.reservation_source NOT NULL DEFAULT 'internal',
    notes TEXT CHECK (char_length(notes) <= 2500),
    status operations.reservation_status NOT NULL DEFAULT 'hold',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    FOREIGN KEY (property_id, primary_guest_id) REFERENCES identity.guests (property_id, id),
    CONSTRAINT chk_res_expires_at CHECK (
        (status = 'hold' AND expires_at IS NOT NULL) OR 
        (status <> 'hold')
    ),
    UNIQUE (property_id, id)
);

ALTER TABLE operations.reservations ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations.reservations FORCE ROW LEVEL SECURITY;
CREATE POLICY reservations_isolation ON operations.reservations USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_reservations_property ON operations.reservations (property_id);
CREATE INDEX idx_reservations_primary_guest ON operations.reservations (primary_guest_id);

CREATE UNIQUE INDEX idx_reservations_code_lookup 
ON operations.reservations (property_id, code) 
WHERE (deleted_at IS NULL);

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
CREATE TRIGGER trg_assign_reservation_code
    BEFORE INSERT ON operations.reservations
    FOR EACH ROW EXECUTE FUNCTION operations.fn_assign_reservation_code();

CREATE TABLE operations.reservation_items (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
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
    do_not_move BOOLEAN NOT NULL DEFAULT FALSE,
    base_rate_pence INT NOT NULL DEFAULT 0 CHECK (base_rate_pence >= 0),
    adults_count INT NOT NULL DEFAULT 2 CHECK (adults_count >= 1),
    children_count INT NOT NULL DEFAULT 0,
    status operations.reservation_item_status NOT NULL DEFAULT 'booked',
    version INT NOT NULL DEFAULT 1,
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
ALTER TABLE operations.reservation_items FORCE ROW LEVEL SECURITY;
CREATE POLICY res_items_isolation ON operations.reservation_items USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_res_items_property ON operations.reservation_items (property_id);
CREATE INDEX idx_res_items_reservation ON operations.reservation_items (reservation_id);
CREATE INDEX idx_res_items_room_type ON operations.reservation_items (booked_room_type_id);
CREATE INDEX idx_res_items_assigned_room ON operations.reservation_items (assigned_room_id);
CREATE INDEX idx_res_items_rate_plan ON operations.reservation_items (rate_plan_id);
CREATE INDEX idx_res_items_guest ON operations.reservation_items (guest_id);

CREATE INDEX idx_res_items_arrival ON operations.reservation_items (property_id, lower(stay_period));
CREATE INDEX idx_res_items_departure ON operations.reservation_items (property_id, upper(stay_period));
CREATE INDEX idx_res_items_stay_overlap ON operations.reservation_items 
USING GIST (property_id, stay_period);

CREATE TABLE inventory.room_inventory_ledger (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_id UUID NOT NULL REFERENCES inventory.rooms (id) ON DELETE RESTRICT,
    reservation_id UUID REFERENCES operations.reservations (id) ON DELETE SET NULL,
    reservation_item_id UUID REFERENCES operations.reservation_items (id) ON DELETE SET NULL,
    maintenance_block_id UUID REFERENCES inventory.maintenance_blocks (id) ON DELETE SET NULL,
    calendar_date DATE NOT NULL,
    status inventory.inventory_status NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (room_id, calendar_date),
    FOREIGN KEY (property_id, room_id) REFERENCES inventory.rooms (property_id, id),
    FOREIGN KEY (property_id, reservation_id) REFERENCES operations.reservations (property_id, id),
    CONSTRAINT inv_ledger_status_consistency CHECK (
        (status = 'sold' AND reservation_id IS NOT NULL) OR
        (status = 'maintenance' AND maintenance_block_id IS NOT NULL) OR
        status IN ('available', 'decommissioned')
    )
);

ALTER TABLE inventory.room_inventory_ledger ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.room_inventory_ledger FORCE ROW LEVEL SECURITY;
CREATE POLICY inv_ledger_isolation ON inventory.room_inventory_ledger USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_inv_ledger_property ON inventory.room_inventory_ledger (property_id);
CREATE INDEX idx_inv_ledger_room ON inventory.room_inventory_ledger (room_id);
CREATE INDEX idx_inv_ledger_reservation ON inventory.room_inventory_ledger (reservation_id);
CREATE INDEX idx_inv_ledger_item ON inventory.room_inventory_ledger (reservation_item_id);
CREATE INDEX idx_inv_ledger_maintenance_block
    ON inventory.room_inventory_ledger (maintenance_block_id)
    WHERE (maintenance_block_id IS NOT NULL);
CREATE INDEX idx_inv_ledger_grid_view 
ON inventory.room_inventory_ledger (property_id, calendar_date, room_id)
INCLUDE (status, reservation_id);
CREATE INDEX idx_inv_ledger_availability_check
ON inventory.room_inventory_ledger (property_id, calendar_date)
WHERE (status IN ('sold', 'on_hold') AND deleted_at IS NULL);

-- +goose Down
DROP TRIGGER IF EXISTS trg_assign_reservation_code ON operations.reservations;
DROP FUNCTION IF EXISTS operations.fn_assign_reservation_code();

DROP TABLE inventory.room_inventory_ledger CASCADE;
DROP TABLE operations.reservation_items CASCADE;
DROP TABLE operations.reservations CASCADE;
