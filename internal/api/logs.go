package api

import (
	"log/slog"
	"net/http"

	"github.com/pyyupsk/discord-stayonline/internal/ws"
)

// LogsHandler handles log retrieval requests.
type LogsHandler struct {
	hub    *ws.Hub
	logger *slog.Logger
}

// NewLogsHandler creates a new logs handler.
func NewLogsHandler(hub *ws.Hub, logger *slog.Logger) *LogsHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogsHandler{
		hub:    hub,
		logger: logger.With("handler", "logs"),
	}
}

// GetLogs handles GET /api/logs requests.
// Query parameters:
//   - level: filter by log level (info, warn, error, debug)
func (h *LogsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	level := r.URL.Query().Get("level")

	logs := h.hub.GetLogs(level)

	writeJSON(w, http.StatusOK, logs)
}
