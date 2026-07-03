-- Property settings lookup

-- name: GetPropertySettings :one
SELECT website_hold_ttl_seconds, internal_hold_ttl_seconds
FROM operations.property_settings
WHERE property_id = $1;
