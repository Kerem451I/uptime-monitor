package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Kerem451I/uptime-monitor/internal/db"
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

	if req.Name == "" || req.URL == "" {
		h.writeError(w, http.StatusBadRequest, "name and url are required")
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

	if req.Name == "" || req.URL == "" {
		h.writeError(w, http.StatusBadRequest, "name and url are required")
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

	// for the del.Error() part, Not the most elegant pattern but it works for now
	// later i can use a sentinel error variable, but that's a refinement for another time

	w.WriteHeader(http.StatusNoContent)
}
