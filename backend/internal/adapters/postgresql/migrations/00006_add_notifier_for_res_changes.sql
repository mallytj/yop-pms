-- +goose Up
-- +goose StatementBegin
ALTER TABLE operations.reservation_items DROP CONSTRAINT IF EXISTS reservation_items_stay_period_check;
ALTER TABLE operations.reservation_items
ADD CONSTRAINT reservation_items_stay_period_check CHECK (
    lower(stay_period) < upper(stay_period)
    AND lower(stay_period) IS NOT NULL
    AND upper(stay_period) IS NOT NULL
  );
ALTER TABLE operations.reservation_items
ADD COLUMN IF NOT EXISTS guest_id UUID REFERENCES identity.guests(id) ON DELETE
SET NULL;
-- Function to notify on reservation changes
CREATE OR REPLACE FUNCTION notify_reservation_change() RETURNS TRIGGER AS $$
DECLARE payload JSON;
BEGIN -- For UPDATE operations where dates changed, notify for BOTH old and new ranges
IF TG_OP = 'UPDATE'
AND (
  lower(OLD.stay_period) != lower(NEW.stay_period)
  OR upper(OLD.stay_period) != upper(NEW.stay_period)
) THEN -- Notify for OLD date range
payload := json_build_object(
  'operation',
  'UPDATE',
  'property_id',
  OLD.property_id::UUID,
  'record_id',
  OLD.id::text,
  'check_in_date',
  lower(OLD.stay_period),
  'check_out_date',
  upper(OLD.stay_period),
  'table',
  'reservations'
);
PERFORM pg_notify('reservation_changes', payload::text);
-- Notify for NEW date range
payload := json_build_object(
  'operation',
  'UPDATE',
  'property_id',
  NEW.property_id::UUID,
  'record_id',
  NEW.id::text,
  'check_in_date',
  lower(NEW.stay_period),
  'check_out_date',
  upper(NEW.stay_period),
  'table',
  'reservations'
);
PERFORM pg_notify('reservation_changes', payload::text);
RETURN NEW;
END IF;
-- For INSERT, DELETE, or UPDATE without date change
payload := json_build_object(
  'operation',
  TG_OP,
  'property_id',
  COALESCE(NEW.property_id, OLD.property_id)::UUID,
  'record_id',
  COALESCE(NEW.id, OLD.id)::text,
  'check_in_date',
  lower(COALESCE(NEW.stay_period, OLD.stay_period)),
  'check_out_date',
  upper(COALESCE(NEW.stay_period, OLD.stay_period)),
  'table',
  'reservations'
);
PERFORM pg_notify('reservation_changes', payload::text);
RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_reservation_change ON operations.reservation_items;
CREATE TRIGGER trg_reservation_change
AFTER
INSERT
  OR
UPDATE
  OR DELETE ON operations.reservation_items FOR EACH ROW EXECUTE FUNCTION notify_reservation_change();
CREATE OR REPLACE FUNCTION notify_guest_change() RETURNS TRIGGER AS $$
DECLARE rec RECORD;
BEGIN -- Get all reservations for the affected guest
FOR rec IN
SELECT r.id,
  r.property_id,
  r.stay_period
FROM operations.reservation_items r
WHERE r.guest_id = COALESCE(NEW.id, OLD.id) -- Use NEW.id for INSERT/UPDATE and OLD.id for DELETE
  LOOP -- Notify for each reservation of the guest
  PERFORM pg_notify(
    'reservation_changes',
    json_build_object(
      'operation',
      TG_OP,
      'property_id',
      rec.property_id::UUID,
      'record_id',
      rec.id::text,
      'check_in_date',
      lower(rec.stay_period),
      'check_out_date',
      upper(rec.stay_period),
      'table',
      'guests'
    )::text
  );
END LOOP;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_guest_update
AFTER
UPDATE ON identity.guests FOR EACH ROW
  WHEN (
    -- Only trigger on changes to first_name or last_name, as other changes to the guest record do not affect the planner data
    OLD.first_name IS DISTINCT
    FROM NEW.first_name
      OR OLD.last_name IS DISTINCT
    FROM NEW.last_name
  ) EXECUTE FUNCTION notify_guest_change();
CREATE TRIGGER trg_guest_delete
AFTER DELETE ON identity.guests FOR EACH ROW EXECUTE FUNCTION notify_guest_change();
-- Booked daily rates change
CREATE OR REPLACE FUNCTION notify_booked_daily_rate_change() RETURNS TRIGGER AS $$
DECLARE rec RECORD;
BEGIN
SELECT r.id,
  r.property_id,
  r.stay_period
FROM operations.reservation_items r
WHERE r.id = COALESCE(NEW.reservation_item_id, OLD.reservation_item_id) INTO rec;
PERFORM pg_notify(
  'reservation_changes',
  json_build_object(
    'operation',
    TG_OP,
    'property_id',
    rec.property_id::UUID,
    'record_id',
    rec.id::text,
    'check_in_date',
    lower(rec.stay_period),
    'check_out_date',
    upper(rec.stay_period),
    'table',
    'booked_daily_rates'
  )::text
);
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_booked_daily_rates_update_or_delete
AFTER
UPDATE
  OR DELETE ON pricing.booked_daily_rates FOR EACH ROW EXECUTE FUNCTION notify_booked_daily_rate_change();
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
ALTER TABLE operations.reservation_items DROP CONSTRAINT IF EXISTS reservation_items_stay_period_check;
DROP TRIGGER IF EXISTS trg_booked_daily_rates_update_or_delete ON pricing.booked_daily_rates;
DROP TRIGGER IF EXISTS trg_reservation_change ON operations.reservation_items;
DROP TRIGGER IF EXISTS trg_guest_update ON identity.guests;
DROP TRIGGER IF EXISTS trg_guest_delete ON identity.guests;
DROP FUNCTION IF EXISTS notify_reservation_change() CASCADE;
DROP FUNCTION IF EXISTS notify_booked_daily_rate_change() CASCADE;
DROP FUNCTION IF EXISTS notify_guest_change() CASCADE;
-- +goose StatementEnd