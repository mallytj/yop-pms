-- +goose Up
-- +goose StatementBegin
ALTER TABLE pricing.daily_price_grid DROP COLUMN IF EXISTS is_available;
CREATE TABLE IF NOT EXISTS pricing.base_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_type_id UUID REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
    rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE RESTRICT,
    day_of_week INT NOT NULL CHECK (
        day_of_week >= 0
        AND day_of_week <= 6
    ),
    base_price_pence INTEGER NOT NULL DEFAULT 0,
    min_los_restriction INT CHECK (min_los_restriction > 0),
    max_los_restriction INT CHECK (
        max_los_restriction > 0
        AND max_los_restriction > min_los_restriction
    ),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    -- For soft deletes
    -- Ensure that the room type and rate plan belong to the same property
    FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id) ON DELETE RESTRICT,
    UNIQUE (property_id, id),
    UNIQUE (
        property_id,
        room_type_id,
        rate_plan_id,
        day_of_week
    )
);
CREATE TABLE IF NOT EXISTS pricing.seasonal_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    property_id UUID REFERENCES operations.properties (id) ON DELETE RESTRICT,
    room_type_id UUID REFERENCES inventory.room_types (id) ON DELETE RESTRICT,
    rate_plan_id UUID REFERENCES pricing.rate_plans (id) ON DELETE RESTRICT,
    override_period TSTZRANGE NOT NULL CHECK (
        lower(override_period) < upper(override_period)
        AND lower(override_period) IS NOT NULL
        AND upper(override_period) IS NOT NULL
    ),
    day_of_week INT NOT NULL CHECK (
        day_of_week >= 0
        AND day_of_week <= 6
    ),
    base_price_pence INTEGER NOT NULL DEFAULT 0,
    min_los_restriction INT CHECK (min_los_restriction > 0),
    max_los_restriction INT CHECK (
        max_los_restriction > 0
        AND max_los_restriction > min_los_restriction
    ),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    -- For soft deletes
    FOREIGN KEY (property_id, room_type_id) REFERENCES inventory.room_types (property_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (property_id, rate_plan_id) REFERENCES pricing.rate_plans (property_id, id) ON DELETE RESTRICT,
    UNIQUE (property_id, id),
    EXCLUDE USING GIST (
        property_id WITH =,
        room_type_id WITH =,
        rate_plan_id WITH =,
        day_of_week WITH =,
        override_period WITH &&
    )
    WHERE (deleted_at IS NULL)
);
CREATE OR REPLACE FUNCTION fn_calculate_res_item_price() RETURNS TRIGGER AS $$
DECLARE total INT;
BEGIN
SELECT SUM(final_price_pence)
FROM pricing.booked_daily_rates bdr
WHERE bdr.reservation_item_id = COALESCE(NEW.reservation_item_id, OLD.reservation_item_id) INTO total;
UPDATE operations.reservation_items
SET base_rate_pence = COALESCE(total, 0),
    updated_at = NOW()
WHERE id = COALESCE(NEW.reservation_item_id, OLD.reservation_item_id);
RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_bdr_change
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON pricing.booked_daily_rates FOR EACH ROW EXECUTE FUNCTION fn_calculate_res_item_price();
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_bdr_change ON pricing.booked_daily_rates;
DROP FUNCTION IF EXISTS fn_calculate_res_item_price();
DROP TABLE IF EXISTS pricing.seasonal_rates;
DROP TABLE IF EXISTS pricing.base_rates;
-- +goose StatementEnd