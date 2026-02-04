-- name: CreateReservationHold :one
INSERT INTO operations.reservations (
  property_id,
  primary_guest_id,
  source,
  status
)
VALUES ($1, $2, $3, 'hold')
RETURNING *;

-- name: CreateReservationItem :one
INSERT INTO operations.reservation_items (
  reservation_id,
  booked_room_type_id,
  stay_period,
  status
)
VALUES (
  $1,
  $2,
  tstzrange($3, $4, '[)'),
  'booked'
)
RETURNING *;

-- name: LockAvailableRoomsForStay :many
SELECT ril.id AS ledger_id,
       ril.room_id,
       ril.calendar_date
FROM inventory.room_inventory_ledger ril
JOIN inventory.rooms r ON r.id = ril.room_id
WHERE r.room_type_id = $1
  AND ril.calendar_date BETWEEN $2 AND $3
  AND ril.status = 'available'
FOR UPDATE SKIP LOCKED;

-- name: MarkInventorySold :exec
UPDATE inventory.room_inventory_ledger
SET status = 'sold',
    reservation_id = $2,
    updated_at = now()
WHERE id = $1;

-- name: AssignRoomToReservationItem :exec
UPDATE operations.reservation_items
SET assigned_room_id = $2,
    updated_at = now()
WHERE id = $1;

-- name: CreateCheckoutSession :one
INSERT INTO operations.checkout_sessions (
  property_id,
  reservation_id,
  payment_intent_id,
  expires_at
)
VALUES ($1, $2, $3, now() + interval '15 minutes')
RETURNING *;

-- name: ConfirmReservation :exec
UPDATE operations.reservations
SET status = 'confirmed',
    updated_at = now()
WHERE id = $1
  AND status = 'hold';

-- name: ReleaseInventoryForReservation :exec
UPDATE inventory.room_inventory_ledger
SET status = 'available',
    reservation_id = NULL,
    updated_at = now()
WHERE reservation_id = $1
  AND status = 'sold';

-- name: CancelReservationHold :exec
UPDATE operations.reservations
SET status = 'cancelled',
    updated_at = now()
WHERE id = $1
  AND status = 'hold';
