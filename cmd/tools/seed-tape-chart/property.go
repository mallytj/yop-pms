package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func ensureLicence(ctx context.Context, conn *pgx.Conn) (string, error) {
	var licenceID string
	err := conn.QueryRow(ctx, `
		SELECT id FROM operations.licences WHERE deleted_at IS NULL ORDER BY created_at LIMIT 1
	`).Scan(&licenceID)
	if err == nil {
		return licenceID, nil
	}

	err = conn.QueryRow(ctx, `
		INSERT INTO operations.licences (licence_key, organisation_name, contact_email)
		VALUES ('YOP-99999', 'Fenchurch House Hotel Group', 'test@example.com')
		RETURNING id
	`).Scan(&licenceID)
	if err != nil {
		return "", fmt.Errorf("licence: %w", err)
	}
	return licenceID, nil
}

func ensureProperty(ctx context.Context, conn *pgx.Conn, licenceID string) (string, error) {
	var propertyID string
	err := conn.QueryRow(ctx, `
		SELECT id FROM operations.properties WHERE name = 'Fenchurch House Hotel'
	`).Scan(&propertyID)
	if err == nil {
		return propertyID, nil
	}

	err = conn.QueryRow(ctx, `
		INSERT INTO operations.properties (licence_id, name, address, timezone)
		VALUES ($1, 'Fenchurch House Hotel', '14 Fenchurch Street, London', 'Europe/London')
		RETURNING id
	`, licenceID).Scan(&propertyID)
	if err != nil {
		return "", fmt.Errorf("property: %w", err)
	}
	return propertyID, nil
}
