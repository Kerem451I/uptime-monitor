package api

import (
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
	i := &IPRateLimiter{
		ips: make(map[string]*client),
		r:   r,
		b:   b,
	}

	// start a background cleanup goroutine
	go i.cleanupClients()

	return i
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

func (i *IPRateLimiter) cleanupClients() {
	for {
		time.Sleep(time.Minute)
		i.mu.Lock()
		for ip, client := range i.ips {
			if time.Since(client.lastSeen) > 3*time.Minute {
				delete(i.ips, ip)
			}
		}
		i.mu.Unlock()
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
