-- Reservation worker queries
-- Background sweeps for hold expiry, no-show reminders, overstays, archival

-- name: FindExpiredHolds :many
SELECT
  *
FROM
  operations.reservations
WHERE
  status = 'hold'
  AND expires_at < NOW()
ORDER BY
  created_at ASC
LIMIT 100 FOR UPDATE SKIP LOCKED;

-- name: FindOverdueCheckins :many
SELECT
  ri.*
FROM
  operations.reservation_items ri
JOIN operations.property_settings ps
  ON ri.property_id = ps.property_id
WHERE
  ri.status = 'booked'
  AND LOWER(ri.stay_period) + (ps.no_show_grace_minutes || ' minutes') ::INTERVAL < NOW();

-- name: FindOverstays :many
SELECT
  ri.*
FROM
  operations.reservation_items ri
JOIN operations.property_settings ps
  ON ri.property_id = ps.property_id
WHERE
  ri.status = 'checked_in'
  AND NOW() > UPPER(ri.stay_period) + (ps.late_checkout_grace_minutes || ' minutes') ::INTERVAL
ORDER BY
  ri.updated_at ASC
LIMIT 100 FOR UPDATE SKIP LOCKED;

-- name: FindArchivableReservations :many
SELECT
  r.*
FROM
  operations.reservations r
JOIN operations.property_settings ps
  ON r.property_id = ps.property_id
WHERE
  r.status IN ('checked_out', 'cancelled')
  AND r.updated_at < NOW() - (ps.reservation_archive_after_days || ' days') ::INTERVAL
ORDER BY
  r.updated_at ASC
LIMIT 500 FOR UPDATE SKIP LOCKED;

-- Rollup applies the new status via service layer; version check prevents races
-- name: UpdateReservationStatus :exec
UPDATE operations.reservations
SET status = @status,
version = version + 1,
updated_at = NOW()
WHERE id = @id AND version = @version;
