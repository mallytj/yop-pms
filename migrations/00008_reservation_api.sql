-- +goose Up

-- ============================================================
-- Reservation API Schema Updates (M6-M19)
-- Based on internal/booking/PLAN.md Phase 0
-- ============================================================

-- M6: stay_period_envelope on reservations (ADR-020)
ALTER TABLE operations.reservations 
    ADD COLUMN IF NOT EXISTS stay_period_envelope TSTZRANGE;

CREATE INDEX IF NOT EXISTS idx_reservations_envelope_gist 
    ON operations.reservations USING GIST (property_id, stay_period_envelope);

CREATE INDEX idx_reservations_arrival_cursor 
    ON operations.reservations (property_id, lower(stay_period_envelope), id)
    WHERE (deleted_at IS NULL);

-- Backfill envelope from items (currently empty in dev, but good practice)
-- +goose StatementBegin
UPDATE operations.reservations r
SET stay_period_envelope = (
    SELECT tstzrange(min(lower(i.stay_period)), max(upper(i.stay_period)), '[]')
    FROM operations.reservation_items i
    WHERE i.reservation_id = r.id AND i.deleted_at IS NULL
);
-- +goose StatementEnd

ALTER TABLE operations.reservations ALTER COLUMN stay_period_envelope SET NOT NULL;

-- M7: Add 'overstay' to reservation_item_status
ALTER TYPE operations.reservation_item_status ADD VALUE IF NOT EXISTS 'overstay';

-- M10: Fix booked_daily_rates unique constraint for soft deletes
ALTER TABLE pricing.booked_daily_rates DROP CONSTRAINT IF EXISTS booked_daily_rates_reservation_item_id_calendar_date_key;
CREATE UNIQUE INDEX idx_booked_daily_rates_unique_active 
    ON pricing.booked_daily_rates (reservation_item_id, calendar_date)
    WHERE (deleted_at IS NULL);

-- M11: Add 'pending_cancellation' to reservation_status
ALTER TYPE operations.reservation_status ADD VALUE IF NOT EXISTS 'pending_cancellation';

-- M12: Capacity model (3-tier: daily_price_grid > seasonal_rates > base_rates)
-- Max daily capacity per rate plan per night
ALTER TABLE pricing.daily_price_grid 
    ADD COLUMN daily_room_capacity INT CHECK (daily_room_capacity > 0);
ALTER TABLE pricing.seasonal_rates 
    ADD COLUMN max_daily_capacity INT CHECK (max_daily_capacity > 0);
ALTER TABLE pricing.base_rates 
    ADD COLUMN max_daily_capacity INT CHECK (max_daily_capacity > 0);

-- M13: final_price_pence trigger for booked_daily_rates
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION pricing.fn_compute_final_price() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.adjustment IS NOT NULL THEN
        DECLARE
            adj_type TEXT;
            adj_value INT;
            delta NUMERIC;
        BEGIN
            adj_type := NEW.adjustment->>'type';
            adj_value := (NEW.adjustment->>'value')::INT;
            
            IF adj_type = 'percentage' THEN
                delta := ROUND(NEW.base_price_pence * adj_value / 100.0);
                NEW.final_price_pence := NEW.base_price_pence + delta::INT;
            ELSIF adj_type = 'fixed' THEN
                NEW.final_price_pence := NEW.base_price_pence + adj_value;
            ELSE
                NEW.final_price_pence := NEW.base_price_pence;
            END IF;

            -- Clamp to 0
            IF NEW.final_price_pence < 0 THEN
                NEW.final_price_pence := 0;
            END IF;
        END;
    ELSE
        NEW.final_price_pence := NEW.base_price_pence;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_compute_final_price ON pricing.booked_daily_rates;
CREATE TRIGGER trg_compute_final_price
    BEFORE INSERT OR UPDATE ON pricing.booked_daily_rates
    FOR EACH ROW EXECUTE FUNCTION pricing.fn_compute_final_price();

-- M14: do_not_move on reservation_items
ALTER TABLE operations.reservation_items 
    ADD COLUMN do_not_move BOOLEAN NOT NULL DEFAULT FALSE;

-- M15: late_checkout_grace_minutes on property_settings
ALTER TABLE operations.property_settings 
    ADD COLUMN late_checkout_grace_minutes INT NOT NULL DEFAULT 0 CHECK (late_checkout_grace_minutes >= 0);

-- M16: reservation_item_id on room_inventory_ledger
ALTER TABLE inventory.room_inventory_ledger
    ADD COLUMN IF NOT EXISTS reservation_item_id UUID;

ALTER TABLE inventory.room_inventory_ledger
    DROP CONSTRAINT IF EXISTS fk_inv_ledger_item;
ALTER TABLE inventory.room_inventory_ledger
    ADD CONSTRAINT fk_inv_ledger_item
    FOREIGN KEY (reservation_item_id) REFERENCES operations.reservation_items (id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_inv_ledger_item ON inventory.room_inventory_ledger (reservation_item_id);

-- M17: expires_at on reservations
ALTER TABLE operations.reservations 
    ADD COLUMN expires_at TIMESTAMPTZ;

-- Constraint: hold must have expires_at
ALTER TABLE operations.reservations 
    ADD CONSTRAINT chk_res_expires_at CHECK (
        (status = 'hold' AND expires_at IS NOT NULL) OR 
        (status <> 'hold')
    );

-- M18: Audit log triggers (ADR-021)
-- Specific per table for clarity; generalize if pattern repeats
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION auth.fn_audit_reservation_changes() RETURNS TRIGGER AS $$
DECLARE
    changes JSONB;
    user_id UUID;
    action auth.audit_log_action;
    entity auth.audit_log_entity;
BEGIN
    BEGIN
        user_id := current_setting('app.current_user_id')::uuid;
    EXCEPTION WHEN OTHERS THEN
        user_id := NULL;
    END;

    IF TG_OP = 'INSERT' THEN
        action := 'create';
        changes := to_jsonb(NEW.*);
    ELSIF TG_OP = 'UPDATE' THEN
        action := 'update';
        changes := jsonb_build_object('old', to_jsonb(OLD), 'new', to_jsonb(NEW));
    ELSIF TG_OP = 'DELETE' THEN
        action := 'delete';
        changes := to_jsonb(OLD.*);
    END IF;

    IF TG_TABLE_NAME = 'reservations' THEN
        entity := 'reservation';
        INSERT INTO auth.audit_logs (action, entity, entity_id, property_id, changes, user_id)
        VALUES (action, entity, NEW.id, NEW.property_id, changes, user_id);
    ELSIF TG_TABLE_NAME = 'reservation_items' THEN
        entity := 'reservation_item';
        INSERT INTO auth.audit_logs (action, entity, entity_id, property_id, changes, user_id)
        VALUES (action, entity, NEW.id, NEW.property_id, changes, user_id);
    ELSIF TG_TABLE_NAME = 'booked_daily_rates' THEN
        entity := 'booked_daily_rate';
        INSERT INTO auth.audit_logs (action, entity, entity_id, property_id, changes, user_id)
        VALUES (action, entity, NEW.id, NEW.property_id, changes, user_id);
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_audit_reservations ON operations.reservations;
CREATE TRIGGER trg_audit_reservations
    AFTER INSERT OR UPDATE OR DELETE ON operations.reservations
    FOR EACH ROW EXECUTE FUNCTION auth.fn_audit_reservation_changes();

DROP TRIGGER IF EXISTS trg_audit_reservation_items ON operations.reservation_items;
CREATE TRIGGER trg_audit_reservation_items
    AFTER INSERT OR UPDATE OR DELETE ON operations.reservation_items
    FOR EACH ROW EXECUTE FUNCTION auth.fn_audit_reservation_changes();

DROP TRIGGER IF EXISTS trg_audit_booked_daily_rates ON pricing.booked_daily_rates;
CREATE TRIGGER trg_audit_booked_daily_rates
    AFTER INSERT OR UPDATE OR DELETE ON pricing.booked_daily_rates
    FOR EACH ROW EXECUTE FUNCTION auth.fn_audit_reservation_changes();

-- M19: cancellation_intent on reservations
ALTER TABLE operations.reservations 
    ADD COLUMN cancellation_intent JSONB;

-- +goose Down

DROP TRIGGER IF EXISTS trg_audit_booked_daily_rates ON pricing.booked_daily_rates;
DROP TRIGGER IF EXISTS trg_audit_reservation_items ON operations.reservation_items;
DROP TRIGGER IF EXISTS trg_audit_reservations ON operations.reservations;
DROP FUNCTION IF EXISTS auth.fn_audit_reservation_changes();

ALTER TABLE operations.reservations DROP COLUMN IF EXISTS cancellation_intent;
ALTER TABLE operations.reservations DROP COLUMN IF EXISTS expires_at;
ALTER TABLE operations.reservations DROP CONSTRAINT IF EXISTS chk_res_expires_at;

ALTER TABLE inventory.room_inventory_ledger DROP CONSTRAINT IF EXISTS fk_inv_ledger_item;
ALTER TABLE inventory.room_inventory_ledger DROP COLUMN IF EXISTS reservation_item_id;
DROP INDEX IF EXISTS inventory.idx_inv_ledger_item;

ALTER TABLE operations.property_settings DROP COLUMN IF EXISTS late_checkout_grace_minutes;
ALTER TABLE operations.reservation_items DROP COLUMN IF EXISTS do_not_move;

DROP TRIGGER IF EXISTS trg_compute_final_price ON pricing.booked_daily_rates;
DROP FUNCTION IF EXISTS pricing.fn_compute_final_price();

ALTER TABLE pricing.daily_price_grid DROP COLUMN IF EXISTS daily_room_capacity;
ALTER TABLE pricing.seasonal_rates DROP COLUMN IF EXISTS max_daily_capacity;
ALTER TABLE pricing.base_rates DROP COLUMN IF EXISTS max_daily_capacity;

-- Note: Enum values cannot be dropped in Postgres. 
-- To fully revert M7/M11, a new type would need to be created and swapped.
-- We leave the extra enum values as they are harmless.

DROP INDEX IF EXISTS operations.idx_reservations_arrival_cursor;
DROP INDEX IF EXISTS operations.idx_reservations_envelope_gist;
ALTER TABLE operations.reservations DROP COLUMN IF EXISTS stay_period_envelope;

DROP INDEX IF EXISTS pricing.idx_booked_daily_rates_unique_active;
CREATE UNIQUE INDEX IF NOT EXISTS booked_daily_rates_reservation_item_id_calendar_date_key 
    ON pricing.booked_daily_rates (reservation_item_id, calendar_date);
