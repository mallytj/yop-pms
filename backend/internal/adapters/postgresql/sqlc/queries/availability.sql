-- name: CountAvailableRoomsForRoomType :one
SELECT COUNT(DISTINCT ril.room_id)::int
FROM inventory.room_inventory_ledger ril
JOIN inventory.rooms r ON r.id = ril.room_id
WHERE r.room_type_id = $1
  AND ril.calendar_date >= $2
  AND ril.calendar_date < $3
  AND ril.status = 'available'
GROUP BY ril.room_id
HAVING COUNT(*) = ($3::date - $2::date);

-- name: ListAvailableRoomTypes :many
WITH fully_available_rooms AS (
  SELECT
    r.id AS room_id,
    r.room_type_id
  FROM inventory.rooms r
  JOIN inventory.room_inventory_ledger ril
    ON ril.room_id = r.id
  WHERE ril.calendar_date >= $1
    AND ril.calendar_date < $2
    AND ril.status = 'available'
  GROUP BY r.id, r.room_type_id
  HAVING COUNT(*) = ($2::date - $1::date)
)
SELECT
  rt.id AS room_type_id,
  rt.code,
  rt.name,
  COUNT(far.room_id)::int AS available_rooms
FROM fully_available_rooms far
JOIN inventory.room_types rt
  ON rt.id = far.room_type_id
GROUP BY rt.id, rt.code, rt.name
ORDER BY rt.code;


-- name: RoomTypeAvailabilityCalendar :many
SELECT
  ril.calendar_date,
  COUNT(*) FILTER (WHERE ril.status = 'available')::int AS available,
  COUNT(*) FILTER (WHERE ril.status = 'sold')::int AS sold,
  COUNT(*) FILTER (WHERE ril.status = 'maintenance')::int AS maintenance
FROM inventory.room_inventory_ledger ril
JOIN inventory.rooms r ON r.id = ril.room_id
WHERE r.room_type_id = $1
  AND ril.calendar_date BETWEEN $2 AND $3
GROUP BY ril.calendar_date
ORDER BY ril.calendar_date;

-- name: AssertAvailabilityForRoomType :one
SELECT EXISTS (
  SELECT 1
  FROM inventory.room_inventory_ledger ril
  JOIN inventory.rooms r ON r.id = ril.room_id
  WHERE r.room_type_id = $1
    AND ril.calendar_date >= $2
    AND ril.calendar_date < $3
    AND ril.status = 'available'
  GROUP BY ril.room_id
  HAVING COUNT(*) = ($3::date - $2::date)
) AS is_available;

-- name: AvailableRoomTypesWithPrice :many
-- param: ratePlanID UUID 
-- param: startDate DATE
-- param: endDate DATE
-- param: stayLength INT
WITH fully_available_rooms AS (
  SELECT
    r.id AS room_id,
    r.room_type_id
  FROM inventory.rooms r
  JOIN inventory.room_inventory_ledger ril
    ON ril.room_id = r.id
  WHERE ril.calendar_date >= @start_date::DATE
    AND ril.calendar_date < @end_date::DATE
    AND ril.status = 'available'
  GROUP BY r.id, r.room_type_id
  HAVING COUNT(*) = @stay_length::INT
),
stay_price AS (
  SELECT
    ppg.room_type_id,
    SUM(ppg.base_price_pence)::int AS total_price_pence
  FROM pricing.daily_price_grid ppg
  WHERE ppg.rate_plan_id = @rate_plan_id::UUID
    AND ppg.calendar_date >= @start_date::DATE
    AND ppg.calendar_date < @end_date::DATE
  GROUP BY ppg.room_type_id
)
SELECT
  rt.id AS room_type_id,
  rt.code,
  rt.name,
  sp.total_price_pence,
  COUNT(far.room_id)::int AS available_rooms
FROM fully_available_rooms far
JOIN inventory.room_types rt
  ON rt.id = far.room_type_id
JOIN stay_price sp
  ON sp.room_type_id = rt.id
GROUP BY
  rt.id,
  rt.code,
  rt.name,
  sp.total_price_pence
ORDER BY rt.code;


-- name: CanBookRoomType :one
-- params: room_type_id UUID, start_date DATE, end_date DATE, required_count INT
SELECT COUNT(*) >= $4 AS can_book
FROM (
  SELECT ril.room_id
  FROM inventory.room_inventory_ledger ril
  JOIN inventory.rooms r ON r.id = ril.room_id
  WHERE r.room_type_id = $1
    AND ril.calendar_date >= $2
    AND ril.calendar_date < $3
    AND ril.status = 'available'
  GROUP BY ril.room_id
  HAVING COUNT(*) = ($3::date - $2::date)
) sub;
