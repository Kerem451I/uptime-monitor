package checker

import (
	"context"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Kerem451I/uptime-monitor/internal/db"
	"github.com/Kerem451I/uptime-monitor/internal/metrics"
	"github.com/Kerem451I/uptime-monitor/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Checker struct {
	pool   *pgxpool.Pool
	client *http.Client
}

func New(pool *pgxpool.Pool) *Checker {
	return &Checker{
		pool: pool,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type checkResult struct {
	succeeded  bool
	statusCode *int
	latencyMs  *int
	errorMsg   *string
}

func (c *Checker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		// waits for one of multiple channel events (ctx cancellation or ticker trigger)
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			endpoints, err := db.GetAllEndpoints(ctx, c.pool)
			if err != nil {
				log.Printf("checker: could not fetch endpoints: %v", err)
				continue
			}

			for _, ep := range endpoints {
				if !ep.IsActive {
					continue
				}
				go c.processEndpoint(ctx, ep)
			}
		}
	}
}

func (c *Checker) processEndpoint(ctx context.Context, ep models.Endpoint) {
	check, err := db.GetLatestCheck(ctx, c.pool, ep.ID)
	if err != nil {
		log.Printf("checker: endpoint %d get last check error: %v", ep.ID, err)
		return
	}

	interval := time.Duration(ep.IntervalSeconds) * time.Second
	if check != nil && time.Since(check.CheckedAt) < interval {
		return
	}

	result := c.ping(ctx, ep.URL, ep.ExpectedStatus)

	if err := db.InsertCheck(ctx, c.pool, ep.ID, result.succeeded, result.statusCode, result.latencyMs, result.errorMsg); err != nil {
		log.Printf("checker: endpoint %d insert check error: %v", ep.ID, err)
		return
	}

	endpointID := strconv.FormatInt(ep.ID, 10)
	statusLabel := "success"
	if !result.succeeded {
		statusLabel = "failure"
	}

	metrics.ChecksTotal.WithLabelValues(endpointID, statusLabel).Inc()
	if result.latencyMs != nil {
		metrics.CheckDuration.WithLabelValues(endpointID).Observe(float64(*result.latencyMs) / 1000.0)
	}
}

func (c *Checker) ping(ctx context.Context, url string, expectedStatus int) checkResult {
	start := time.Now()

	// creates a new HTTP request and binds it to a context.
	// if ctx is canceled, the request will be aborted to prevent hanging requests).
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		msg := err.Error()
		return checkResult{
			succeeded: false,
			errorMsg:  &msg,
		}
	}

	// sends the HTTP request and waits for a response.
	resp, err := c.client.Do(req)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		msg := err.Error()
		return checkResult{
			succeeded: false,
			latencyMs: &latency,
			errorMsg:  &msg,
		}
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	// reads the entire response body and throws it away, then closes the connection.
	// to allow connection reuse (avoids leaks)

	status := resp.StatusCode
	succeeded := status == expectedStatus

	return checkResult{
		succeeded:  succeeded,
		statusCode: &status,
		latencyMs:  &latency,
		errorMsg:   nil,
	}
}
