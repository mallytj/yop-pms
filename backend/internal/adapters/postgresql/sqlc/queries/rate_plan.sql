-- name: GetRatePlans :many
SELECT id,
    parent_rate_plan_id,
    name,
    code,
    description,
    currency_code
FROM pricing.rate_plans
WHERE deleted_at IS NULL
    AND is_active = true;