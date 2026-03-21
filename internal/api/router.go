package api

import (
	"net/http"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", h.HealthCheck)

	// Endpoints CRUD
	mux.HandleFunc("POST /endpoints", h.CreateEndpoint)
	mux.HandleFunc("GET /endpoints", h.ListEndpoints)
	mux.HandleFunc("GET /endpoints/{id}", h.GetEndpoint)
	mux.HandleFunc("PUT /endpoints/{id}", h.UpdateEndpoint)
	mux.HandleFunc("DELETE /endpoints/{id}", h.DeleteEndpoint)

	// Checks and stats
	mux.HandleFunc("GET /endpoints/{id}/checks", h.GetEndpointChecks)
	mux.HandleFunc("GET /endpoints/{id}/checks/latest", h.GetLatestCheck)
	mux.HandleFunc("GET /endpoints/{id}/stats", h.GetEndpointStats)

	return mux
}
