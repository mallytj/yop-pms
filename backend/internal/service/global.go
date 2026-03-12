package service

import (
	repo "ollerod-pms/internal/adapters/postgresql/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
}
