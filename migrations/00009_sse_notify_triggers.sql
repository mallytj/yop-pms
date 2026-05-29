-- +goose Up
-- +goose StatementBegin
--
-- Core Requirements: [R-RES-INTEG-003], [ADR-017]
--
-- Single function handles all reservation tables.
-- Payload: {table, op, id, property_id, version?} — client checks version
-- before deciding to refetch. Version included only on tables that have it
-- (reservations, reservation_items). Others skip the field.
CREATE OR REPLACE FUNCTION notify_reservation_changes() RETURNS TRIGGER AS $$
DECLARE
    payload JSONB;
BEGIN
    payload := jsonb_build_object(
        'table', TG_TABLE_NAME,
        'op', TG_OP,
        'id', COALESCE(NEW.id, OLD.id),
        'property_id', COALESCE(NEW.property_id, OLD.property_id)
    );

    -- Include version on tables with optimistic locking (reservations, items).
    -- Frontend checks local version against SSE version — skip refetch if match.
    IF TG_TABLE_NAME IN ('reservations', 'reservation_items') THEN
        payload := payload || jsonb_build_object('version', COALESCE(NEW.version, OLD.version));
    END IF;

    PERFORM pg_notify('reservation_changes', payload::TEXT);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_reservations_notify
    AFTER INSERT OR UPDATE OR DELETE ON operations.reservations
    FOR EACH ROW EXECUTE FUNCTION notify_reservation_changes();

CREATE TRIGGER trg_reservation_items_notify
    AFTER INSERT OR UPDATE OR DELETE ON operations.reservation_items
    FOR EACH ROW EXECUTE FUNCTION notify_reservation_changes();

CREATE TRIGGER trg_ledger_notify
    AFTER INSERT OR UPDATE OR DELETE ON inventory.room_inventory_ledger
    FOR EACH ROW EXECUTE FUNCTION notify_reservation_changes();

CREATE TRIGGER trg_booked_rates_notify
    AFTER INSERT OR UPDATE OR DELETE ON pricing.booked_daily_rates
    FOR EACH ROW EXECUTE FUNCTION notify_reservation_changes();
-- Guest update notifications are handled in 00005_planner_notifier.sql
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_booked_rates_notify ON pricing.booked_daily_rates;
DROP TRIGGER IF EXISTS trg_ledger_notify ON inventory.room_inventory_ledger;
DROP TRIGGER IF EXISTS trg_reservation_items_notify ON operations.reservation_items;
DROP TRIGGER IF EXISTS trg_reservations_notify ON operations.reservations;
DROP FUNCTION IF EXISTS notify_reservation_changes();
-- +goose StatementEnd
