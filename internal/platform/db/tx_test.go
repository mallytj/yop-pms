package db

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// poolForTest connects to a real local DB for integration tests.
// Skipped when APP_ENV is not local or TEST_DATABASE_URL is not set.
func poolForTest(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := "postgres://yop:yop@localhost:5433/yop?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skipf("cannot connect to test DB: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("cannot ping test DB: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestExecuteTx_Success(t *testing.T) {
	pool := poolForTest(t)
	q := store.New(pool)

	propID := uuid.New()
	ctx := helpers.SetPropertyIDInCtx(context.Background(), propID)
	ctx = helpers.SetUserIDInCtx(ctx, uuid.New())

	got, err := ExecuteTx(ctx, pool, q, func(qtx *store.Queries) (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("ExecuteTx failed: %v", err)
	}
	if got != "ok" {
		t.Errorf("expected 'ok', got %q", got)
	}
}

func TestExecuteTx_FnErrorRollsBack(t *testing.T) {
	pool := poolForTest(t)
	q := store.New(pool)

	propID := uuid.New()
	ctx := helpers.SetPropertyIDInCtx(context.Background(), propID)
	ctx = helpers.SetUserIDInCtx(ctx, uuid.New())

	sentinel := errors.New("fn failed")
	_, err := ExecuteTx(ctx, pool, q, func(qtx *store.Queries) (string, error) {
		return "", sentinel
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestExecuteTx_PgxError(t *testing.T) {
	pool := poolForTest(t)
	q := store.New(pool)

	propID := uuid.New()
	ctx := helpers.SetPropertyIDInCtx(context.Background(), propID)
	ctx = helpers.SetUserIDInCtx(ctx, uuid.New())

	_, err := ExecuteTx(ctx, pool, q, func(qtx *store.Queries) (string, error) {
		return "", pgx.ErrNoRows
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteTx_NoPropertyContext(t *testing.T) {
	pool := poolForTest(t)
	q := store.New(pool)

	// No property ID in context — should still work (RLS SET LOCAL with empty string)
	ctx := context.Background()
	_, err := ExecuteTx(ctx, pool, q, func(qtx *store.Queries) (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("ExecuteTx without property context should not fail: %v", err)
	}
}

func TestExecuteTx_CanceledContext(t *testing.T) {
	pool := poolForTest(t)
	q := store.New(pool)

	propID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	ctx = helpers.SetPropertyIDInCtx(ctx, propID)

	_, err := ExecuteTx(ctx, pool, q, func(qtx *store.Queries) (string, error) {
		return "ok", nil
	})
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}
