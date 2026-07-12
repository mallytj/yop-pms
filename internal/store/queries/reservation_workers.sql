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
  *
FROM
  operations.reservation_items
WHERE
  status = 'booked'
  AND LOWER(stay_period) + '10 minutes'::INTERVAL < NOW();

-- name: FindOverstays :many
SELECT
  *
FROM
  operations.reservation_items
WHERE
  status = 'checked_in'
  AND NOW() > UPPER(stay_period) + '100 minutes'::INTERVAL
ORDER BY
  updated_at ASC
LIMIT 100 FOR UPDATE SKIP LOCKED;

-- name: FindArchivableReservations :many
SELECT
  *
FROM
  operations.reservations
WHERE
  status IN ('checked_out', 'cancelled')
  AND updated_at < NOW() - '365 days'::INTERVAL
ORDER BY
  updated_at ASC
LIMIT 500 FOR UPDATE SKIP LOCKED;

-- Rollup applies the new status via service layer; version check prevents races
-- name: UpdateReservationStatus :execrows
UPDATE operations.reservations
SET status = @status,
  version = version + 1,
  updated_at = NOW()
WHERE id = @id AND version = @version;
