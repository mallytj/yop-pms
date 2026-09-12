package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// seedRatePlan upserts on the (property_id, code) partial unique index so a
// run still succeeds if clearPropertyData failed or was skipped — the stale
// BAR row is reused instead of violating the constraint.
func seedRatePlan(ctx context.Context, conn *pgx.Conn, propertyID string) (string, error) {
	var ratePlanID string
	err := conn.QueryRow(ctx, `
		INSERT INTO pricing.rate_plans (property_id, name, code)
		VALUES ($1, 'Best Available Rate', 'BAR')
		ON CONFLICT (property_id, code) WHERE deleted_at IS NULL
		DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, propertyID).Scan(&ratePlanID)
	if err != nil {
		return "", fmt.Errorf("rate_plan: %w", err)
	}
	return ratePlanID, nil
}
