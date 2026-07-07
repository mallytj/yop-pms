-- Guest queries for inline guest creation and expansion

-- name: CreateGuest :one
INSERT INTO identity.guests (
    property_id, first_name, last_name, email, phone_number
) VALUES (
    @property_id, @first_name, @last_name, @email, @phone_number
) RETURNING *;

-- name: GetGuest :one
SELECT * FROM identity.guests WHERE id = @id AND deleted_at IS NULL;