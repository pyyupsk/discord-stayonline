package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pyyupsk/discord-stayonline/internal/config"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error("Failed to load config", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to load configuration",
		})
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// ReplaceConfig handles POST /api/config requests.
// Replaces the entire server configuration.
func (h *ConfigHandler) ReplaceConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Servers []config.ServerEntry `json:"servers"`
		Status  config.Status        `json:"status,omitempty"`
	}

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
			"message": "Maximum 15 server entries allowed",
		})
		return
	}

	// Load existing config to preserve TOS acknowledgment
	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error("Failed to load config", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to load configuration",
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
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"servers": cfg.Servers,
	})
}

// UpdateConfig handles PUT /api/config requests.
// Partial update - merges with existing configuration by server ID.
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Servers []config.ServerEntry `json:"servers"`
		Status  config.Status        `json:"status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON request body",
		})
		return
	}

	// Load existing config
	cfg, err := h.store.Load()
	if err != nil {
		h.logger.Error("Failed to load config", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to load configuration",
		})
		return
	}

	// Merge servers by ID
	serverMap := make(map[string]*config.ServerEntry)
	for i := range cfg.Servers {
		serverMap[cfg.Servers[i].ID] = &cfg.Servers[i]
	}

	for _, update := range input.Servers {
		if existing, ok := serverMap[update.ID]; ok {
			// Update existing server
			if update.GuildID != "" {
				existing.GuildID = update.GuildID
			}
			if update.ChannelID != "" {
				existing.ChannelID = update.ChannelID
			}
			existing.ConnectOnStart = update.ConnectOnStart
			if update.Priority > 0 {
				existing.Priority = update.Priority
			}
		} else {
			// Add new server if ID provided
			if update.ID != "" {
				cfg.Servers = append(cfg.Servers, update)
			}
		}
	}

	// Rebuild servers slice from map
	cfg.Servers = make([]config.ServerEntry, 0, len(serverMap))
	for _, s := range serverMap {
		cfg.Servers = append(cfg.Servers, *s)
	}

	// Update global status if provided
	if input.Status != "" {
		cfg.Status = input.Status
	}

	// Validate server count after merge
	if len(cfg.Servers) > config.MaxServerEntries {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "validation_error",
			"message": "Maximum 15 server entries allowed",
		})
		return
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

	h.logger.Info("Configuration updated", "servers", len(cfg.Servers))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"servers": cfg.Servers,
	})
}
