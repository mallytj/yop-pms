-- name: GetRatePlanCapacity :one
-- Single-date capacity lookup (0 = unlimited).
SELECT max_daily_capacity FROM pricing.base_rates 
WHERE rate_plan_id = @rate_plan_id 
  AND property_id = @property_id
  AND deleted_at IS NULL;

-- name: GetBaseRate :one
-- Lookup base nightly rate by day-of-week. No date overrides resolved.
SELECT base_price_pence FROM pricing.base_rates
WHERE property_id = @property_id
  AND room_type_id = @room_type_id
  AND rate_plan_id = @rate_plan_id
  AND day_of_week = @day_of_week
  AND deleted_at IS NULL;

