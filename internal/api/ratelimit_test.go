package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// limiter must allow requests within burst capacity
func TestIPRateLimiter_AllowsUnderLimit(t *testing.T) {
	limiter := NewIPRateLimiter(10, 5)
	for i := 0; i < 5; i++ {
		if !limiter.getLimiter("127.0.0.1").Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

// limiter must correctly block when tokens are exhausted
func TestIPRateLimiter_BlocksOverLimit(t *testing.T) {
	// burst of 2, very slow refill
	limiter := NewIPRateLimiter(rate.Limit(0.001), 2)
	limiter.getLimiter("127.0.0.1").Allow()      // 1
	limiter.getLimiter("127.0.0.1").Allow()      // 2
	if limiter.getLimiter("127.0.0.1").Allow() { // 3 - should be blocked
		t.Error("third request should be blocked")
	}
}

// each IP must have its own separate bucket
func TestIPRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Limit(0.001), 1)
	limiter.getLimiter("1.1.1.1").Allow()       // exhaust ip1
	if !limiter.getLimiter("2.2.2.2").Allow() { // ip2 should still work
		t.Error("different IP should have its own bucket")
	}
}

// inactive IPs must be removed from memory
func TestIPRateLimiter_Cleanup(t *testing.T) {
	limiter := &IPRateLimiter{
		ips: make(map[string]*client),
		r:   10,
		b:   20,
	}
	limiter.getLimiter("1.1.1.1")
	// manually set lastSeen to old time
	limiter.mu.Lock()
	limiter.ips["1.1.1.1"].lastSeen = time.Now().Add(-10 * time.Minute)
	limiter.mu.Unlock()

	limiter.cleanup()

	limiter.mu.Lock()
	_, exists := limiter.ips["1.1.1.1"]
	limiter.mu.Unlock()

	if exists {
		t.Error("old client should have been cleaned up")
	}
}

// middleware must correctly block requests and returns 429
func TestRateLimitMiddleware_Returns429(t *testing.T) {
	limiter := NewIPRateLimiter(rate.Limit(0.001), 0)
	middleware := RateLimitMiddleware(limiter)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.1.1.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr.Code)
	}
}
