package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pyyupsk/discord-stayonline/internal/manager"
)

// ServersHandler handles server action requests.
type ServersHandler struct {
	manager *manager.SessionManager
	logger  *slog.Logger
}

// NewServersHandler creates a new servers handler.
func NewServersHandler(mgr *manager.SessionManager, logger *slog.Logger) *ServersHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ServersHandler{
		manager: mgr,
		logger:  logger.With("handler", "servers"),
	}
}

// ExecuteAction handles POST /api/servers/{id}/action requests.
func (h *ServersHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse server ID from path
	// Path format: /api/servers/{id}/action
	path := strings.TrimPrefix(r.URL.Path, "/api/servers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "action" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_path",
			"message": "Invalid path format",
		})
		return
	}
	serverID := parts[0]

	if serverID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Server ID is required",
		})
		return
	}

	// Parse action from body
	var req struct {
		Action string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON request body",
		})
		return
	}

	// Validate action
	if req.Action != "join" && req.Action != "rejoin" && req.Action != "exit" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_action",
			"message": "Action must be 'join', 'rejoin', or 'exit'",
		})
		return
	}

	// Execute action
	var err error
	switch req.Action {
	case "join":
		err = h.manager.Join(serverID)
	case "rejoin":
		err = h.manager.Rejoin(serverID)
	case "exit":
		err = h.manager.Exit(serverID)
	}

	if err != nil {
		h.logger.Error("Action failed", "server_id", serverID, "action", req.Action, "error", err)

		status := http.StatusInternalServerError
		errorCode := "action_failed"

		switch err {
		case manager.ErrServerNotFound:
			status = http.StatusNotFound
			errorCode = "server_not_found"
		case manager.ErrTooManyConnections:
			status = http.StatusConflict
			errorCode = "too_many_connections"
		case manager.ErrTOSNotAcknowledged:
			status = http.StatusForbidden
			errorCode = "tos_not_acknowledged"
		case manager.ErrAlreadyConnected:
			status = http.StatusConflict
			errorCode = "already_connected"
		case manager.ErrNotConnected:
			status = http.StatusConflict
			errorCode = "not_connected"
		}

		writeJSON(w, status, map[string]string{
			"error":   errorCode,
			"message": err.Error(),
		})
		return
	}

	// Get new status
	newStatus, _ := h.manager.GetStatus(serverID)

	h.logger.Info("Action executed", "server_id", serverID, "action", req.Action, "new_status", newStatus)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"server_id":  serverID,
		"action":     req.Action,
		"new_status": string(newStatus),
	})
}
