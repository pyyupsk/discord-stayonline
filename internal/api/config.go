package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pyyupsk/discord-stayonline/internal/config"
)

const (
	errMethodNotAllowed   = "Method not allowed"
	errFailedToLoadConfig = "Failed to load config"
	msgFailedToLoadConfig = "Failed to load configuration"
)

// ConfigHandler handles configuration API requests.
type ConfigHandler struct {
	store  config.ConfigStore
	logger *slog.Logger
}

// NewConfigHandler creates a new config handler.
func NewConfigHandler(store config.ConfigStore, logger *slog.Logger) *ConfigHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConfigHandler{
		store:  store,
		logger: logger.With("handler", "config"),
	}
}

// GetConfig handles GET /api/config requests.
// Returns the current configuration (never includes the token).
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error(errFailedToLoadConfig, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": msgFailedToLoadConfig,
		})
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// ReplaceConfig handles POST /api/config requests.
// Replaces the entire server configuration.
func (h *ConfigHandler) ReplaceConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Servers []config.ServerEntry `json:"servers"`
		Status  config.Status        `json:"status,omitempty"`
	}

	limitBody(r)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON request body",
		})
		return
	}

	// Validate server count
	if len(input.Servers) > config.MaxServerEntries {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "validation_error",
			"message": "Maximum 35 server entries allowed",
		})
		return
	}

	// Load existing config to preserve TOS acknowledgment
	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error(errFailedToLoadConfig, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": msgFailedToLoadConfig,
		})
		return
	}

	// Replace servers
	cfg.Servers = input.Servers

	// Update global status if provided
	if input.Status != "" {
		cfg.Status = input.Status
	}

	// Save config
	if err := h.store.Save(cfg); err != nil {
		h.logger.Error("Failed to save config", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	h.logger.Info("Configuration replaced", "servers", len(cfg.Servers))
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"servers": cfg.Servers,
	})
}

// UpdateConfig handles PUT /api/config requests.
// Partial update - merges with existing configuration by server ID.
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Servers []config.ServerEntry `json:"servers"`
		Status  config.Status        `json:"status,omitempty"`
	}

	limitBody(r)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error(errFailedToLoadConfig, "error", err)
		h.writeError(w, http.StatusInternalServerError, "internal_error", msgFailedToLoadConfig)
		return
	}

	cfg.Servers = mergeServers(cfg.Servers, input.Servers)

	if input.Status != "" {
		cfg.Status = input.Status
	}

	if len(cfg.Servers) > config.MaxServerEntries {
		h.writeError(w, http.StatusBadRequest, "validation_error", "Maximum 35 server entries allowed")
		return
	}

	if err := h.store.Save(cfg); err != nil {
		h.logger.Error("Failed to save config", "error", err)
		h.writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	h.logger.Info("Configuration updated", "servers", len(cfg.Servers))
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"servers": cfg.Servers,
	})
}

// writeError writes a standardized error response.
func (h *ConfigHandler) writeError(w http.ResponseWriter, status int, errCode, message string) {
	writeJSON(w, status, map[string]string{
		"error":   errCode,
		"message": message,
	})
}

// mergeServers merges updates into existing servers by ID.
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

// updateServerEntry applies non-zero fields from update to entry.
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
