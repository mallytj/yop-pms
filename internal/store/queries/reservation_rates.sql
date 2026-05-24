-- Reservation rate queries
-- All price changes go through adjustment column + audit log
-- Base price is immutable after creation
-- See ADR-021 for 3-tiered capacity model

-- name: InsertBookedDailyRate :exec
INSERT INTO pricing.booked_daily_rates (
    property_id, reservation_item_id, calendar_date, rate_plan_id, base_price_pence
) VALUES (
    @property_id, @reservation_item_id, @calendar_date, @rate_plan_id, @base_price_pence
);

-- name: BulkInsertBookedDailyRates :exec
INSERT INTO pricing.booked_daily_rates (
    property_id, reservation_item_id, calendar_date, rate_plan_id, base_price_pence
) SELECT 
    unnest(@property_ids::uuid[]),
    unnest(@reservation_item_ids::uuid[]),
    unnest(@calendar_dates::date[]),
    unnest(@rate_plan_ids::uuid[])::uuid,
    unnest(@base_price_pences::int[]);

-- name: SoftDeleteBookedRatesNotInPeriod :exec
UPDATE pricing.booked_daily_rates
SET deleted_at = NOW()
WHERE reservation_item_id = @reservation_item_id
AND property_id = @property_id
AND deleted_at IS NULL
AND calendar_date NOT IN (SELECT unnest(@dates::date[]));

-- name: ApplyRateAdjustment :execresult
UPDATE pricing.booked_daily_rates
SET adjustment = jsonb_build_object('type', @type, 'value', @value, 'reason', @reason)
WHERE reservation_item_id = @reservation_item_id
AND property_id = @property_id
AND calendar_date = @calendar_date
RETURNING *;

-- name: ApproveRateAdjustments :exec
UPDATE pricing.booked_daily_rates
SET adjustment_approved = true, adjustment_approved_by_user_id = @user_id
WHERE reservation_item_id = @reservation_item_id
AND property_id = @property_id
AND calendar_date = ANY(@dates::date[]);

-- Check if a rate plan has capacity for the requested dates.
--
-- 3-tier inheritance: daily_price_grid (most specific) > seasonal_rates > base_rates.
-- COALESCE tries each tier in order; first non-NULL wins. NULL = unlimited.
-- Only dates WITH a non-NULL capacity are checked against BDRs (fail early for unlimited).
-- Returns only dates where the limit is met or exceeded.
--
-- See ADR-021 for the 3-tier capacity model.
-- name: CheckRatePlanCapacity :many
WITH limited_dates AS (
    SELECT d.calendar_date,
           COALESCE(
               (SELECT dp.daily_room_capacity FROM pricing.daily_price_grid dp 
                WHERE dp.rate_plan_id = @rate_plan_id AND dp.calendar_date = d.calendar_date AND dp.property_id = @property_id),
               (SELECT sr.max_daily_capacity FROM pricing.seasonal_rates sr 
                WHERE sr.rate_plan_id = @rate_plan_id AND d.calendar_date <@ sr.override_period AND sr.property_id = @property_id
                LIMIT 1),
               (SELECT br.max_daily_capacity FROM pricing.base_rates br 
                WHERE br.rate_plan_id = @rate_plan_id AND br.property_id = @property_id
                LIMIT 1)
           ) AS daily_room_capacity
    FROM unnest(@dates::date[]) AS d(calendar_date)
)
SELECT ld.calendar_date, 
       COUNT(bdr.id)::INT AS current_bookings,
       ld.daily_room_capacity
FROM limited_dates ld
LEFT JOIN pricing.booked_daily_rates bdr ON bdr.rate_plan_id = @rate_plan_id
    AND bdr.calendar_date = ld.calendar_date
    AND bdr.property_id = @property_id
    AND bdr.deleted_at IS NULL
WHERE ld.daily_room_capacity IS NOT NULL
GROUP BY ld.calendar_date, ld.daily_room_capacity
HAVING COUNT(bdr.id) >= ld.daily_room_capacity;

-- name: GetBookedRates :many
SELECT * FROM pricing.booked_daily_rates
WHERE reservation_item_id = @reservation_item_id
AND property_id = @property_id
AND deleted_at IS NULL
ORDER BY calendar_date;
