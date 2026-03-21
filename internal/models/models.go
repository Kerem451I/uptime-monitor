package models

import (
	"time"
)

type Endpoint struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	IntervalSeconds int       `json:"interval_seconds"`
	ExpectedStatus  int       `json:"expected_status"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

type Check struct {
	ID         int64     `json:"id"`
	EndpointID int64     `json:"endpoint_id"`
	CheckedAt  time.Time `json:"checked_at"`
	Succeeded  bool      `json:"succeeded"`
	StatusCode *int      `json:"status_code"`
	LatencyMs  *int      `json:"latency_ms"`
	ErrorMsg   *string   `json:"error_msg"`
}

type CheckFilter struct {
	Status string `json:"status"` // "failed", "success", or "" for all
	Days   int    `json:"days"`   // 0 means no time filter
	Limit  int    `json:"limit"`  // 0 means use default
	Offset int    `json:"offset"`
}

// instead of a growing list of optional parameters, we group them into a struct.
// cleaner function signature, easier to extend later.

type EndpointStats struct {
	TotalChecks      int     `json:"total_checks"`
	TotalFailures    int     `json:"total_failures"`
	AvgLatencyMs     int     `json:"avg_latency_ms"`
	UptimePercentage float64 `json:"uptime_percentage"`
}
