package service

import (
	"context"
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RatePlan interface {
	Get(ctx context.Context) ([]repo.GetRatePlansRow, error)
}

type ratePlanService struct {
	*Svc
}

func NewRatePlanService(r repo.Queries, db *pgxpool.Pool) RatePlan {
	return &ratePlanService{
		&Svc{
			repo: &r,
			db:   db,
		},
	}
}

func (s *ratePlanService) Get(ctx context.Context) ([]repo.GetRatePlansRow, error) {
	return ExecuteTx(s.Svc, ctx, func(qtx *repo.Queries) ([]repo.GetRatePlansRow, error) {
		// TODO Auth check

		return qtx.GetRatePlans(ctx)
	})
}
