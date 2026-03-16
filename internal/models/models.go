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
