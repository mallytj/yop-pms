-- Reservation CRUD queries
-- See ADR-015 for state machine, ADR-020 for stay_period_envelope

-- name: CreateReservation :one
INSERT INTO operations.reservations (
    property_id, primary_guest_id, group_id, source, travel_agent_id, notes, status, version, stay_period_envelope, expires_at
) VALUES (
    @property_id, @primary_guest_id, @group_id, @source, @travel_agent_id, @notes, @status, 1, @stay_period_envelope, @expires_at
) RETURNING *;

-- name: CreateReservationItem :one
INSERT INTO operations.reservation_items (
    property_id, reservation_id, booked_room_type_id, assigned_room_id, guest_id, rate_plan_id, stay_period, base_rate_pence, adults_count, children_count, status, version, do_not_move
) VALUES (
    @property_id, @reservation_id, @booked_room_type_id, @assigned_room_id, @guest_id, @rate_plan_id, @stay_period, @base_rate_pence, @adults_count, @children_count, @status, 1, @do_not_move
) RETURNING *;

-- name: GetReservation :one
SELECT 
    r.*,
    COALESCE(
        json_agg(
            json_build_object(
                'id', i.id,
                'property_id', i.property_id,
                'reservation_id', i.reservation_id,
                'booked_room_type_id', i.booked_room_type_id,
                'assigned_room_id', i.assigned_room_id,
                'guest_id', i.guest_id,
                'rate_plan_id', i.rate_plan_id,
                'stay_period', i.stay_period,
                'base_rate_pence', i.base_rate_pence,
                'adults_count', i.adults_count,
                'children_count', i.children_count,
                'status', i.status,
                'version', i.version,
                'do_not_move', i.do_not_move,
                'created_at', i.created_at,
                'updated_at', i.updated_at,
                'deleted_at', i.deleted_at
            )
        ) FILTER (WHERE i.id IS NOT NULL), '[]'::json
    ) AS items
FROM operations.reservations r
LEFT JOIN operations.reservation_items i ON r.id = i.reservation_id AND i.deleted_at IS NULL
WHERE r.id = @id AND r.deleted_at IS NULL
GROUP BY r.id;

-- Cursor pagination per ADR-014
-- name: ListReservations :many
SELECT r.* 
FROM operations.reservations r
WHERE r.property_id = @property_id 
AND r.deleted_at IS NULL
AND (@status::operations.reservation_status IS NULL OR r.status = @status)
AND (@cursor_date::timestamptz IS NULL OR lower(r.stay_period_envelope) < @cursor_date
    OR (lower(r.stay_period_envelope) = @cursor_date AND r.id < @cursor_id))
AND (@start_date::date IS NULL OR lower(r.stay_period_envelope) >= @start_date::date)
AND (@end_date::date IS NULL OR upper(r.stay_period_envelope) <= @end_date::date)
ORDER BY lower(r.stay_period_envelope) DESC, r.id DESC
LIMIT sqlc.arg('limit');

-- name: UpdateReservationMetadata :one
UPDATE operations.reservations 
SET 
    notes = COALESCE(@notes, notes),
    travel_agent_id = COALESCE(@travel_agent_id, travel_agent_id),
    group_id = COALESCE(@group_id, group_id),
    primary_guest_id = COALESCE(@primary_guest_id, primary_guest_id),
    version = version + 1,
    updated_at = NOW()
WHERE id = @id AND version = @version
RETURNING *;

-- name: UpdateReservationItem :one
UPDATE operations.reservation_items 
SET 
    assigned_room_id = COALESCE(@assigned_room_id, assigned_room_id),
    stay_period = COALESCE(@stay_period, stay_period),
    rate_plan_id = COALESCE(@rate_plan_id, rate_plan_id),
    adults_count = COALESCE(@adults_count, adults_count),
    children_count = COALESCE(@children_count, children_count),
    status = COALESCE(@status, status),
    version = version + 1,
    updated_at = NOW()
WHERE id = @id AND version = @version
RETURNING *;
