-- name: ListRooms :many
SELECT * FROM rooms;

-- name: ListUsers :many
SELECT * FROM users;

-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, first_name, last_name, is_active, licence_id, role) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUser :one
UPDATE users 
SET 
    username = COALESCE(sqlc.narg('username'), username),
    email = COALESCE(sqlc.narg('email'), email),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    licence_id = COALESCE(sqlc.narg('licence_id'), licence_id),
    first_name = COALESCE(sqlc.narg('first_name'), first_name),
    last_name = COALESCE(sqlc.narg('last_name'), last_name),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    role = COALESCE(sqlc.narg('role'), role),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :execresult
DELETE FROM users WHERE id = $1;

-- name: ListLicences :many
SELECT * FROM licences;

-- name: GetLicenceByUserID :one
SELECT l.*
FROM licences l
JOIN users u ON l.id = u.licence_id
WHERE u.id = $1 LIMIT 1;

-- name: GetLicenceByID :one
SELECT * FROM licences WHERE id = $1;

-- name: CreateLicence :one
INSERT INTO licences (licence_key, organisation_name, contact_email, licence_notes) 
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpdateLicence :one
UPDATE licences 
SET organisation_name = COALESCE($2, organisation_name),
    licence_key = COALESCE($3, licence_key),
    contact_email = COALESCE($4, contact_email),
    licence_notes = COALESCE($5, licence_notes),
    updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: DeleteLicence :execresult
DELETE FROM licences WHERE id = $1;

-- name: GetUsersByLicenceID :many
SELECT * FROM users WHERE licence_id = $1;

-- name: CheckLicenceExists :one
SELECT EXISTS(SELECT 1 FROM licences WHERE id = $1) AS exists;