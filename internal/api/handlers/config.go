package handlers

import (
	"log/slog"
	"net/http"

	"github.com/pyyupsk/discord-stayonline/internal/api/responses"
	"github.com/pyyupsk/discord-stayonline/internal/config"
)

type ConfigHandler struct {
	store  config.ConfigStore
	logger *slog.Logger
}

func NewConfigHandler(store config.ConfigStore, logger *slog.Logger) *ConfigHandler {
	return &ConfigHandler{
		store:  store,
		logger: logger.With("handler", "config"),
	}
}

// GetConfig handles GET /api/config requests.
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error(responses.ErrLoadConfig, "error", err)
		responses.Error(w, http.StatusInternalServerError, "internal_error", responses.ErrLoadConfigMsg)
		return
	}
	responses.JSON(w, http.StatusOK, cfg)
}

// ReplaceConfig handles POST /api/config requests.
func (h *ConfigHandler) ReplaceConfig(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Servers []config.ServerEntry `json:"servers"`
		Status  config.Status        `json:"status,omitempty"`
	}

	if !responses.DecodeJSON(w, r, h.logger, &input) {
		return
	}

	if len(input.Servers) > config.MaxServerEntries {
		responses.Error(w, http.StatusBadRequest, "validation_error", "Maximum 35 server entries allowed")
		return
	}

	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error(responses.ErrLoadConfig, "error", err)
		responses.Error(w, http.StatusInternalServerError, "internal_error", responses.ErrLoadConfigMsg)
		return
	}

	cfg.Servers = input.Servers
	if input.Status != "" {
		cfg.Status = input.Status
	}

	if err := h.store.Save(cfg); err != nil {
		h.logger.Error(responses.ErrSaveConfig, "error", err)
		responses.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	h.logger.Info("Configuration replaced", "servers", len(cfg.Servers))
	responses.JSON(w, http.StatusOK, map[string]any{
		"success": true,
		"servers": cfg.Servers,
	})
}

// UpdateConfig handles PUT /api/config requests.
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Servers []config.ServerEntry `json:"servers"`
		Status  config.Status        `json:"status,omitempty"`
	}

	if !responses.DecodeJSON(w, r, h.logger, &input) {
		return
	}

	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error(responses.ErrLoadConfig, "error", err)
		responses.Error(w, http.StatusInternalServerError, "internal_error", responses.ErrLoadConfigMsg)
		return
	}

	cfg.Servers = mergeServers(cfg.Servers, input.Servers)
	if input.Status != "" {
		cfg.Status = input.Status
	}

	if len(cfg.Servers) > config.MaxServerEntries {
		responses.Error(w, http.StatusBadRequest, "validation_error", "Maximum 35 server entries allowed")
		return
	}

	if err := h.store.Save(cfg); err != nil {
		h.logger.Error(responses.ErrSaveConfig, "error", err)
		responses.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	h.logger.Info("Configuration updated", "servers", len(cfg.Servers))
	responses.JSON(w, http.StatusOK, map[string]any{
		"success": true,
		"servers": cfg.Servers,
	})
}

func mergeServers(existing, updates []config.ServerEntry) []config.ServerEntry {
	serverMap := make(map[string]*config.ServerEntry)
	for i := range existing {
		serverMap[existing[i].ID] = &existing[i]
	}

	for _, update := range updates {
		if entry, ok := serverMap[update.ID]; ok {
			updateServerEntry(entry, update)
		} else if update.ID != "" {
			newEntry := update
			serverMap[update.ID] = &newEntry
		}
	}

	result := make([]config.ServerEntry, 0, len(serverMap))
	for _, s := range serverMap {
		result = append(result, *s)
	}
	return result
}

func updateServerEntry(entry *config.ServerEntry, update config.ServerEntry) {
	if update.GuildID != "" {
		entry.GuildID = update.GuildID
	}
	if update.ChannelID != "" {
		entry.ChannelID = update.ChannelID
	}
	entry.ConnectOnStart = update.ConnectOnStart
	if update.Priority > 0 {
		entry.Priority = update.Priority
	}
}
