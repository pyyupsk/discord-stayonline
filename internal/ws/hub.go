// Package ws provides WebSocket hub for real-time UI updates.
package ws

import "time"

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
