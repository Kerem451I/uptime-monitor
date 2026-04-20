package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// the reason i used promauto, because it automatically
	// creates metric and registers it with prometheus. so its simpler

	// counter, only goes up
	HTTPRequestsTotal = promauto.NewCounterVec(
		// this defines the identity of the metric
		prometheus.CounterOpts{
			Namespace: "uptime",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// tracks distribution, not just value
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "uptime",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// gauges can go up and down
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "uptime",
			Name:      "http_requests_in_flight",
			Help:      "Current number of in-flight HTTP requests",
		},
	)

	ChecksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "uptime",
			Subsystem: "checker",
			Name:      "checks_total",
			Help:      "Total number of endpoint checks performed",
		},
		[]string{"endpoint_id", "status"},
	)

	CheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "uptime",
			Subsystem: "checker",
			Name:      "check_duration_seconds",
			Help:      "Duration of endpoint checks in seconds",
			Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"endpoint_id"},
	)
)
