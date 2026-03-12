-- name: SetCurrentPropertyID :exec
SELECT set_config('app.current_property_id', @property_id::text, true);