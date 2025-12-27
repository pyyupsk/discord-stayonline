// Package api provides HTTP API handlers for the Discord presence service.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pyyupsk/discord-stayonline/internal/config"
)

// TOSHandler handles TOS acknowledgment requests.
type TOSHandler struct {
	store  config.ConfigStore
	logger *slog.Logger
}

// NewTOSHandler creates a new TOS handler.
func NewTOSHandler(store config.ConfigStore, logger *slog.Logger) *TOSHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TOSHandler{
		store:  store,
		logger: logger.With("handler", "tos"),
	}
}

// AcknowledgeTOS handles POST /api/acknowledge-tos requests.
func (h *TOSHandler) AcknowledgeTOS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Acknowledged bool `json:"acknowledged"`
	}

	limitBody(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON request body",
		})
		return
	}

	if !req.Acknowledged {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "acknowledged must be true",
		})
		return
	}

	// Load current config
	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error("Failed to load config", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to load configuration",
		})
		return
	}

	// Update TOS acknowledgment
	cfg.TOSAcknowledged = true

	// Save config
	if err := h.store.Save(cfg); err != nil {
		h.logger.Error("Failed to save config", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to save configuration",
		})
		return
	}

	h.logger.Info("TOS acknowledged")
	writeJSON(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}