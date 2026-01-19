-- name: ListRooms :many
SELECT * FROM rooms;

-- name: ListUsers :many
SELECT * FROM users;

-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, first_name, last_name, role, is_active) 
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUser :one
UPDATE users 
SET username = COALESCE($2, username),
    email = COALESCE($3, email),
    password_hash = COALESCE($4, password_hash),
    first_name = COALESCE($5, first_name),
    last_name = COALESCE($6, last_name),
    role = COALESCE($7, role),
    is_active = COALESCE($8, is_active)
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
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