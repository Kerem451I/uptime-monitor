package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// metrics and health are unwrapped
	// if i wrap the health, "total requests" metric will be dominated by health checks
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /health", h.HealthCheck)

	// pass the full pattern for Go's router,
	// and a clean path for Prometheus labels.
	register := func(pattern string, labelPath string, handlerFunc http.HandlerFunc) {
		mux.Handle(pattern, MetricsMiddleware(labelPath, handlerFunc))
	}

	register("POST /endpoints", "/endpoints", h.CreateEndpoint)
	register("GET /endpoints", "/endpoints", h.ListEndpoints)
	register("GET /endpoints/{id}", "/endpoints/{id}", h.GetEndpoint)
	register("PUT /endpoints/{id}", "/endpoints/{id}", h.UpdateEndpoint)
	register("DELETE /endpoints/{id}", "/endpoints/{id}", h.DeleteEndpoint)

	register("GET /endpoints/{id}/checks", "/endpoints/{id}/checks", h.GetEndpointChecks)
	register("GET /endpoints/{id}/checks/latest", "/endpoints/{id}/checks/latest", h.GetLatestCheck)
	register("GET /endpoints/{id}/stats", "/endpoints/{id}/stats", h.GetEndpointStats)

	// RateLimitMiddleware(limiter) returns a wrapper function,
	// which is then immediately applied to the handler to produce a rate limited handler

	limiter := NewIPRateLimiter(10, 20)

	// return SecurityHeadersMiddleware(RateLimitMiddleware(limiter)(MaxBytesMiddleware(mux)))

	handler := http.Handler(mux)

	// apply from inner to outer, the order matters
	handler = MaxBytesMiddleware(handler)
	handler = RateLimitMiddleware(limiter)(handler)
	handler = SecurityHeadersMiddleware(handler)

	return handler
}
