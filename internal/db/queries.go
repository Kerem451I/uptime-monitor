package db

import (
	"context"
	"fmt"

	"github.com/Kerem451I/uptime-monitor/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateEndpoint(ctx context.Context, pool *pgxpool.Pool, name string, url string, intervalSeconds int, expectedStatus int) (*models.Endpoint, error) {
	query := `
        INSERT INTO endpoints (name, url, interval_seconds, expected_status, is_active)
        VALUES ($1, $2, $3, $4, true)
        RETURNING id, name, url, interval_seconds, expected_status, is_active, created_at
    `

	endpoint := &models.Endpoint{}

	err := pool.QueryRow(ctx, query, name, url, intervalSeconds, expectedStatus).Scan(
		&endpoint.ID,
		&endpoint.Name,
		&endpoint.URL,
		&endpoint.IntervalSeconds,
		&endpoint.ExpectedStatus,
		&endpoint.IsActive,
		&endpoint.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create endpoint: %w", err)
	}

	return endpoint, nil
}
