package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"vermory/internal/store/postgres/db"
)

type Store struct {
	Pool    *pgxpool.Pool
	Queries *db.Queries
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{
		Pool:    pool,
		Queries: db.New(pool),
	}, nil
}

func (s *Store) Close() {
	s.Pool.Close()
}
