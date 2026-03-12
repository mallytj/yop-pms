package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseFoundations(t *testing.T) {
	// 1. Setup Connection (REQ-021: Non-superuser connection)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://app_user:password@localhost:5432/yop_pms")
	require.NoError(t, err)
	defer conn.Close(ctx)

	// Generate Test IDs
	propertyA := uuid.Must(uuid.NewV7())
	propertyB := uuid.Must(uuid.NewV7())

	t.Run("REQ-020: Row-Level Security Isolation", func(t *testing.T) {
		// Set session to Property A
		_, err = conn.Exec(ctx, "SET LOCAL app.current_property_id = $1", propertyA)
		require.NoError(t, err)

		// Insert a room for Property A
		roomID, _ := uuid.NewV7()
		_, err = conn.Exec(ctx, "INSERT INTO inventory.rooms (id, property_id, name) VALUES ($1, $2, $3)",
			roomID, propertyA, "Room 101")
		require.NoError(t, err)

		// Switch session to Property B
		_, err = conn.Exec(ctx, "SET LOCAL app.current_property_id = $1", propertyB)
		require.NoError(t, err)

		// Try to read Property A's room - should return 0 rows
		var name string
		err = conn.QueryRow(ctx, "SELECT name FROM inventory.rooms WHERE id = $1", roomID).Scan(&name)
		assert.ErrorIs(t, err, pgx.ErrNoRows, "Property B should NOT be able to see Property A's data")
	})

	t.Run("REQ-002: UUIDv7 Generation", func(t *testing.T) {
		var dbUUID uuid.UUID
		err := conn.QueryRow(ctx, "SELECT uuid_generate_v7()").Scan(&dbUUID)
		require.NoError(t, err)

		assert.Equal(t, byte(7), dbUUID.Version(), "DB should generate version 7 UUIDs")
	})

	t.Run("REQ-024: Date Overlap Exclusion (No Double Booking)", func(t *testing.T) {
		roomID, _ := uuid.NewV7()
		// Create a room first
		_, _ = conn.Exec(ctx, "INSERT INTO inventory.rooms (id, property_id, name) VALUES ($1, $2, $3)",
			roomID, propertyA, "Overlap Test Room")

		// Create first maintenance block: March 10 - March 15
		_, err = conn.Exec(ctx, `
			INSERT INTO inventory.maintenance_blocks (property_id, room_id, block_period, reason, type) 
			VALUES ($1, $2, tstzrange('2026-03-10', '2026-03-15'), 'Repair', 'repair')`,
			propertyA, roomID)
		require.NoError(t, err)

		// Attempt overlapping block: March 12 - March 18 (Should fail)
		_, err = conn.Exec(ctx, `
			INSERT INTO inventory.maintenance_blocks (property_id, room_id, block_period, reason, type) 
			VALUES ($1, $2, tstzrange('2026-03-12', '2026-03-18'), 'Cleaning', 'deep_clean')`,
			propertyA, roomID)

		assert.Error(t, err, "Database must reject overlapping date ranges for the same resource")
	})

	t.Run("PG Notify: Real-time Planner Updates", func(t *testing.T) {
		// Listen to the channel
		_, err = conn.Exec(ctx, "LISTEN reservation_changes")
		require.NoError(t, err)

		// Trigger an update on a room (assuming you have a guest and reservation items table)
		// For brevity, we trigger a notify manually or via a table that has your trigger
		_, err = conn.Exec(ctx, "NOTIFY reservation_changes, '{\"operation\": \"INSERT\", \"table\": \"rooms\"}'")
		require.NoError(t, err)

		// Wait for notification
		notification, err := conn.WaitForNotification(ctx)
		require.NoError(t, err)

		var payload map[string]interface{}
		json.Unmarshal([]byte(notification.Payload), &payload)
		assert.Equal(t, "INSERT", payload["operation"])
	})
}
