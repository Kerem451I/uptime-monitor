package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kerem451I/uptime-monitor/internal/models"
	"github.com/jackc/pgx/v5"
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

func GetAllEndpoints(ctx context.Context, pool *pgxpool.Pool) ([]models.Endpoint, error) {
	query := `
        SELECT id, name, url, interval_seconds, expected_status, is_active, created_at
        FROM endpoints
    `

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("could not get endpoints: %w", err)
	}
	defer rows.Close()

	endpoints := []models.Endpoint{} // initializing as empty slice not nil
	for rows.Next() {
		var e models.Endpoint
		err := rows.Scan(&e.ID, &e.Name, &e.URL, &e.IntervalSeconds,
			&e.ExpectedStatus, &e.IsActive, &e.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("could not scan endpoint: %w", err)
		}
		endpoints = append(endpoints, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoints: %w", err)
	}

	return endpoints, nil
}

func GetEndpointByID(ctx context.Context, pool *pgxpool.Pool, id int64) (*models.Endpoint, error) {
	query := `
        SELECT id, name, url, interval_seconds, expected_status, is_active, created_at
        FROM endpoints
        WHERE id = $1
    `

	endpoint := &models.Endpoint{}

	err := pool.QueryRow(ctx, query, id).Scan(
		&endpoint.ID,
		&endpoint.Name,
		&endpoint.URL,
		&endpoint.IntervalSeconds,
		&endpoint.ExpectedStatus,
		&endpoint.IsActive,
		&endpoint.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // endpoint not found
		}
		return nil, fmt.Errorf("could not get endpoint: %w", err)
	}

	return endpoint, nil
}

func UpdateEndpoint(ctx context.Context, pool *pgxpool.Pool, id int64, name string, url string, intervalSeconds int, expectedStatus int, isActive bool) (*models.Endpoint, error) {
	query := `
        UPDATE endpoints
        SET name = $1, url = $2, interval_seconds = $3, expected_status = $4, is_active = $5
        WHERE id = $6
        RETURNING id, name, url, interval_seconds, expected_status, is_active, created_at
    `

	endpoint := &models.Endpoint{}
	err := pool.QueryRow(ctx, query, name, url, intervalSeconds, expectedStatus, isActive, id).Scan(
		&endpoint.ID,
		&endpoint.Name,
		&endpoint.URL,
		&endpoint.IntervalSeconds,
		&endpoint.ExpectedStatus,
		&endpoint.IsActive,
		&endpoint.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // endpoint not found
		}
		return nil, fmt.Errorf("could not get endpoint: %w", err)
	}

	return endpoint, nil
}

func DeleteEndpoint(ctx context.Context, pool *pgxpool.Pool, id int64) error {
	query := `
        DELETE FROM endpoints
		WHERE id = $1
    `

	result, err := pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("could not delete endpoint: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("endpoint not found")
	}

	return nil
}

func InsertCheck(ctx context.Context, pool *pgxpool.Pool, endpointID int64, succeeded bool, statusCode *int, latencyMs *int, errorMsg *string) error {
	// statusCode *int, latencyMs *int, errorMsg *string are passed straight to pgx. if they're nil, NULL goes into the database. No special handling needed.
	query := `
        INSERT INTO checks (endpoint_id, succeeded, status_code, latency_ms, error_msg)
        VALUES ($1, $2, $3, $4, $5)
    `

	_, err := pool.Exec(ctx, query, endpointID, succeeded, statusCode, latencyMs, errorMsg)
	if err != nil {
		return fmt.Errorf("could not insert check: %w", err)
	}

	// we're not returning the created check. the worker fires and forgets, it doesn't need the row back.
	return nil
}

func GetChecksByEndpointID(ctx context.Context, pool *pgxpool.Pool, endpointID int64, filter models.CheckFilter) ([]models.Check, error) {
	query := `
        SELECT id, endpoint_id, checked_at, succeeded, status_code, latency_ms, error_msg
        FROM checks
        WHERE endpoint_id = $1
    `
	args := []any{endpointID}
	// this creates a slice that will hold all the values we want to pass to the query
	// we start it with endpointID already in it because $1 in the query is always endpointID
	// []any means a slice that can hold any type.

	argPos := 2
	// tracks which $N placeholder we're on as we add conditions
	// starts at 2 because $1 is already endpointID

	if filter.Status == "failed" {
		query += fmt.Sprintf(" AND succeeded = $%d", argPos)
		args = append(args, false)
		argPos++
	} else if filter.Status == "success" {
		query += fmt.Sprintf(" AND succeeded = $%d", argPos)
		args = append(args, true)
		argPos++
	}
	// we can't write the value false directly into the SQL string, that's SQL injection territory
	// so it builds the string " AND succeeded = $2". The %d is a format verb for integers, so argPos which is 2 gets inserted there.
	// args = [endpointID, false]. When pgx runs the query it will substitute $2 with false

	if filter.Days > 0 {
		if filter.Days > 90 {
			filter.Days = 90 // server side cap
		}
		query += fmt.Sprintf(" AND checked_at >= NOW() - INTERVAL '%d days'", filter.Days)
	}
	// we're formatting `filter.Days` directly into the SQL string with `%d` instead of using a `$N` placeholder
	// this is safe because `filter.Days` is a Go `int` it can only ever be a number, never a string a user could inject SQL through
	// we also capped it at 90 above. We don't add to `argPos` here because we didn't add a new placeholder

	query += " ORDER BY checked_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50 // default
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, filter.Offset)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not get checks: %w", err)
	}
	defer rows.Close()

	var checks []models.Check
	for rows.Next() {
		var c models.Check
		err := rows.Scan(&c.ID, &c.EndpointID, &c.CheckedAt, &c.Succeeded,
			&c.StatusCode, &c.LatencyMs, &c.ErrorMsg)
		if err != nil {
			return nil, fmt.Errorf("could not scan check: %w", err)
		}
		checks = append(checks, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating checks: %w", err)
	}

	return checks, nil
}

func GetLatestCheck(ctx context.Context, pool *pgxpool.Pool, endpointID int64) (*models.Check, error) {
	query := `
        SELECT id, endpoint_id, checked_at, succeeded, status_code, latency_ms, error_msg
        FROM checks
        WHERE endpoint_id = $1
		ORDER BY checked_at DESC
		LIMIT 1
    `

	check := &models.Check{}

	err := pool.QueryRow(ctx, query, endpointID).Scan(
		&check.ID,
		&check.EndpointID,
		&check.CheckedAt,
		&check.Succeeded,
		&check.StatusCode,
		&check.LatencyMs,
		&check.ErrorMsg,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no checks yet, not an error
		}
		return nil, fmt.Errorf("could not get latest check: %w", err)
	}

	return check, nil
}

func GetEndpointStats(ctx context.Context, pool *pgxpool.Pool, endpointID int64, days int) (*models.EndpointStats, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	query := fmt.Sprintf(`
        SELECT
            COUNT(*) as total_checks,
            COUNT(*) FILTER (WHERE succeeded = false) as total_failures,
            COALESCE(ROUND(AVG(latency_ms)), 0) as avg_latency_ms,
            COALESCE(ROUND(100.0 * COUNT(*) FILTER (WHERE succeeded = true) / NULLIF(COUNT(*), 0), 2), 0) as uptime_percentage
        FROM checks
        WHERE endpoint_id = $1
        AND checked_at >= NOW() - INTERVAL '%d days'
    `, days)

	// coalesce: if the value is NULL, return 0 instead. needed because if there are zero checks, AVG returns NULL not 0
	// nullif: if total checks is 0, return NULL instead of 0. this prevents division by zero
	// combined with COALESCE wrapping the whole thing, we get 0% uptime when there are no checks rather than a division by zero crash

	stats := &models.EndpointStats{}
	err := pool.QueryRow(ctx, query, endpointID).Scan(
		&stats.TotalChecks,
		&stats.TotalFailures,
		&stats.AvgLatencyMs,
		&stats.UptimePercentage,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get endpoint stats: %w", err)
	}

	return stats, nil
}
