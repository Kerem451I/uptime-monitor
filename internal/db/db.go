package db

import (
	"context" // to control timeouts, cancellation, and request-scoped values.
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(connString string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// %w wraps the original error rather than just printing it,
	// Wrapping preserves the original error so callers can inspect it if needed.

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("unable to reach database: %w", err)
	}

	// trying to reach the db right after creating the pool,
	// if theres anything wrong, we'll kmow immediately.

	return pool, nil
}
