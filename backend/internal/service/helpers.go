package service

import (
	"context"
	"fmt"
	hf "ollerod-pms/internal/helpers"

	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	"github.com/google/uuid"
)

// ExecuteTx handles the heavy lifting of RLS and Tx lifecycle
func ExecuteTx[T any](s *Svc, ctx context.Context, fn func(*repo.Queries) (T, error)) (T, error) {
	var result T

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("starting tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create a transaction-bound repository
	qtx := s.repo.WithTx(tx)

	// Enforce RLS: Get Tenant/Property ID from Context
	// This ensures every query in 'fn' is scoped
	propertyID := hf.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return result, fmt.Errorf("unauthorized: no property context")
	}

	if err := qtx.SetCurrentPropertyID(ctx, propertyID.String()); err != nil {
		return result, fmt.Errorf("setting rls context: %w", err)
	}

	// Execute business logic
	fnResult, err := fn(qtx)
	if err != nil {
		return result, hf.PsqlErrToCustomErr(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("committing tx: %w", err)
	}

	return fnResult, nil
}
