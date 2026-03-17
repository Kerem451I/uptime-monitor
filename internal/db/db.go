package db

import (
	"context" // to control timeouts, cancellation, and request-scoped values.
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(connString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// %w wraps the original error rather than just printing it,
	// Wrapping preserves the original error so callers can inspect it if needed.

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to reach adatabase: %w", err)
	}

	// trying to reach the db right after creating the pool,
	// if theres anything wrong, we'll kmow immediately.

	return pool, nil
}
