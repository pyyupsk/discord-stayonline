// Package manager provides session management for Discord Gateway connections.
package manager

import "time"

// ConnectionStatus represents the current state of a server connection.
type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusError        ConnectionStatus = "error"
	StatusBackoff      ConnectionStatus = "backoff"
)

// SessionState represents the runtime state of a server connection.
// This is not persisted - it exists only while the service is running.
type SessionState struct {
	// ServerEntryID references the ServerEntry.ID this session belongs to
	ServerEntryID string

	// ConnectionStatus is the current state of the connection
	ConnectionStatus ConnectionStatus

	// LastError is the most recent error message (if any)
	LastError string

	// BackoffAttempt is the current backoff attempt count (resets on success)
	BackoffAttempt int

	// LastConnectTime is when the connection was last established
	LastConnectTime time.Time

	// SessionID is the Discord session ID for resume functionality
	SessionID string

	// Sequence is the last received sequence number from Gateway
	Sequence int
}

// NewSessionState creates a new session state for a server entry.
func NewSessionState(serverEntryID string) *SessionState {
	return &SessionState{
		ServerEntryID:    serverEntryID,
		ConnectionStatus: StatusDisconnected,
		BackoffAttempt:   0,
	}
}

// Reset clears the session state for a fresh connection attempt.
func (s *SessionState) Reset() {
	s.ConnectionStatus = StatusDisconnected
	s.LastError = ""
	s.BackoffAttempt = 0
	s.SessionID = ""
	s.Sequence = 0
}

// MarkConnecting updates the state to connecting.
func (s *SessionState) MarkConnecting() {
	s.ConnectionStatus = StatusConnecting
}

// MarkConnected updates the state to connected.
func (s *SessionState) MarkConnected(sessionID string) {
	s.ConnectionStatus = StatusConnected
	s.LastConnectTime = time.Now()
	s.SessionID = sessionID
	s.BackoffAttempt = 0 // Reset backoff on successful connection
	s.LastError = ""
}

// MarkError updates the state to error with a message.
func (s *SessionState) MarkError(err string) {
	s.ConnectionStatus = StatusError
	s.LastError = err
}

// MarkBackoff updates the state to backoff and increments the attempt counter.
func (s *SessionState) MarkBackoff() {
	s.ConnectionStatus = StatusBackoff
	s.BackoffAttempt++
}

// MarkDisconnected updates the state to disconnected.
func (s *SessionState) MarkDisconnected() {
	s.ConnectionStatus = StatusDisconnected
	s.LastError = ""
}

// UpdateSequence updates the last received sequence number.
func (s *SessionState) UpdateSequence(seq int) {
	if seq > 0 {
		s.Sequence = seq
	}
}
