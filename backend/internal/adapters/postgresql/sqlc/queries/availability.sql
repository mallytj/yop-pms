-- name: GetRoomsForPlanner :many
SELECT r.id as room_id,
  r.name as room_name,
  rt.code as room_type_code,
  rt.id as room_type_id
FROM inventory.rooms r
  INNER JOIN inventory.room_types rt ON r.room_type_id = rt.id
WHERE r.deleted_at IS NULL
ORDER BY rt.name,
  r.name;
-- name: GetReservationItemsForPlanner :many
SELECT ri.id as reservation_item_id,
  ri.reservation_id as reservation_id,
  ri.assigned_room_id,
  ri.booked_room_type_id,
  ri.stay_period,
  ri.base_rate_pence as stay_price_pence,
  ri.status as item_status,
  res.status as reservation_status,
  res.code as reservation_code,
  g.id as guest_id,
  g.first_name as guest_first_name,
  g.last_name as guest_last_name
FROM operations.reservation_items ri
  INNER JOIN operations.reservations res ON ri.reservation_id = res.id
  LEFT JOIN identity.guests g ON res.primary_guest_id = g.id
WHERE ri.deleted_at IS NULL
  AND res.deleted_at IS NULL
  AND ri.status NOT IN ('cancelled', 'archived')
  AND res.status NOT IN ('cancelled', 'archived')
  AND ri.stay_period && tstzrange(
    @start_date::timestamptz,
    @end_date::timestamptz,
    '[)'
  )
ORDER BY lower(ri.stay_period);