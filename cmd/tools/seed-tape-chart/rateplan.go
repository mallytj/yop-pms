package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// seedRatePlan assumes clearPropertyData already wiped any prior rate plans
// for this property, so it always inserts a fresh row.
func seedRatePlan(ctx context.Context, conn *pgx.Conn, propertyID string) (string, error) {
	var ratePlanID string
	err := conn.QueryRow(ctx, `
		INSERT INTO pricing.rate_plans (property_id, name, code)
		VALUES ($1, 'Best Available Rate', 'BAR')
		RETURNING id
	`, propertyID).Scan(&ratePlanID)
	if err != nil {
		return "", fmt.Errorf("rate_plan: %w", err)
	}
	return ratePlanID, nil
}
