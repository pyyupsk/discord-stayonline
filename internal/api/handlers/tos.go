package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pyyupsk/discord-stayonline/internal/api/responses"
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
	var req struct {
		Acknowledged bool `json:"acknowledged"`
	}

	responses.LimitBody(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		responses.Error(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	if !req.Acknowledged {
		responses.Error(w, http.StatusBadRequest, "invalid_request", "acknowledged must be true")
		return
	}

	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error(responses.ErrLoadConfig, "error", err)
		responses.Error(w, http.StatusInternalServerError, "internal_error", responses.ErrLoadConfigMsg)
		return
	}

	cfg.TOSAcknowledged = true

	if err := h.store.Save(cfg); err != nil {
		h.logger.Error(responses.ErrSaveConfig, "error", err)
		responses.Error(w, http.StatusInternalServerError, "internal_error", responses.ErrSaveConfigMsg)
		return
	}

	h.logger.Info("TOS acknowledged")
	responses.JSON(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}
