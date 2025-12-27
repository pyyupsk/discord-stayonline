package ws

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

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

type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

type StatusUpdate struct {
	Type      MessageType `json:"type"`
	ServerID  string      `json:"server_id"`
	Status    string      `json:"status"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type LogMessage struct {
	Type      MessageType `json:"type"`
	Level     LogLevel    `json:"level"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}

type ErrorMessage struct {
	Type      MessageType `json:"type"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	ServerID  string      `json:"server_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

const (
	ErrCodeGatewayError     = "gateway_error"
	ErrCodeConnectionFailed = "connection_failed"
	ErrCodeAuthFailed       = "auth_failed"
	ErrCodeRateLimited      = "rate_limited"
	ErrCodeInvalidConfig    = "invalid_config"
)

func NewStatusUpdate(serverID, status, message string) *StatusUpdate {
	return &StatusUpdate{
		Type:      TypeStatus,
		ServerID:  serverID,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func NewLogMessage(level LogLevel, message string) *LogMessage {
	return &LogMessage{
		Type:      TypeLog,
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func NewErrorMessage(code, message, serverID string) *ErrorMessage {
	return &ErrorMessage{
		Type:      TypeError,
		Code:      code,
		Message:   message,
		ServerID:  serverID,
		Timestamp: time.Now(),
	}
}

type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type LogStore interface {
	AddLog(level, message string) error
	GetLogs(level string) ([]LogEntry, error)
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *slog.Logger
	logStore   LogStore
}

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

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(data []byte) {
	select {
	case h.broadcast <- data:
	default:
		h.logger.Warn("Broadcast channel full, dropping message")
	}
}

func (h *Hub) BroadcastStatus(serverID, status, message string) {
	update := NewStatusUpdate(serverID, status, message)
	data, err := json.Marshal(update)
	if err != nil {
		h.logger.Error("Failed to marshal status update", "error", err)
		return
	}
	h.Broadcast(data)

	if h.logStore != nil && message != "" {
		logMsg := fmt.Sprintf("[%s] %s", serverID, message)
		if err := h.logStore.AddLog("info", logMsg); err != nil {
			h.logger.Error("Failed to store status log entry", "error", err)
		}
	}
}

func (h *Hub) BroadcastLog(level LogLevel, message string) {
	logMsg := NewLogMessage(level, message)

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

func (h *Hub) BroadcastError(code, message, serverID string) {
	errMsg := NewErrorMessage(code, message, serverID)
	data, err := json.Marshal(errMsg)
	if err != nil {
		h.logger.Error("Failed to marshal error message", "error", err)
		return
	}
	h.Broadcast(data)
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
}
