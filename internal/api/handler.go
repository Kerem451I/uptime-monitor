package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Kerem451I/uptime-monitor/internal/db"
	"github.com/Kerem451I/uptime-monitor/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

type createEndpointRequest struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
	ExpectedStatus  int    `json:"expected_status"`
}

func (h *Handler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	var req createEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// presence check
	if req.Name == "" || req.URL == "" {
		h.writeError(w, http.StatusBadRequest, "name and url are required")
		return
	}

	// SSRF and safety validation
	if err := validateURL(req.URL); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 30 // default
	}

	if req.ExpectedStatus <= 0 {
		req.ExpectedStatus = 200 // default
	}

	endpoint, err := db.CreateEndpoint(r.Context(), h.pool, req.Name, req.URL, req.IntervalSeconds, req.ExpectedStatus)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not create endpoint")
		return
	}

	h.writeJSON(w, http.StatusCreated, endpoint)
}

func (h *Handler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	endpoints, err := db.GetAllEndpoints(r.Context(), h.pool)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not get endpoints")
		return
	}

	h.writeJSON(w, http.StatusOK, endpoints)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// we do not handle errors in the parseID func, so it would not need access to w
// also it does not need to know about HTTP, so its reusable anywhere, CLI tool, test or a different handler
// because it has no dependencies beyond the request

func (h *Handler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	endpoint, err := db.GetEndpointByID(r.Context(), h.pool, id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not get endpoint")
		return
	}

	if endpoint == nil {
		h.writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	h.writeJSON(w, http.StatusOK, endpoint)
}

type updateEndpointRequest struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
	ExpectedStatus  int    `json:"expected_status"`
	IsActive        bool   `json:"is_active"`
}

func (h *Handler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req updateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// presence check
	if req.Name == "" || req.URL == "" {
		h.writeError(w, http.StatusBadRequest, "name and url are required")
		return
	}

	// SSRF and safety validation
	if err := validateURL(req.URL); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 30
	}

	if req.ExpectedStatus <= 0 {
		req.ExpectedStatus = 200
	}

	endpoint, err := db.UpdateEndpoint(r.Context(), h.pool, id, req.Name, req.URL, req.IntervalSeconds, req.ExpectedStatus, req.IsActive)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not update endpoint")
		return
	}

	if endpoint == nil {
		h.writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	h.writeJSON(w, http.StatusOK, endpoint)
}

func (h *Handler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	err = db.DeleteEndpoint(r.Context(), h.pool, id)
	if err != nil {
		if err.Error() == "endpoint not found" {
			h.writeError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "could not delete endpoint")
		return
	}

	// for the err.Error() part, Not the most elegant pattern but it works for now
	// later i can use a sentinel error variable, but that's a refinement for another time

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetEndpointChecks(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// verifying that endpoint exists before querying checks
	// if the endpoint doesn't exist we want a 404, not an empty checks array. this is the right order
	endpoint, err := db.GetEndpointByID(r.Context(), h.pool, id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not get endpoint")
		return
	}
	if endpoint == nil {
		h.writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	// reading an integer query param requires strconv.Atoi since query params are always strings
	q := r.URL.Query()
	days, _ := strconv.Atoi(q.Get("days"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	// ignoring the error with _ if the param is empty or not a number
	// atoi returns 0 which is a valid default for all three int fields
	// no need to handle the error explicitly here

	filter := models.CheckFilter{
		Status: q.Get("status"),
		Days:   days,
		Limit:  limit,
		Offset: offset,
	}

	checks, err := db.GetChecksByEndpointID(r.Context(), h.pool, id, filter)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not get checks")
		return
	}

	h.writeJSON(w, http.StatusOK, checks)
}

func (h *Handler) GetLatestCheck(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// verifying that endpoint exists before querying checks
	// if the endpoint doesn't exist we want a 404, not an empty checks array. this is the right order
	endpoint, err := db.GetEndpointByID(r.Context(), h.pool, id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not get endpoint")
		return
	}
	if endpoint == nil {
		h.writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	check, err := db.GetLatestCheck(r.Context(), h.pool, id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not get the latest check")
		return
	}

	if check == nil {
		h.writeError(w, http.StatusNotFound, "no checks found for this endpoint")
		return
	}

	h.writeJSON(w, http.StatusOK, check)
}

func (h *Handler) GetEndpointStats(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// verifying that endpoint exists before querying checks
	// if the endpoint doesn't exist we want a 404, not an empty checks array. this is the right order
	endpoint, err := db.GetEndpointByID(r.Context(), h.pool, id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not get endpoint")
		return
	}
	if endpoint == nil {
		h.writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	// reading an integer query param requires strconv.Atoi since query params are always strings
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))

	// ignoring the error with _ if the param is empty or not a number
	// atoi returns 0 which is a valid default for int field
	// no need to handle the error explicitly here

	stats, err := db.GetEndpointStats(r.Context(), h.pool, id, days)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not get stats")
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}

func validateURL(rawURL string) error {
	// basic format check
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must use http or https")
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	// resolve host to IP to check for SSRF (server side request forgery)
	host := parsed.Hostname()
	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("could not resolve host")
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if isPrivateIP(ip) {
			return fmt.Errorf("URL resolves to a private IP address")
		}
	}

	return nil
}

// IsPrivate covers RFC 1918 (10.x, 172.16.x, 192.168.x)
// IsLoopback covers 127.0.0.1
// IsLinkLocalUnicast covers 169.254.x (Cloud metadata IPs)
// ip.IsUnspecified() blocks 0.0.0.0
func isPrivateIP(ip net.IP) bool {
	return ip.IsPrivate() ||
		ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsUnspecified()
}
