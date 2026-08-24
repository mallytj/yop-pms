package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// clearPropertyData wipes every fixture this tool seeds for propertyID, in
// FK-safe order, so each run starts from a clean slate regardless of what a
// prior run (possibly with a different -xl flag) left behind. The licence
// and property rows themselves are left in place so VITE_DEV_PROPERTY_ID
// stays stable across reseeds.
func clearPropertyData(ctx context.Context, conn *pgx.Conn, propertyID string) error {
	statements := []struct {
		label string
		sql   string
	}{
		{"inventory ledger", `DELETE FROM inventory.room_inventory_ledger WHERE property_id = $1`},
		{"maintenance blocks", `DELETE FROM inventory.maintenance_blocks WHERE property_id = $1`},
		{"reservation items", `DELETE FROM operations.reservation_items WHERE property_id = $1`},
		{"reservations", `DELETE FROM operations.reservations WHERE property_id = $1`},
		{"rooms", `DELETE FROM inventory.rooms WHERE property_id = $1`},
		{"room types", `DELETE FROM inventory.room_types WHERE property_id = $1`},
		{"rate plans", `DELETE FROM pricing.rate_plans WHERE property_id = $1`},
		{"guests", `DELETE FROM identity.guests WHERE property_id = $1`},
	}

	for _, stmt := range statements {
		if _, err := conn.Exec(ctx, stmt.sql, propertyID); err != nil {
			return fmt.Errorf("clear %s: %w", stmt.label, err)
		}
	}
	return nil
}
