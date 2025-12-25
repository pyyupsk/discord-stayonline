// Package manager provides session management for Discord Gateway connections.
package manager

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/pyyupsk/discord-stayonline/internal/config"
	"github.com/pyyupsk/discord-stayonline/internal/gateway"
)

// Common errors
var (
	ErrServerNotFound     = errors.New("server not found")
	ErrTooManyConnections = errors.New("maximum 15 connections allowed")
	ErrTOSNotAcknowledged = errors.New("TOS not acknowledged")
	ErrAlreadyConnected   = errors.New("already connected")
	ErrNotConnected       = errors.New("not connected")
)

// SessionManager manages multiple Gateway connections.
type SessionManager struct {
	token  string
	store  config.ConfigStore
	logger *slog.Logger

	sessions map[string]*Session
	mu       sync.RWMutex

	// Status callback for WebSocket hub broadcast
	OnStatusChange func(serverID string, status ConnectionStatus, message string)

	ctx    context.Context
	cancel context.CancelFunc
}

// Session represents a single Gateway connection.
type Session struct {
	serverEntry config.ServerEntry
	state       *SessionState
	client      *gateway.Client

	ctx    context.Context
	cancel context.CancelFunc

	stopReconnect chan struct{}
	reconnecting  bool
}

// NewSessionManager creates a new session manager.
func NewSessionManager(token string, store config.ConfigStore, logger *slog.Logger) *SessionManager {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &SessionManager{
		token:    token,
		store:    store,
		logger:   logger.With("component", "manager"),
		sessions: make(map[string]*Session),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start initializes the session manager and auto-connects configured servers.
func (m *SessionManager) Start() error {
	cfg, err := m.store.Load()
	if err != nil {
		return err
	}

	// Check TOS acknowledgment
	if !cfg.TOSAcknowledged {
		m.logger.Warn("TOS not acknowledged - skipping auto-connect")
		return nil
	}

	// Collect servers to auto-connect
	var toConnect []config.ServerEntry
	for _, server := range cfg.Servers {
		if server.ConnectOnStart {
			toConnect = append(toConnect, server)
		}
	}

	// Auto-connect with staggered delays to avoid Discord rate limits
	// Discord limits IDENTIFY to ~1 per 5 seconds per token
	// Initial delay allows old container sessions to close during rolling deploys
	if len(toConnect) > 0 {
		go func() {
			// Wait for old container to fully stop (rolling deploy overlap)
			time.Sleep(5 * time.Second)

			for i, s := range toConnect {
				if i > 0 {
					time.Sleep(2 * time.Second)
				}
				if err := m.Join(s.ID); err != nil {
					m.logger.Error("Failed to auto-connect", "server_id", s.ID, "error", err)
				}
			}
		}()
	}

	return nil
}

// Stop gracefully closes all connections.
func (m *SessionManager) Stop() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for id, session := range m.sessions {
		m.logger.Info("Stopping session", "server_id", id)
		session.cancel()
		if session.client != nil {
			session.client.Close()
		}
		if session.stopReconnect != nil {
			close(session.stopReconnect)
		}
	}
}

// Join starts a connection for a server entry.
func (m *SessionManager) Join(serverID string) error {
	// Check TOS acknowledgment first
	cfg, err := m.store.Load()
	if err != nil {
		return err
	}
	if !cfg.TOSAcknowledged {
		return ErrTOSNotAcknowledged
	}

	// Find server entry
	var serverEntry *config.ServerEntry
	for i := range cfg.Servers {
		if cfg.Servers[i].ID == serverID {
			serverEntry = &cfg.Servers[i]
			break
		}
	}
	if serverEntry == nil {
		return ErrServerNotFound
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already connected
	if session, exists := m.sessions[serverID]; exists {
		if session.state.ConnectionStatus == StatusConnected ||
			session.state.ConnectionStatus == StatusConnecting {
			return ErrAlreadyConnected
		}
	}

	// Check connection limit
	activeCount := 0
	for _, s := range m.sessions {
		if s.state.ConnectionStatus == StatusConnected ||
			s.state.ConnectionStatus == StatusConnecting {
			activeCount++
		}
	}
	if activeCount >= config.MaxServerEntries {
		return ErrTooManyConnections
	}

	// Create session
	ctx, cancel := context.WithCancel(m.ctx)
	session := &Session{
		serverEntry:   *serverEntry,
		state:         NewSessionState(serverID),
		ctx:           ctx,
		cancel:        cancel,
		stopReconnect: make(chan struct{}),
	}

	m.sessions[serverID] = session

	// Start connection in goroutine
	go m.runSession(session)

	return nil
}

// Rejoin closes existing connection and reconnects.
func (m *SessionManager) Rejoin(serverID string) error {
	m.mu.Lock()
	session, exists := m.sessions[serverID]
	m.mu.Unlock()

	if !exists {
		// Just join if not exists
		return m.Join(serverID)
	}

	// Stop reconnection loop
	if session.stopReconnect != nil && !session.reconnecting {
		close(session.stopReconnect)
	}

	// Close existing connection
	if session.client != nil {
		session.client.Close()
	}
	session.cancel()

	// Remove old session
	m.mu.Lock()
	delete(m.sessions, serverID)
	m.mu.Unlock()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Start new connection
	return m.Join(serverID)
}

// Exit closes a connection and stops reconnection.
func (m *SessionManager) Exit(serverID string) error {
	m.mu.Lock()
	session, exists := m.sessions[serverID]
	if !exists {
		m.mu.Unlock()
		return ErrNotConnected
	}

	// Mark as disconnected first
	session.state.MarkDisconnected()
	m.mu.Unlock()

	// Notify status change
	m.notifyStatusChange(serverID, StatusDisconnected, "User requested exit")

	// Stop reconnection
	if session.stopReconnect != nil {
		select {
		case <-session.stopReconnect:
			// Already closed
		default:
			close(session.stopReconnect)
		}
	}

	// Close connection
	if session.client != nil {
		session.client.Close()
	}
	session.cancel()

	// Remove session
	m.mu.Lock()
	delete(m.sessions, serverID)
	m.mu.Unlock()

	m.logger.Info("Session exited", "server_id", serverID)
	return nil
}

// GetStatus returns the current status of a session.
func (m *SessionManager) GetStatus(serverID string) (ConnectionStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[serverID]
	if !exists {
		return StatusDisconnected, nil
	}
	return session.state.ConnectionStatus, nil
}

// GetAllStatuses returns status for all sessions.
func (m *SessionManager) GetAllStatuses() map[string]ConnectionStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]ConnectionStatus)
	for id, session := range m.sessions {
		statuses[id] = session.state.ConnectionStatus
	}
	return statuses
}

// runSession runs a Gateway connection session.
func (m *SessionManager) runSession(session *Session) {
	serverID := session.serverEntry.ID
	m.logger.Info("Starting session", "server_id", serverID)

	for {
		select {
		case <-session.ctx.Done():
			return
		case <-session.stopReconnect:
			return
		default:
		}

		// Update status
		session.state.MarkConnecting()
		m.notifyStatusChange(serverID, StatusConnecting, "Connecting...")

		// Load config to get global status
		cfg, err := m.store.Load()
		if err != nil {
			m.logger.Error("Failed to load config", "error", err)
			return
		}
		status := string(cfg.Status)
		if status == "" {
			status = "online"
		}

		// Create Gateway client
		client := gateway.NewClient(m.token, m.logger)
		client.SetStatus(status)
		session.client = client

		// Set up callbacks
		client.OnReady = func(sessionID string) {
			session.state.MarkConnected(sessionID)
			m.notifyStatusChange(serverID, StatusConnected, "Connected")

			// Join voice channel if configured (presence is already set in IDENTIFY)
			if session.serverEntry.ChannelID != "" {
				ctx, cancel := context.WithTimeout(session.ctx, 5*time.Second)
				_ = client.SendVoiceStateUpdate(ctx, session.serverEntry.GuildID, session.serverEntry.ChannelID, true, true)
				cancel()
			}
		}

		client.OnDisconnect = func(code int, reason string) {
			session.state.MarkError(reason)
			m.notifyStatusChange(serverID, StatusError, reason)
		}

		client.OnError = func(err error) {
			session.state.MarkError(err.Error())
			m.notifyStatusChange(serverID, StatusError, err.Error())

			// Check for fatal errors
			if errors.Is(err, gateway.ErrFatalClose) {
				m.logger.Error("Fatal Gateway error - stopping reconnection", "server_id", serverID, "error", err)
				select {
				case <-session.stopReconnect:
				default:
					close(session.stopReconnect)
				}
			}
		}

		// Connect
		if err := client.Connect(session.ctx); err != nil {
			session.state.MarkError(err.Error())
			m.notifyStatusChange(serverID, StatusError, err.Error())

			// Wait for reconnect with backoff
			session.state.MarkBackoff()
			m.notifyStatusChange(serverID, StatusBackoff, "Waiting to reconnect...")

			delay := gateway.CalculateBackoff(session.state.BackoffAttempt)
			m.logger.Info("Waiting before reconnect", "server_id", serverID, "delay", delay)

			select {
			case <-session.ctx.Done():
				return
			case <-session.stopReconnect:
				return
			case <-time.After(delay):
				continue
			}
		}

		// Wait for disconnection or stop signal
		select {
		case <-session.ctx.Done():
			client.Close()
			return
		case <-session.stopReconnect:
			client.Close()
			return
		}
	}
}

// notifyStatusChange calls the status change callback.
func (m *SessionManager) notifyStatusChange(serverID string, status ConnectionStatus, message string) {
	if m.OnStatusChange != nil {
		m.OnStatusChange(serverID, status, message)
	}
}
