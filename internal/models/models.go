package models

import "time"

type Endpoint struct {
	ID              int64
	Name            string
	URL             string
	IntervalSeconds int
	ExpectedStatus  int
	IsActive        bool
	CreatedAt       time.Time
}

type Check struct {
	ID         int64
	EndpointID int64
	CheckedAt  time.Time
	Succeeded  bool
	StatusCode *int
	LatencyMs  *int
	ErrorMsg   *string
}

type CheckFilter struct {
	Status string // "failed", "success", or "" for all
	Days   int    // 0 means no time filter
	Limit  int    // 0 means use default
	Offset int
}

// instead of a growing list of optional parameters, we group them into a struct.
// cleaner function signature, easier to extend later.

type EndpointStats struct {
	TotalChecks      int
	TotalFailures    int
	AvgLatencyMs     int
	UptimePercentage float64
}
