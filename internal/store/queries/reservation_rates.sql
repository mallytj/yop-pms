-- name: GetRatePlanCapacity :one
-- Single-date capacity lookup (0 = unlimited).
SELECT max_daily_capacity FROM pricing.base_rates 
WHERE rate_plan_id = @rate_plan_id 
  AND property_id = @property_id
  AND deleted_at IS NULL;


