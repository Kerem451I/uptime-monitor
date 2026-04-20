package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Kerem451I/uptime-monitor/internal/metrics"
)

// responseWriter wraps http.ResponseWriter to capture status code
// since default http.ResponseWriter does not expose the status code after writing
type responseWriter struct {
	http.ResponseWriter
	status int
}

// override WriteHeader to capture status
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func MetricsMiddleware(pattern string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// increment in flight gauge to track in flight requests
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		// calculate duration and record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.status)

		// record duration histogram
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, pattern, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, pattern).Observe(duration)
	})
}
