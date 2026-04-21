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

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Without this a browser might execute a JSON response as JavaScript if an attacker tricks it into loading it as a script.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// prevents the API from being embedded in an iframe
		w.Header().Set("X-Frame-Options", "DENY")
		// controls how much referrer info is sent with requests. prevents leaking internal URLs to external services
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// to disable the old, buggy XSS Auditors in older browsers, as they can actually create security holes
		w.Header().Set("X-XSS-Protection", "0")
		// API responses should never be cached. prevents sensitive data sitting in browser or proxy caches.
		w.Header().Set("Cache-Control", "no-store")

		next.ServeHTTP(w, r)
	})
}

func MaxBytesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
		next.ServeHTTP(w, r)
	})
}
