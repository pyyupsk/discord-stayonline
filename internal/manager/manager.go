package manager

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/pyyupsk/discord-stayonline/internal/config"
	"github.com/pyyupsk/discord-stayonline/internal/gateway"
	"github.com/pyyupsk/discord-stayonline/internal/webhook"
)

var (
	ErrServerNotFound     = errors.New("server not found")
	ErrTooManyConnections = errors.New("maximum 35 connections allowed")
	ErrTOSNotAcknowledged = errors.New("TOS not acknowledged")
	ErrAlreadyConnected   = errors.New("already connected")
	ErrNotConnected       = errors.New("not connected")
)

type SessionStore interface {
	SaveSession(state config.SessionState) error
	LoadSession(serverID string) (*config.SessionState, error)
	DeleteSession(serverID string) error
	UpdateSessionSequence(serverID string, sequence int) error
}

type SessionManager struct {
	token        string
	store        config.ConfigStore
	sessionStore SessionStore
	logger       *slog.Logger
	webhook      *webhook.Notifier

	sessions map[string]*Session
	mu       sync.RWMutex

	OnStatusChange func(serverID string, status ConnectionStatus, message string)

	ctx    context.Context
	cancel context.CancelFunc
}

type Session struct {
	serverEntry config.ServerEntry
	state       *SessionState
	client      *gateway.Client

	ctx    context.Context
	cancel context.CancelFunc

	stopReconnect chan struct{}
}

func NewSessionManager(token string, store config.ConfigStore, sessionStore SessionStore, webhookNotifier *webhook.Notifier, logger *slog.Logger) *SessionManager {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &SessionManager{
		token:        token,
		store:        store,
		sessionStore: sessionStore,
		webhook:      webhookNotifier,
		logger:       logger.With("component", "manager"),
		sessions:     make(map[string]*Session),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (m *SessionManager) Start() error {
	cfg, err := m.store.Load()
	if err != nil {
		return err
	}

	if !cfg.TOSAcknowledged {
		m.logger.Warn("TOS not acknowledged - skipping auto-connect")
		return nil
	}

	var toConnect []config.ServerEntry
	for _, server := range cfg.Servers {
		if server.ConnectOnStart {
			toConnect = append(toConnect, server)
		}
	}

	if len(toConnect) > 0 {
		go func() {
			for _, s := range toConnect {
				if err := m.Join(s.ID); err != nil {
					m.logger.Error("Failed to auto-connect", "server_id", s.ID, "error", err)
				}
			}
		}()
	}

	return nil
}

func (m *SessionManager) Stop() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for id, session := range m.sessions {
		m.logger.Info("Stopping session", "server_id", id)
		session.cancel()
		if session.client != nil {
			_ = session.client.Close()
		}
		if session.stopReconnect != nil {
			select {
			case <-session.stopReconnect:
			default:
				close(session.stopReconnect)
			}
		}
	}
}

func (m *SessionManager) Join(serverID string) error {
	cfg, err := m.store.Load()
	if err != nil {
		return err
	}
	if !cfg.TOSAcknowledged {
		return ErrTOSNotAcknowledged
	}

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

	if session, exists := m.sessions[serverID]; exists {
		if session.state.ConnectionStatus == StatusConnected ||
			session.state.ConnectionStatus == StatusConnecting {
			return ErrAlreadyConnected
		}
	}

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

	ctx, cancel := context.WithCancel(m.ctx)
	session := &Session{
		serverEntry:   *serverEntry,
		state:         NewSessionState(serverID),
		ctx:           ctx,
		cancel:        cancel,
		stopReconnect: make(chan struct{}),
	}

	m.sessions[serverID] = session

	go m.runSession(session)

	return nil
}

func (m *SessionManager) Rejoin(serverID string) error {
	m.mu.Lock()
	session, exists := m.sessions[serverID]
	m.mu.Unlock()

	if !exists {
		m.deleteSessionData(serverID)
		return m.Join(serverID)
	}

	if session.stopReconnect != nil {
		select {
		case <-session.stopReconnect:
		default:
			close(session.stopReconnect)
		}
	}

	if session.client != nil {
		_ = session.client.Close()
	}
	session.cancel()

	m.mu.Lock()
	delete(m.sessions, serverID)
	m.mu.Unlock()

	m.deleteSessionData(serverID)

	time.Sleep(100 * time.Millisecond)

	return m.Join(serverID)
}

func (m *SessionManager) deleteSessionData(serverID string) {
	if m.sessionStore == nil {
		return
	}
	if err := m.sessionStore.DeleteSession(serverID); err != nil {
		m.logger.Debug("Failed to delete session data", "server_id", serverID, "error", err)
	}
}

func (m *SessionManager) Exit(serverID string) error {
	m.mu.Lock()
	session, exists := m.sessions[serverID]
	if !exists {
		m.mu.Unlock()
		return ErrNotConnected
	}

	session.state.MarkDisconnected()
	m.mu.Unlock()

	m.notifyStatusChange(serverID, StatusDisconnected, "User requested exit")

	if session.stopReconnect != nil {
		select {
		case <-session.stopReconnect:
		default:
			close(session.stopReconnect)
		}
	}

	if session.client != nil {
		_ = session.client.Close()
	}
	session.cancel()

	m.mu.Lock()
	delete(m.sessions, serverID)
	m.mu.Unlock()

	m.deleteSessionData(serverID)

	m.logger.Info("Session exited", "server_id", serverID)
	return nil
}

func (m *SessionManager) GetStatus(serverID string) (ConnectionStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[serverID]
	if !exists {
		return StatusDisconnected, nil
	}
	return session.state.ConnectionStatus, nil
}

func (m *SessionManager) GetAllStatuses() map[string]ConnectionStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]ConnectionStatus)
	for id, session := range m.sessions {
		statuses[id] = session.state.ConnectionStatus
	}
	return statuses
}

func (m *SessionManager) runSession(session *Session) {
	serverID := session.serverEntry.ID
	m.logger.Info("Starting session", "server_id", serverID)

	for {
		if m.shouldStopSession(session) {
			return
		}

		session.state.MarkConnecting()
		m.notifyStatusChange(serverID, StatusConnecting, "Connecting...")

		status := m.loadGlobalStatus()
		if status == "" {
			return
		}

		client := m.createAndConfigureClient(session, status)

		if err := client.Connect(session.ctx); err != nil {
			if m.handleConnectionError(session, err) {
				continue
			}
			return
		}

		if m.waitForDisconnection(session, client) {
			return
		}
	}
}

func (m *SessionManager) shouldStopSession(session *Session) bool {
	select {
	case <-session.ctx.Done():
		return true
	case <-session.stopReconnect:
		return true
	default:
		return false
	}
}

func (m *SessionManager) loadGlobalStatus() string {
	cfg, err := m.store.Load()
	if err != nil {
		m.logger.Error("Failed to load config", "error", err)
		return ""
	}
	status := string(cfg.Status)
	if status == "" {
		status = "online"
	}
	return status
}

func (m *SessionManager) createAndConfigureClient(session *Session, status string) *gateway.Client {
	serverID := session.serverEntry.ID
	client := gateway.NewClient(m.token, m.logger)
	client.SetStatus(status)
	session.client = client

	m.tryResumeSession(client, serverID)
	m.setupClientCallbacks(session, client)

	return client
}

func (m *SessionManager) tryResumeSession(client *gateway.Client, serverID string) {
	if m.sessionStore == nil {
		return
	}
	savedSession, err := m.sessionStore.LoadSession(serverID)
	if err != nil || savedSession == nil {
		return
	}
	client.SetResumeData(savedSession.SessionID, savedSession.Sequence, savedSession.ResumeURL)
	m.logger.Info("Attempting session resume", "server_id", serverID, "session_id", savedSession.SessionID)
}

func (m *SessionManager) setupClientCallbacks(session *Session, client *gateway.Client) {
	serverID := session.serverEntry.ID

	client.OnReady = func(sessionID string) {
		wasReconnecting := session.state.BackoffAttempt > 0

		session.state.MarkConnected(sessionID)
		m.notifyStatusChange(serverID, StatusConnected, "Connected")
		m.saveSessionState(serverID, client)
		m.joinVoiceChannel(session, client)

		if wasReconnecting && m.webhook != nil {
			go m.webhook.NotifyUp(
				serverID,
				session.serverEntry.GuildID,
				session.serverEntry.ChannelID,
			)
		}
	}

	client.OnDisconnect = func(_ int, reason string) {
		session.state.MarkError(reason)
		m.notifyStatusChange(serverID, StatusError, reason)
	}

	client.OnError = func(err error) {
		session.state.MarkError(err.Error())
		m.notifyStatusChange(serverID, StatusError, err.Error())
		m.handleInvalidSession(serverID, err)
		m.handleFatalError(session, serverID, err)
	}
}

func (m *SessionManager) saveSessionState(serverID string, client *gateway.Client) {
	if m.sessionStore == nil {
		return
	}
	sid, seq, resumeURL := client.GetSessionData()
	if sid == "" || resumeURL == "" {
		return
	}
	state := config.SessionState{
		ServerID:  serverID,
		SessionID: sid,
		Sequence:  seq,
		ResumeURL: resumeURL,
	}
	if err := m.sessionStore.SaveSession(state); err != nil {
		m.logger.Error("Failed to save session state", "server_id", serverID, "error", err)
	}
}

func (m *SessionManager) joinVoiceChannel(session *Session, client *gateway.Client) {
	if session.serverEntry.ChannelID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(session.ctx, 5*time.Second)
	defer cancel()
	_ = client.SendVoiceStateUpdate(ctx, session.serverEntry.GuildID, session.serverEntry.ChannelID, true, true)
}

func (m *SessionManager) handleInvalidSession(serverID string, err error) {
	if !errors.Is(err, gateway.ErrInvalidSession) {
		return
	}
	m.logger.Info("Clearing invalid session data", "server_id", serverID)
	if m.sessionStore != nil {
		if delErr := m.sessionStore.DeleteSession(serverID); delErr != nil {
			m.logger.Error("Failed to delete session", "server_id", serverID, "error", delErr)
		}
	}
}

func (m *SessionManager) handleFatalError(session *Session, serverID string, err error) {
	if !errors.Is(err, gateway.ErrFatalClose) {
		return
	}
	m.logger.Error("Fatal Gateway error - stopping reconnection", "server_id", serverID, "error", err)

	if m.webhook != nil {
		go m.webhook.NotifyDown(
			serverID,
			session.serverEntry.GuildID,
			session.serverEntry.ChannelID,
			err.Error(),
		)
	}

	select {
	case <-session.stopReconnect:
	default:
		close(session.stopReconnect)
	}
}

func (m *SessionManager) handleConnectionError(session *Session, err error) bool {
	serverID := session.serverEntry.ID
	session.state.MarkError(err.Error())
	m.notifyStatusChange(serverID, StatusError, err.Error())

	session.state.MarkBackoff()
	m.notifyStatusChange(serverID, StatusBackoff, "Waiting to reconnect...")

	delay := gateway.CalculateBackoff(session.state.BackoffAttempt)
	m.logger.Info("Waiting before reconnect", "server_id", serverID, "delay", delay)

	select {
	case <-session.ctx.Done():
		return false
	case <-session.stopReconnect:
		return false
	case <-time.After(delay):
		return true
	}
}

func (m *SessionManager) waitForDisconnection(session *Session, client *gateway.Client) bool {
	disconnected := client.Disconnected()
	select {
	case <-session.ctx.Done():
		_ = client.Close()
		return true
	case <-session.stopReconnect:
		_ = client.Close()
		return true
	case <-disconnected:
		serverID := session.serverEntry.ID
		m.logger.Info("Connection lost, will reconnect", "server_id", serverID)
		_ = client.Close()

		session.state.MarkBackoff()
		m.notifyStatusChange(serverID, StatusBackoff, "Reconnecting...")
		delay := gateway.CalculateBackoff(session.state.BackoffAttempt)
		m.logger.Info("Waiting before reconnect", "server_id", serverID, "delay", delay)

		if m.webhook != nil {
			go m.webhook.NotifyReconnecting(serverID, session.state.BackoffAttempt, delay)
		}

		select {
		case <-session.ctx.Done():
			return true
		case <-session.stopReconnect:
			return true
		case <-time.After(delay):
			return false
		}
	}
}

func (m *SessionManager) notifyStatusChange(serverID string, status ConnectionStatus, message string) {
	if m.OnStatusChange != nil {
		m.OnStatusChange(serverID, status, message)
	}
}
