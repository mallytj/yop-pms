-- name: UpdateReservationItem :one
UPDATE operations.reservation_items
SET assigned_room_id = COALESCE(
    sqlc.narg('assigned_room_id')::uuid,
    assigned_room_id
  ),
  booked_room_type_id = COALESCE(
    sqlc.narg('booked_room_type_id')::uuid,
    booked_room_type_id
  ),
  stay_period = COALESCE(
    tstzrange(
      sqlc.narg('check_in_date')::timestamptz,
      sqlc.narg('check_out_date')::timestamptz,
      '[)'
    ),
    stay_period
  ),
  status = COALESCE(
    sqlc.narg('status')::operations.reservation_item_status,
    status
  ),
  updated_at = NOW()
WHERE id = @reservation_item_id::uuid
  AND deleted_at IS NULL
RETURNING *;
-- name: CreateReservationItem :one
INSERT INTO operations.reservation_items (
    property_id,
    reservation_id,
    booked_room_type_id,
    assigned_room_id,
    rate_plan_id,
    stay_period,
    adults_count,
    children_count,
    status
  )
VALUES (
    @property_id,
    @reservation_id,
    @booked_room_type_id,
    @assigned_room_id,
    @rate_plan_id,
    tstzrange(
      @check_in::timestamptz,
      @check_out::timestamptz,
      '[)'
    ),
    @adults_count,
    @children_count,
    'booked'
  )
RETURNING *;
-- name: CreateBookedDailyRate :one
INSERT INTO pricing.booked_daily_rates (
    property_id,
    reservation_item_id,
    calendar_date,
    rate_plan_id,
    base_price_pence,
    adjustment
  )
VALUES (
    @property_id,
    @reservation_item_id,
    @calendar_date,
    @rate_plan_id,
    @base_price_pence,
    sqlc.narg(adjustment)
  )
RETURNING *;