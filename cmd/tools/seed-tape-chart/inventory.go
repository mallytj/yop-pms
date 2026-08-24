package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func seedInventory(
	ctx context.Context,
	conn *pgx.Conn,
	propertyID string,
	roomIDs []string,
	roomTypeIDs []string,
	roomsPerType int,
	ledgerStartOffset int,
	ledgerDays int,
) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for dayOffset := range ledgerDays {
		date := today.AddDate(0, 0, ledgerStartOffset+dayOffset)
		for roomIndex, roomID := range roomIDs {
			typeIndex := roomIndex / roomsPerType
			_, err := conn.Exec(ctx, `
				INSERT INTO inventory.room_inventory_ledger (property_id, room_id, room_type_id, calendar_date, status)
				VALUES ($1, $2, $3, $4, 'available')
				ON CONFLICT (room_id, calendar_date) DO NOTHING
			`, propertyID, roomID, roomTypeIDs[typeIndex], date.Format("2006-01-02"))
			if err != nil {
				return fmt.Errorf("inventory %s %s: %w", roomID, date.Format("2006-01-02"), err)
			}
		}
	}
	return nil
}
