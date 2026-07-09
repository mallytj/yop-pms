-- +goose Up
CREATE TABLE inventory.room_types (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 75),
    code CITEXT NOT NULL CHECK (char_length(code) <= 10),
    std_occupancy INT NOT NULL DEFAULT 2 CHECK (std_occupancy > 0),
    min_occupancy INT NOT NULL DEFAULT 1,
    max_occupancy INT NOT NULL DEFAULT 2,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_occupancy_range 
      CHECK (min_occupancy > 0 AND min_occupancy <= std_occupancy AND max_occupancy >= std_occupancy),
    UNIQUE (property_id, id)
);

ALTER TABLE inventory.room_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory.room_types FORCE ROW LEVEL SECURITY;
CREATE POLICY room_types_isolation ON inventory.room_types 
  USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE UNIQUE INDEX idx_room_types_code_act ON inventory.room_types (property_id, code) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX idx_room_types_name_act ON inventory.room_types (property_id, name) WHERE (deleted_at IS NULL);
CREATE INDEX idx_room_types_property ON inventory.room_types (property_id) WHERE (deleted_at IS NULL);

CREATE TABLE inventory.rooms (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
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
ALTER TABLE inventory.rooms FORCE ROW LEVEL SECURITY;
CREATE POLICY rooms_isolation ON inventory.rooms USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_rooms_property ON inventory.rooms (property_id) WHERE (deleted_at IS NULL);
CREATE INDEX idx_rooms_room_type ON inventory.rooms (room_type_id);
CREATE UNIQUE INDEX idx_active_room_unique_name ON inventory.rooms (property_id, name) WHERE (deleted_at IS NULL);

CREATE TABLE inventory.maintenance_blocks (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
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
ALTER TABLE inventory.maintenance_blocks FORCE ROW LEVEL SECURITY;
CREATE POLICY maint_isolation ON inventory.maintenance_blocks USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_maint_blocks_property ON inventory.maintenance_blocks (property_id) WHERE (deleted_at IS NULL);
CREATE INDEX idx_maint_blocks_period ON inventory.maintenance_blocks USING GIST (property_id, block_period) WHERE (deleted_at IS NULL);
CREATE INDEX idx_maint_blocks_room ON inventory.maintenance_blocks (room_id);
CREATE INDEX idx_maint_blocks_created_by ON inventory.maintenance_blocks (created_by_user_id);


CREATE TABLE pricing.rate_plans (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (char_length(name) <= 50),
    code CITEXT NOT NULL CHECK (char_length(code) <= 10),
    description TEXT CHECK (char_length(description) <= 300),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    currency_code CITEXT NOT NULL DEFAULT 'GBP' CHECK (char_length(currency_code) = 3),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (property_id, id)
);

ALTER TABLE pricing.rate_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.rate_plans FORCE ROW LEVEL SECURITY;
CREATE POLICY rate_plans_isolation ON pricing.rate_plans USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE UNIQUE INDEX idx_active_rate_plan_unique_name ON pricing.rate_plans (property_id, code) WHERE (deleted_at IS NULL);
CREATE INDEX idx_rate_plans_property ON pricing.rate_plans (property_id) WHERE (deleted_at IS NULL);
CREATE INDEX idx_active_rate_plans_property ON pricing.rate_plans (property_id) WHERE (deleted_at IS NULL AND is_active = FALSE);

CREATE TABLE pricing.base_rates (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    property_id UUID NOT NULL REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_type_id UUID NOT NULL REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
    rate_plan_id UUID NOT NULL REFERENCES pricing.rate_plans (id) ON DELETE RESTRICT,
    day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    base_price_pence INTEGER NOT NULL DEFAULT 0,
    min_los_restriction INT CHECK (min_los_restriction > 0),
    max_los_restriction INT CHECK (max_los_restriction > min_los_restriction),
    max_daily_capacity INT CHECK (max_daily_capacity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    UNIQUE (property_id, id),
    UNIQUE (property_id, room_type_id, rate_plan_id, day_of_week),
    FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id),
    FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id)
);

ALTER TABLE pricing.base_rates ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.base_rates FORCE ROW LEVEL SECURITY;
CREATE POLICY base_rates_isolation ON pricing.base_rates USING (property_id = current_setting('app.current_property_id')::uuid);

CREATE INDEX idx_base_rates_property ON pricing.base_rates (property_id) WHERE (deleted_at IS NULL);
CREATE INDEX idx_base_rates_room_type ON pricing.base_rates (room_type_id);
CREATE INDEX idx_base_rates_rate_plan ON pricing.base_rates (rate_plan_id);
CREATE INDEX idx_base_rates_lookup
  ON pricing.base_rates (property_id, rate_plan_id, room_type_id, day_of_week) 
  INCLUDE (base_price_pence, min_los_restriction, max_los_restriction)
  WHERE (deleted_at IS NULL);

-- +goose Down
DROP TABLE pricing.base_rates CASCADE;
DROP TABLE pricing.rate_plans CASCADE;
DROP TABLE inventory.maintenance_blocks CASCADE;
DROP TABLE inventory.rooms CASCADE;
DROP TABLE inventory.room_types CASCADE;
