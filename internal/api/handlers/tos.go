package handlers

import (
	"log/slog"
	"net/http"

	"github.com/pyyupsk/discord-stayonline/internal/api/responses"
	"github.com/pyyupsk/discord-stayonline/internal/config"
)

type TOSHandler struct {
	store  config.ConfigStore
	logger *slog.Logger
}

func NewTOSHandler(store config.ConfigStore, logger *slog.Logger) *TOSHandler {
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

	if !responses.DecodeJSON(w, r, h.logger, &req) {
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
