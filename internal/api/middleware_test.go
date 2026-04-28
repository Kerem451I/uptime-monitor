package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBytesMiddleware(t *testing.T) {
	// wrap a handler that tries to read the entire body
	middleware := MaxBytesMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			// if it hits the limit, an error is expected here
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// test allowed size (under 1MB)
	smallBody := strings.Repeat("a", 100)
	reqSmall := httptest.NewRequest("POST", "/", strings.NewReader(smallBody))
	rrSmall := httptest.NewRecorder()
	middleware.ServeHTTP(rrSmall, reqSmall)

	if rrSmall.Code != http.StatusOK {
		t.Errorf("expected OK for small body, got %d", rrSmall.Code)
	}

	// test forbidden size (2MB)
	largeBody := strings.Repeat("a", 2<<20)
	reqLarge := httptest.NewRequest("POST", "/", strings.NewReader(largeBody))
	rrLarge := httptest.NewRecorder()
	middleware.ServeHTTP(rrLarge, reqLarge)

	if rrLarge.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 Payload Too Large, got %d", rrLarge.Code)
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// checks all the headers that are defined in middleware
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"X-XSS-Protection":       "0",
		"Cache-Control":          "no-store",
	}

	for header, expected := range headers {
		if got := rr.Header().Get(header); got != expected {
			t.Errorf("header %s = %q, want %q", header, got, expected)
		}
	}
}
