package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	"github.com/lexxcode1/yop-pms/internal/store"
)

// ExecuteTx runs fn inside a serialisable transaction with RLS context set.
//
// Steps:
//  1. Begin tx
//  2. Create store.Queries scoped to tx (qtx)
//  3. Set app.current_property_id from ctx (RLS gate)
//  4. Set app.current_user_id from ctx if present (M18 audit trigger)
//  5. Run fn(qtx)
//  6. On fn error → rollback, wrap via PsqlErrToCustomErr
//  7. On success → commit
//
// Generic T allows any return type; callers supply the fn closure.
func ExecuteTx[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	base *store.Queries,
	fn func(qtx *store.Queries) (T, error),
) (result T, err error) {
	tx, beginErr := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if beginErr != nil {
		return result, fmt.Errorf("begin tx: %w", beginErr)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			err = errors.Join(err, rbErr)
		}
	}()

	qtx := base.WithTx(tx)

	// RLS context — must be set inside tx
	propertyID := helpers.GetPropertyIDFromCtx(ctx)
	if propertyID == uuid.Nil {
		return result, apierror.ErrBadRequest.WithMessage("X-Property-ID header required")
	}
	if err := qtx.SetCurrentPropertyID(ctx, propertyID.String()); err != nil {
		return result, fmt.Errorf("set rls property id: %w", err)
	}

	// Set user ID if present (for M18 audit trigger)
	if uid := helpers.GetUserIDFromCtx(ctx); uid != uuid.Nil {
		_, execErr := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", uid.String())
		if execErr != nil {
			return result, fmt.Errorf("set rls user id: %w", execErr)
		}
	}

	result, fnErr := fn(qtx)
	if fnErr != nil {
		var apiErr *apierror.APIError
		if errors.As(fnErr, &apiErr) {
			return result, apiErr
		}
		return result, apierror.MapPostgresError(fnErr)
	}

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit tx: %w", err)
	}

	return result, nil
}
