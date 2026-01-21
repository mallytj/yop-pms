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

-- name: GetLicenceByKey :one
SELECT * FROM licences WHERE licence_key = $1 LIMIT 1;

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

-- name: CreateProperty :one
INSERT INTO properties (address, name, timezone, licence_id, property_notes) 
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListPropertiesByLicenceID :many
SELECT * FROM properties WHERE licence_id = $1;

-- name: ListProperties :many
SELECT * FROM properties;

-- name: GetPropertyByID :one
SELECT * FROM properties WHERE id = $1;

-- name: UpdateProperty :one
UPDATE properties 
SET 
    name = COALESCE(sqlc.narg('name'), name),
    address = COALESCE(sqlc.narg('address'), address),
    timezone = COALESCE(sqlc.narg('timezone'), timezone),
    property_notes = COALESCE(sqlc.narg('property_notes'), property_notes),
    updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: DeleteProperty :execresult
DELETE FROM properties WHERE id = $1;

-- name: GetLicenceByPropertyID :one
SELECT l.*
FROM licences l
JOIN properties p ON l.id = p.licence_id
WHERE p.id = $1 LIMIT 1; 

-- name: GetPropertiesByLicenceID :many
SELECT * FROM properties WHERE licence_id = $1;

-- name: GetUsersByPropertyID :many
SELECT u.*
FROM users u
JOIN licences l ON u.licence_id = l.id
JOIN properties p ON l.id = p.licence_id
WHERE p.id = $1;

-- name: CreatePropertyAmenity :one
INSERT INTO property_amenities (property_id, name, short_code, description, is_active) 
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListPropertyAmenities :many
SELECT * FROM property_amenities;

-- name: GetPropertyAmenityByID :one
SELECT * FROM property_amenities WHERE id = $1;

-- name: UpdatePropertyAmenity :one
UPDATE property_amenities 
SET 
    name = COALESCE(sqlc.narg('name'), name),
    short_code = COALESCE(sqlc.narg('short_code'), short_code),
    description = COALESCE(sqlc.narg('description'), description),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = $1 RETURNING *;  

-- name: DeletePropertyAmenity :execresult
DELETE FROM property_amenities WHERE id = $1;

-- name: GetPropertyByPropertyAmenityID :one
SELECT p.*
FROM properties p
JOIN property_amenities pa ON p.id = pa.property_id
WHERE pa.id = $1 LIMIT 1;

-- name: GetLicenceByPropertyAmenityID :one
SELECT l.*
FROM licences l
JOIN properties p ON l.id = p.licence_id
JOIN property_amenities pa ON p.id = pa.property_id
WHERE pa.id = $1 LIMIT 1;