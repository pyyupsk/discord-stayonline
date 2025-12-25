package api

import (
	"net/http"
)

// HealthHandler handles health check requests.
type HealthHandler struct{}

// NewHealthHandler creates a new health handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health handles GET/HEAD /health requests.
// Returns 200 OK with body "OK" if service is running.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		w.Write([]byte("OK"))
	}
}
