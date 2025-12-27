package handlers

import (
	"log/slog"
	"net/http"

	"github.com/pyyupsk/discord-stayonline/internal/api/responses"
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
func (h *LogsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	logs := h.hub.GetLogs(level)
	responses.JSON(w, http.StatusOK, logs)
}
