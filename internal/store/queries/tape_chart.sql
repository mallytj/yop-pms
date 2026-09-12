-- name: GetTapeChartRoomTypes :many
SELECT id, property_id, name, std_occupancy, min_occupancy, max_occupancy, created_at, updated_at
FROM inventory.room_types
WHERE property_id = @property_id
  AND deleted_at IS NULL
ORDER BY name;

-- name: GetTapeChartRooms :many
SELECT id, property_id, room_type_id, name, housekeeping_status, occupancy_status, created_at, updated_at
FROM inventory.rooms
WHERE property_id = @property_id
  AND deleted_at IS NULL
ORDER BY room_type_id, name;

-- name: GetTapeChartInventory :many
SELECT
    l.id,
    l.property_id,
    l.room_id,
    l.reservation_id,
    l.reservation_item_id,
    l.maintenance_block_id,
    l.calendar_date,
    l.status,
    r.name AS room_name,
    l.room_type_id
FROM inventory.room_inventory_ledger l
JOIN inventory.rooms r ON r.id = l.room_id
WHERE l.property_id = @property_id
  AND l.calendar_date >= sqlc.arg(from_date)::date
  AND l.calendar_date < sqlc.arg(to_date)::date
  AND l.deleted_at IS NULL
ORDER BY l.calendar_date, l.room_id;

-- name: GetTapeChartReservations :many
SELECT
    r.id,
    r.property_id,
    r.primary_guest_id,
    CONCAT_WS(' ', g.first_name, g.last_name) AS guest_name,
    r.sequential,
    r.code,
    r.source,
    r.status,
    LOWER(r.stay_period_envelope)::timestamptz AS check_in,
    UPPER(r.stay_period_envelope)::timestamptz AS check_out,
    r.notes,
    r.expires_at,
    r.version,
    r.created_at,
    r.updated_at
FROM operations.reservations r
LEFT JOIN identity.guests g ON g.id = r.primary_guest_id AND g.deleted_at IS NULL
WHERE r.property_id = @property_id
  AND LOWER(r.stay_period_envelope) < sqlc.arg(to_date)::timestamptz
  AND UPPER(r.stay_period_envelope) >= sqlc.arg(from_date)::timestamptz
  AND r.deleted_at IS NULL;

-- name: GetTapeChartReservationItems :many
SELECT
    ri.id,
    ri.reservation_id,
    ri.assigned_room_id,
    ri.booked_room_type_id,
    ri.guest_id,
    ri.rate_plan_id,
    ri.stay_period,
    ri.do_not_move,
    ri.base_rate_pence,
    ri.adults_count,
    ri.children_count,
    ri.status,
    ri.version,
    ri.created_at,
    ri.updated_at
FROM operations.reservation_items ri
WHERE ri.reservation_id = ANY(@reservation_ids::uuid[])
  AND ri.deleted_at IS NULL;

-- name: GetTapeChartMaintenanceBlocks :many
SELECT
    mb.id,
    mb.property_id,
    mb.room_id,
    LOWER(mb.block_period)::date AS start_date,
    UPPER(mb.block_period)::date AS end_date,
    mb.reason,
    mb.type,
    mb.created_at,
    mb.updated_at
FROM inventory.maintenance_blocks mb
WHERE mb.property_id = @property_id
  AND LOWER(mb.block_period)::date < sqlc.arg(to_date)::date
  AND UPPER(mb.block_period)::date >= sqlc.arg(from_date)::date
  AND mb.deleted_at IS NULL;
