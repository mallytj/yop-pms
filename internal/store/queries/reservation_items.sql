-- Reservation items queries
-- See ADR-015 for rollup rule, ADR-013 for locking & availability

-- ADR-015 rollup: item status changes drive reservation status
-- Returns new_status TEXT (NULL = unchanged)
-- name: RollupReservationStatus :one
SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN 'cancelled'::operations.reservation_status
        WHEN COUNT(*) FILTER (WHERE status IN ('checked_out', 'no_show', 'cancelled', 'archived')) = COUNT(*) AND COUNT(*) FILTER (WHERE status = 'checked_out') > 0 THEN 'checked_out'::operations.reservation_status
        WHEN COUNT(*) FILTER (WHERE status = 'checked_in') > 0 AND COUNT(*) FILTER (WHERE status = 'booked') = 0 THEN 'checked_in'::operations.reservation_status
        ELSE NULL
    END::TEXT AS new_status
FROM operations.reservation_items
WHERE reservation_id = @reservation_id AND deleted_at IS NULL;

-- name: CountRoomsByType :one
SELECT COUNT(*)::INT AS cnt
FROM inventory.rooms
WHERE property_id = @property_id AND room_type_id = @room_type_id;

-- name: BlockedCountByType :many
SELECT ril.calendar_date, COUNT(DISTINCT ril.room_id)::INT AS blocked_count
FROM inventory.room_inventory_ledger ril
INNER JOIN inventory.rooms r ON r.id = ril.room_id
WHERE r.property_id = @property_id
  AND r.room_type_id = @room_type_id
  AND ril.status IN ('sold', 'on_hold', 'maintenance', 'decommissioned')
  AND ril.deleted_at IS NULL
  AND ril.calendar_date BETWEEN @start_date::date AND @end_date::date
GROUP BY ril.calendar_date
ORDER BY ril.calendar_date;

-- ADR-013: Auto-pin lowest available room of requested type
-- Uses LEFT JOIN with FOR UPDATE on the rooms side, not the outer join.
-- name: SelectRoomForAutoPin :one
SELECT r.id FROM inventory.rooms r
WHERE r.property_id = @property_id
AND r.room_type_id = @room_type_id
AND NOT EXISTS (
    SELECT 1 FROM inventory.room_inventory_ledger ril
    WHERE ril.room_id = r.id
    AND ril.calendar_date = ANY(@dates::date[])
    AND ril.status IN ('sold', 'on_hold', 'maintenance', 'decommissioned')
    AND ril.deleted_at IS NULL
)
ORDER BY r.name ASC
LIMIT 1 FOR UPDATE SKIP LOCKED;

-- Returns conflicting dates for precise error messaging
-- @exclude_item_id is nullable: NULL = check all items
-- name: ConflictCheckOnLedger :many
SELECT ril.calendar_date::date
FROM inventory.room_inventory_ledger ril
WHERE ril.room_id = @room_id
AND ril.calendar_date = ANY(@dates::date[])
AND ril.property_id = @property_id
AND ril.status IN ('sold', 'on_hold', 'maintenance', 'decommissioned')
AND ril.deleted_at IS NULL
AND (@exclude_item_id::uuid IS NULL OR ril.reservation_item_id != @exclude_item_id)
ORDER BY ril.calendar_date;

-- name: InsertLedgerRow :exec
INSERT INTO inventory.room_inventory_ledger (
    property_id, room_id, reservation_id, reservation_item_id, calendar_date, status
) VALUES (
    @property_id, @room_id, @reservation_id, @reservation_item_id, @calendar_date, @status
);

-- name: BulkInsertLedgerRows :exec
INSERT INTO inventory.room_inventory_ledger (
    property_id, room_id, reservation_id, reservation_item_id, calendar_date, status
) SELECT 
    unnest(@property_ids::uuid[]),
    unnest(@room_ids::uuid[]),
    unnest(@reservation_ids::uuid[]),
    unnest(@reservation_item_ids::uuid[]),
    unnest(@calendar_dates::date[]),
    unnest(@statuses::text[])::inventory.inventory_status;

-- name: UpdateLedgerRowRoom :exec
UPDATE inventory.room_inventory_ledger
SET room_id = @new_room_id
WHERE reservation_item_id = @reservation_item_id
AND property_id = @property_id;

-- name: DeleteLedgerRowsByItem :exec
DELETE FROM inventory.room_inventory_ledger
WHERE reservation_item_id = @reservation_item_id
AND property_id = @property_id;

-- name: DeleteLedgerRowsByItemFromDate :exec
DELETE FROM inventory.room_inventory_ledger
WHERE reservation_item_id = @reservation_item_id
AND property_id = @property_id
AND calendar_date >= @from_date::date;

-- Used by both workers and service layer
-- name: CancelReservationItems :exec
UPDATE operations.reservation_items
SET status = 'cancelled', deleted_at = NOW(), version = version + 1
WHERE reservation_id = @reservation_id
AND property_id = @property_id
AND status NOT IN ('checked_out', 'cancelled', 'archived')
AND version = @version;

-- name: DeleteLedgerForReservation :exec
DELETE FROM inventory.room_inventory_ledger
WHERE reservation_id = @reservation_id
AND property_id = @property_id;

-- name: GetReservationItems :many
SELECT * FROM operations.reservation_items
WHERE reservation_id = @reservation_id
AND property_id = @property_id
AND deleted_at IS NULL
ORDER BY created_at ASC;
