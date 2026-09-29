package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"oktopus/internal/db/infrastructure/postgres/store"
	"oktopus/internal/db/proxyacl/repository"
)

type RulesRepository struct {
	pool *pgxpool.Pool
	q    *store.Queries
}

func NewRulesRepository(pool *pgxpool.Pool, q *store.Queries) repository.RulesRepository {
	return &RulesRepository{pool: pool, q: q}
}
