-- Cross-cutting queries used by ExecuteTx and notification infrastructure.

-- name: SetCurrentPropertyID :exec
SELECT set_config('app.current_property_id', @property_id::text, true);

-- name: NotifyChannel :exec
SELECT pg_notify(@channel::text, @payload::text);

-- name: GetPropertyTimezone :one
SELECT timezone FROM operations.properties
WHERE id = @id AND deleted_at IS NULL;
