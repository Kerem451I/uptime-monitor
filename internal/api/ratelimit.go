package api

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// client tracks the limiter and the last time it was seen
// rate.limiter is like a token bucket. Eeery time a user makes a request, they take 1 token
// If the bucket is empty, they are rejected, the bucket refills at a rate of 10 tokens per second
type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	ips map[string]*client
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

// b for burst
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*client),
		r:   r,
		b:   b,
	}
}

// Start begins the background cleanup loop.
// it will run until the provided context is cancelled.
func (i *IPRateLimiter) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop() // Clean up the ticker when the loop exits

	for {
		select {
		case <-ticker.C:
			i.cleanup()
		case <-ctx.Done():
			log.Println("Rate limiter cleanup stopped.")
			return
			// Context was cancelled (for example, server shutdown), exit the goroutine
		}
	}
}

func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	if client, exists := i.ips[ip]; exists {
		client.lastSeen = time.Now()
		return client.limiter
	}

	limiter := rate.NewLimiter(i.r, i.b)
	i.ips[ip] = &client{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	return limiter
}

// the actual cleanup logic
func (i *IPRateLimiter) cleanup() {
	i.mu.Lock()
	defer i.mu.Unlock()

	for ip, client := range i.ips {
		if time.Since(client.lastSeen) > 3*time.Minute {
			delete(i.ips, ip)
		}
	}
}

// returns a middleware that limits requests per IP
// it uses net.SplitHostPort to turn 192.168.1.1:54321 into just 192.168.1.1
// this is critical because a browser often uses a different port for every new connection
// if i didn't strip the port, the user would get a fresh limit every time their browser opened a new connection
func RateLimitMiddleware(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			if !limiter.getLimiter(ip).Allow() {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
