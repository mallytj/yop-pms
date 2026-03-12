-- name: GetRate :one
SELECT COALESCE(
        dpg.base_price_pence,
        sr.base_price_pence,
        br.base_price_pence
    ) AS price_pence,
    COALESCE(
        dpg.min_los_restriction,
        sr.min_los_restriction,
        br.min_los_restriction,
        1
    ) AS min_los,
    COALESCE(
        dpg.max_los_restriction,
        sr.max_los_restriction,
        br.max_los_restriction,
        365
    ) AS max_los,
    CASE
        WHEN dpg.id IS NOT NULL THEN 'override'
        WHEN sr.id IS NOT NULL THEN 'seasonal'
        ELSE 'base'
    END::text AS source
FROM pricing.base_rates br
    LEFT JOIN pricing.seasonal_rates sr ON sr.property_id = br.property_id
    AND sr.room_type_id = br.room_type_id
    AND sr.rate_plan_id = br.rate_plan_id
    AND sr.day_of_week = EXTRACT(
        DOW
        FROM @calendar_date::date
    )::int
    AND sr.override_period @> @calendar_date::timestamptz
    AND sr.deleted_at IS NULL
    LEFT JOIN pricing.daily_price_grid dpg ON dpg.property_id = br.property_id
    AND dpg.room_type_id = br.room_type_id
    AND dpg.rate_plan_id = br.rate_plan_id
    AND dpg.calendar_date = @calendar_date
    AND dpg.deleted_at IS NULL
WHERE br.property_id = @property_id::uuid
    AND br.room_type_id = @room_type_id::uuid
    AND br.rate_plan_id = @rate_plan_id::uuid
    AND br.day_of_week = EXTRACT(
        DOW
        FROM @calendar_date::date
    )::int
    AND br.deleted_at IS NULL;

-- name: GetRatesForRange :many
SELECT d.date::date AS calendar_date,
    br.room_type_id,
    br.rate_plan_id,
    COALESCE(
        dpg.base_price_pence,
        sr.base_price_pence,
        br.base_price_pence
    ) AS price_pence,
    COALESCE(
        dpg.min_los_restriction,
        sr.min_los_restriction,
        br.min_los_restriction,
        1
    ) AS min_los,
    COALESCE(
        dpg.max_los_restriction,
        sr.max_los_restriction,
        br.max_los_restriction,
        365
    ) AS max_los,
    CASE
        WHEN dpg.id IS NOT NULL THEN 'override'
        WHEN sr.id IS NOT NULL THEN 'seasonal'
        ELSE 'base'
    END AS source
FROM generate_series(
        @start_date::date,
        @end_date::date,
        '1 day'::interval
    ) AS d(date)
    CROSS JOIN pricing.base_rates br
    LEFT JOIN pricing.seasonal_rates sr ON sr.property_id = br.property_id
    AND sr.room_type_id = br.room_type_id
    AND sr.rate_plan_id = br.rate_plan_id
    AND sr.day_of_week = EXTRACT(
        DOW
        FROM d.date
    )::int
    AND sr.override_period @> d.date::timestamptz
    AND sr.deleted_at IS NULL
    LEFT JOIN pricing.daily_price_grid dpg ON dpg.property_id = br.property_id
    AND dpg.room_type_id = br.room_type_id
    AND dpg.rate_plan_id = br.rate_plan_id
    AND dpg.calendar_date = d.date
    AND dpg.deleted_at IS NULL
WHERE br.property_id = @property_id::uuid
    AND br.day_of_week = EXTRACT(
        DOW
        FROM d.date
    )::int
    AND br.deleted_at IS NULL
ORDER BY br.room_type_id,
    br.rate_plan_id,
    d.date;