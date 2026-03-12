package seeders

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Seed struct {
	db *pgxpool.Pool
}
type Seeder interface {
	SeedPlannerData(ctx context.Context) error
}

func NewSeeder(db *pgxpool.Pool) Seeder {
	return &Seed{
		db: db,
	}
}
