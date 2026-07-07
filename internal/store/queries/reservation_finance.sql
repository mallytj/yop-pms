-- Finance folio queries

-- name: CreateFolio :one
INSERT INTO finance.folios (
    property_id, reservation_id, folio_part, balance_pence
) VALUES (
    @property_id, @reservation_id, @folio_part, 0
) RETURNING *;

-- name: ArchiveFolios :exec
UPDATE finance.folios
SET deleted_at = NOW()
WHERE reservation_id = @reservation_id
AND property_id = @property_id
AND deleted_at IS NULL;