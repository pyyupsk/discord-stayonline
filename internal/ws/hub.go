// Package ws provides WebSocket hub for real-time UI updates.
package ws

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// MessageType represents the type of WebSocket message.
type MessageType string

const (
	TypeStatus        MessageType = "status"
	TypeLog           MessageType = "log"
	TypeConfigChanged MessageType = "config_changed"
	TypeError         MessageType = "error"
	TypeAction        MessageType = "action"
	TypeSubscribe     MessageType = "subscribe"
	TypeUnsubscribe   MessageType = "unsubscribe"
)

// LogLevel represents log severity levels.
type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// StatusUpdate is sent to UI clients when a server connection state changes.
type StatusUpdate struct {
	Type      MessageType `json:"type"`
	ServerID  string      `json:"server_id"`
	Status    string      `json:"status"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// LogMessage is sent to UI clients for log events.
type LogMessage struct {
	Type      MessageType `json:"type"`
	Level     LogLevel    `json:"level"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorMessage is sent to UI clients when an error occurs.
type ErrorMessage struct {
	Type      MessageType `json:"type"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	ServerID  string      `json:"server_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Error codes for WebSocket error messages.
const (
	ErrCodeGatewayError     = "gateway_error"
	ErrCodeConnectionFailed = "connection_failed"
	ErrCodeAuthFailed       = "auth_failed"
	ErrCodeRateLimited      = "rate_limited"
	ErrCodeInvalidConfig    = "invalid_config"
)

// NewStatusUpdate creates a new status update message.
func NewStatusUpdate(serverID, status, message string) *StatusUpdate {
	return &StatusUpdate{
		Type:      TypeStatus,
		ServerID:  serverID,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewLogMessage creates a new log message.
func NewLogMessage(level LogLevel, message string) *LogMessage {
	return &LogMessage{
		Type:      TypeLog,
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewErrorMessage creates a new error message.
func NewErrorMessage(code, message, serverID string) *ErrorMessage {
	return &ErrorMessage{
		Type:      TypeError,
		Code:      code,
		Message:   message,
		ServerID:  serverID,
		Timestamp: time.Now(),
	}
}

// LogEntry represents a stored log entry.
type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// LogStore is the interface for persisting logs.
type LogStore interface {
	AddLog(level, message string) error
	GetLogs(level string) ([]LogEntry, error)
}

// Hub manages WebSocket client connections and message broadcasting.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *slog.Logger
	logStore   LogStore
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *slog.Logger, logStore LogStore) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger.With("component", "ws-hub"),
		logStore:   logStore,
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("Client registered", "total_clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Debug("Client unregistered", "total_clients", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				client.Send(message)
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(data []byte) {
	select {
	case h.broadcast <- data:
	default:
		h.logger.Warn("Broadcast channel full, dropping message")
	}
}

// BroadcastStatus sends a status update to all clients and stores it.
func (h *Hub) BroadcastStatus(serverID, status, message string) {
	update := NewStatusUpdate(serverID, status, message)
	data, err := json.Marshal(update)
	if err != nil {
		h.logger.Error("Failed to marshal status update", "error", err)
		return
	}
	h.Broadcast(data)

	// Store status update as log entry
	if h.logStore != nil && message != "" {
		logMsg := fmt.Sprintf("[%s] %s", serverID, message)
		if err := h.logStore.AddLog("info", logMsg); err != nil {
			h.logger.Error("Failed to store status log entry", "error", err)
		}
	}
}

// BroadcastLog sends a log message to subscribed clients and stores it.
func (h *Hub) BroadcastLog(level LogLevel, message string) {
	logMsg := NewLogMessage(level, message)

	// Store log entry to database
	if h.logStore != nil {
		if err := h.logStore.AddLog(string(level), message); err != nil {
			h.logger.Error("Failed to store log entry", "error", err)
		}
	}

	data, err := json.Marshal(logMsg)
	if err != nil {
		h.logger.Error("Failed to marshal log message", "error", err)
		return
	}

	h.mu.RLock()
	for client := range h.clients {
		if client.IsSubscribed("logs") {
			client.Send(data)
		}
	}
	h.mu.RUnlock()
}

// GetLogs returns stored log entries from the database, optionally filtered by level.
func (h *Hub) GetLogs(level string) []LogEntry {
	if h.logStore == nil {
		return nil
	}

	logs, err := h.logStore.GetLogs(level)
	if err != nil {
		h.logger.Error("Failed to get logs from store", "error", err)
		return nil
	}
	return logs
}

// BroadcastError sends an error message to all clients.
func (h *Hub) BroadcastError(code, message, serverID string) {
	errMsg := NewErrorMessage(code, message, serverID)
	data, err := json.Marshal(errMsg)
	if err != nil {
		h.logger.Error("Failed to marshal error message", "error", err)
		return
	}
	h.Broadcast(data)
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Close stops the hub (for graceful shutdown).
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
}
