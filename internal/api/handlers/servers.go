package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pyyupsk/discord-stayonline/internal/api/responses"
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

// GetStatuses handles GET /api/statuses requests.
func (h *ServersHandler) GetStatuses(w http.ResponseWriter, r *http.Request) {
	statuses := h.manager.GetAllStatuses()

	result := make(map[string]string)
	for id, status := range statuses {
		result[id] = string(status)
	}

	responses.JSON(w, http.StatusOK, result)
}

// ExecuteAction handles POST /api/servers/{id}/action requests.
func (h *ServersHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/servers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "action" {
		responses.Error(w, http.StatusBadRequest, "invalid_path", "Invalid path format")
		return
	}
	serverID := parts[0]

	if serverID == "" {
		responses.Error(w, http.StatusBadRequest, "invalid_request", "Server ID is required")
		return
	}

	var req struct {
		Action string `json:"action"`
	}

	responses.LimitBody(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		responses.Error(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	if req.Action != "join" && req.Action != "rejoin" && req.Action != "exit" {
		responses.Error(w, http.StatusBadRequest, "invalid_action", "Action must be 'join', 'rejoin', or 'exit'")
		return
	}

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

		responses.Error(w, status, errorCode, err.Error())
		return
	}

	newStatus, _ := h.manager.GetStatus(serverID)

	h.logger.Info("Action executed", "server_id", serverID, "action", req.Action, "new_status", newStatus)
	responses.JSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"server_id":  serverID,
		"action":     req.Action,
		"new_status": string(newStatus),
	})
}
