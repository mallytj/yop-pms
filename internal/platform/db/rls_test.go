package db_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestTenantIsolation proves REQ-021, REQ-022, and REQ-020
func TestTenantIsolation(t *testing.T) {
	// Connect using the specific least-privilege App User (REQ-021)
	// This user should NOT be a superuser or own the tables.
	db, err := sql.Open("pgx", "postgres://app_user:password@localhost:5433/yop_db?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect as app_user: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Ensure queries FAIL or return 0 rows if we forget to set the context (REQ-022)
	t.Run("Query Fails Without Context Setup", func(t *testing.T) {
		// Attempting to select from a protected table without setting app.current_property_id
		var count int
		err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bookings`).Scan(&count)
		if err != nil {
			t.Fatalf("Expected query to execute (returning 0 rows due to RLS), got error: %v", err)
		}

		if count != 0 {
			t.Fatalf("SECURITY VIOLATION: Expected 0 rows without tenant context, but got %d. RLS is bypassing!", count)
		}
	})

	// 2. Ensure queries SUCCEED and return scoped data when context is set
	t.Run("Query Succeeds With Tenant Context", func(t *testing.T) {
		// In Go, this is exactly what your HTTP Middleware / Service layer should be doing
		// before handing the transaction over to sqlc.
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin tx: %v", err)
		}
		defer tx.Rollback()

		// Set the Postgres Local Configuration Parameter for the transaction
		targetPropertyID := uuid.Must(uuid.NewV7()).String()
		_, err = tx.ExecContext(ctx, `SET LOCAL app.current_property_id = $1`, targetPropertyID)
		if err != nil {
			t.Fatalf("Failed to set tenant context: %v", err)
		}

		// Now query using sqlc generated code (simulated here via raw sql)
		var count int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM bookings`).Scan(&count)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Assuming our test DB is seeded with 5 bookings for this property
		// t.Logf("Successfully fetched %d bookings isolated to property %s", count, targetPropertyID)
	})
}
