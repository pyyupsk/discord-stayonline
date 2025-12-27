package manager

import "time"

type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusError        ConnectionStatus = "error"
	StatusBackoff      ConnectionStatus = "backoff"
)

type SessionState struct {
	ServerEntryID    string
	ConnectionStatus ConnectionStatus
	LastError        string
	BackoffAttempt   int
	LastConnectTime  time.Time
	SessionID        string
	Sequence         int
}

func NewSessionState(serverEntryID string) *SessionState {
	return &SessionState{
		ServerEntryID:    serverEntryID,
		ConnectionStatus: StatusDisconnected,
		BackoffAttempt:   0,
	}
}

func (s *SessionState) Reset() {
	s.ConnectionStatus = StatusDisconnected
	s.LastError = ""
	s.BackoffAttempt = 0
	s.SessionID = ""
	s.Sequence = 0
}

func (s *SessionState) MarkConnecting() {
	s.ConnectionStatus = StatusConnecting
}

func (s *SessionState) MarkConnected(sessionID string) {
	s.ConnectionStatus = StatusConnected
	s.LastConnectTime = time.Now()
	s.SessionID = sessionID
	s.BackoffAttempt = 0
	s.LastError = ""
}

func (s *SessionState) MarkError(err string) {
	s.ConnectionStatus = StatusError
	s.LastError = err
}

func (s *SessionState) MarkBackoff() {
	s.ConnectionStatus = StatusBackoff
	s.BackoffAttempt++
}

func (s *SessionState) MarkDisconnected() {
	s.ConnectionStatus = StatusDisconnected
	s.LastError = ""
}

func (s *SessionState) UpdateSequence(seq int) {
	if seq > 0 {
		s.Sequence = seq
	}
}
