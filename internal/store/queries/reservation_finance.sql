-- Finance folio queries

-- name: CreateFolio :one
INSERT INTO finance.folios (
    property_id, reservation_id, folio_part, balance_pence
) VALUES (
    @property_id, @reservation_id, @folio_part, 0
) RETURNING *;