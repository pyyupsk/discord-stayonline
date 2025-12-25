package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Gateway URL for Discord
const (
	GatewayURL     = "wss://gateway.discord.gg/?v=10&encoding=json"
	GatewayVersion = 10
)

// Client connection states
const (
	StateDisconnected = iota
	StateConnecting
	StateConnected
	StateClosed
)

// Common errors
var (
	ErrNotConnected   = errors.New("not connected to gateway")
	ErrAlreadyClosed  = errors.New("connection already closed")
	ErrFatalClose     = errors.New("fatal close code received")
	ErrInvalidSession = errors.New("session is invalid")
)

// Client represents a Discord Gateway WebSocket client.
type Client struct {
	token string

	conn   *websocket.Conn
	state  int
	mu     sync.RWMutex

	// Session state
	sessionID string
	sequence  int
	resumeURL string

	// Heartbeat management
	heartbeatInterval time.Duration
	heartbeatTicker   *time.Ticker
	lastHeartbeatAck  time.Time
	heartbeatStop     chan struct{}

	// Message handling
	readStop  chan struct{}
	readDone  chan struct{}

	// Status callbacks
	OnReady       func(sessionID string)
	OnDisconnect  func(code int, reason string)
	OnError       func(err error)
	OnStateChange func(state int)

	logger *slog.Logger
}

// NewClient creates a new Gateway client.
func NewClient(token string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		token:  token,
		state:  StateDisconnected,
		logger: logger.With("component", "gateway"),
	}
}

// Connect establishes a connection to the Discord Gateway.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.state == StateConnected {
		c.mu.Unlock()
		return nil
	}
	c.state = StateConnecting
	c.mu.Unlock()

	c.notifyStateChange(StateConnecting)

	// Connect to Gateway
	c.logger.Info("Connecting to Discord Gateway", "url", GatewayURL)

	conn, _, err := websocket.Dial(ctx, GatewayURL, &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		c.setState(StateDisconnected)
		return fmt.Errorf("dial gateway: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.heartbeatStop = make(chan struct{})
	c.readStop = make(chan struct{})
	c.readDone = make(chan struct{})
	c.mu.Unlock()

	// Start read loop
	go c.readLoop(ctx)

	return nil
}

// Close gracefully closes the Gateway connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateClosed || c.state == StateDisconnected {
		return nil
	}

	c.state = StateClosed

	// Stop heartbeat
	if c.heartbeatStop != nil {
		close(c.heartbeatStop)
	}

	// Stop read loop
	if c.readStop != nil {
		close(c.readStop)
	}

	// Wait for read loop to finish
	if c.readDone != nil {
		<-c.readDone
	}

	// Close WebSocket
	if c.conn != nil {
		c.conn.Close(websocket.StatusGoingAway, "client closing")
	}

	c.notifyStateChange(StateClosed)
	return nil
}

// SendIdentify sends the IDENTIFY payload to Discord.
func (c *Client) SendIdentify(ctx context.Context) error {
	return c.SendIdentifyWithStatus(ctx, "online")
}

// SendIdentifyWithStatus sends the IDENTIFY payload with a specific status.
func (c *Client) SendIdentifyWithStatus(ctx context.Context, status string) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	identify := struct {
		Op   int          `json:"op"`
		Data IdentifyData `json:"d"`
	}{
		Op: OpIdentify,
		Data: IdentifyData{
			Token: c.token,
			Properties: IdentifyProperties{
				OS:      "Windows",
				Browser: "Discord Client",
				Device:  "",
			},
			Presence: &PresenceData{
				Status:     status,
				Since:      new(int64), // 0 = not idle
				Activities: []Activity{},
				AFK:        false,
			},
		},
	}

	data, err := json.Marshal(identify)
	if err != nil {
		return fmt.Errorf("marshal identify: %w", err)
	}

	c.logger.Debug("Sending IDENTIFY", "status", status)
	return conn.Write(ctx, websocket.MessageText, data)
}

// SendHeartbeat sends a heartbeat to Discord.
func (c *Client) SendHeartbeat(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	seq := c.sequence
	c.mu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	heartbeat := struct {
		Op   int  `json:"op"`
		Data *int `json:"d"`
	}{
		Op: OpHeartbeat,
	}

	// Send sequence if we have one
	if seq > 0 {
		heartbeat.Data = &seq
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	c.logger.Debug("Sending heartbeat", "sequence", seq)
	return conn.Write(ctx, websocket.MessageText, data)
}

// SendPresenceUpdate sends a presence update to Discord.
// MANUAL REVIEW: Verify status values are valid (online, idle, dnd, invisible).
func (c *Client) SendPresenceUpdate(ctx context.Context, status string) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	presence := struct {
		Op   int          `json:"op"`
		Data PresenceData `json:"d"`
	}{
		Op: OpPresenceUpdate,
		Data: PresenceData{
			Since:      nil,
			Activities: []Activity{},
			Status:     status,
			AFK:        false,
		},
	}

	data, err := json.Marshal(presence)
	if err != nil {
		return fmt.Errorf("marshal presence: %w", err)
	}

	c.logger.Debug("Sending presence update", "status", status)
	return conn.Write(ctx, websocket.MessageText, data)
}

// SendVoiceStateUpdate joins or leaves a voice channel.
// MANUAL REVIEW: Verify guild_id and channel_id are valid Discord snowflakes.
func (c *Client) SendVoiceStateUpdate(ctx context.Context, guildID, channelID string, selfMute, selfDeaf bool) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	voiceState := struct {
		Op   int            `json:"op"`
		Data VoiceStateData `json:"d"`
	}{
		Op: OpVoiceStateUpdate,
		Data: VoiceStateData{
			GuildID:  guildID,
			SelfMute: selfMute,
			SelfDeaf: selfDeaf,
		},
	}

	// Set channel ID (nil to disconnect)
	if channelID != "" {
		voiceState.Data.ChannelID = &channelID
	}

	data, err := json.Marshal(voiceState)
	if err != nil {
		return fmt.Errorf("marshal voice state: %w", err)
	}

	c.logger.Debug("Sending voice state update", "guild_id", guildID, "channel_id", channelID)
	return conn.Write(ctx, websocket.MessageText, data)
}

// readLoop continuously reads messages from the Gateway.
func (c *Client) readLoop(ctx context.Context) {
	defer close(c.readDone)

	for {
		select {
		case <-c.readStop:
			return
		case <-ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		// Read with timeout
		readCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		_, data, err := conn.Read(readCtx)
		cancel()

		if err != nil {
			c.handleReadError(err)
			return
		}

		if err := c.handleMessage(ctx, data); err != nil {
			c.logger.Error("Error handling message", "error", err)
		}
	}
}

// handleMessage processes an incoming Gateway message.
func (c *Client) handleMessage(ctx context.Context, data []byte) error {
	var msg GatewayMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	// Update sequence if present
	if msg.Sequence != nil {
		c.mu.Lock()
		c.sequence = *msg.Sequence
		c.mu.Unlock()
	}

	switch msg.Op {
	case OpHello:
		return c.handleHello(ctx, msg.Data)

	case OpDispatch:
		return c.handleDispatch(ctx, msg.Type, msg.Data)

	case OpHeartbeatAck:
		c.handleHeartbeatAck()

	case OpReconnect:
		c.logger.Info("Received reconnect request from Gateway")
		c.handleReconnect()

	case OpInvalidSession:
		c.logger.Warn("Received invalid session from Gateway")
		c.handleInvalidSession(msg.Data)

	default:
		c.logger.Debug("Received unknown opcode", "op", msg.Op)
	}

	return nil
}

// handleHello processes the HELLO message and starts heartbeating.
func (c *Client) handleHello(ctx context.Context, data json.RawMessage) error {
	var hello HelloData
	if err := json.Unmarshal(data, &hello); err != nil {
		return fmt.Errorf("unmarshal hello: %w", err)
	}

	c.mu.Lock()
	c.heartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
	c.mu.Unlock()

	c.logger.Info("Received HELLO", "heartbeat_interval_ms", hello.HeartbeatInterval)

	// Start heartbeat goroutine
	go c.startHeartbeat(ctx)

	// Send IDENTIFY
	return c.SendIdentify(ctx)
}

// handleDispatch processes dispatch events.
func (c *Client) handleDispatch(ctx context.Context, eventType string, data json.RawMessage) error {
	c.logger.Debug("Received dispatch event", "type", eventType)

	switch eventType {
	case "READY":
		var ready ReadyData
		if err := json.Unmarshal(data, &ready); err != nil {
			return fmt.Errorf("unmarshal ready: %w", err)
		}

		c.mu.Lock()
		c.sessionID = ready.SessionID
		c.resumeURL = ready.ResumeURL
		c.state = StateConnected
		c.mu.Unlock()

		c.logger.Info("Connected to Discord Gateway", "session_id", ready.SessionID)
		c.notifyStateChange(StateConnected)

		if c.OnReady != nil {
			c.OnReady(ready.SessionID)
		}

	case "RESUMED":
		c.logger.Info("Session resumed successfully")
		c.setState(StateConnected)
		c.notifyStateChange(StateConnected)
	}

	return nil
}

// handleHeartbeatAck processes heartbeat acknowledgements.
func (c *Client) handleHeartbeatAck() {
	c.mu.Lock()
	c.lastHeartbeatAck = time.Now()
	c.mu.Unlock()
	c.logger.Debug("Received heartbeat ACK")
}

// handleReconnect handles a reconnect request from the Gateway.
func (c *Client) handleReconnect() {
	if c.OnDisconnect != nil {
		c.OnDisconnect(0, "reconnect requested")
	}
}

// handleInvalidSession handles an invalid session notification.
func (c *Client) handleInvalidSession(data json.RawMessage) {
	// Data is a boolean indicating if the session is resumable
	var resumable bool
	json.Unmarshal(data, &resumable)

	if !resumable {
		c.mu.Lock()
		c.sessionID = ""
		c.sequence = 0
		c.mu.Unlock()
	}

	if c.OnError != nil {
		c.OnError(ErrInvalidSession)
	}
}

// handleReadError processes read errors and determines if retryable.
func (c *Client) handleReadError(err error) {
	c.logger.Error("Read error", "error", err)

	// Check for close status
	closeStatus := websocket.CloseStatus(err)
	if closeStatus != -1 {
		c.logger.Info("Connection closed", "code", closeStatus)

		if IsFatalCloseCode(int(closeStatus)) {
			if c.OnError != nil {
				c.OnError(fmt.Errorf("%w: code %d", ErrFatalClose, closeStatus))
			}
		} else {
			if c.OnDisconnect != nil {
				c.OnDisconnect(int(closeStatus), "connection closed")
			}
		}
	} else {
		if c.OnDisconnect != nil {
			c.OnDisconnect(0, err.Error())
		}
	}

	c.setState(StateDisconnected)
}

// startHeartbeat runs the heartbeat loop.
func (c *Client) startHeartbeat(ctx context.Context) {
	c.mu.RLock()
	interval := c.heartbeatInterval
	stopChan := c.heartbeatStop
	c.mu.RUnlock()

	if interval == 0 {
		return
	}

	// Send initial heartbeat
	if err := c.SendHeartbeat(ctx); err != nil {
		c.logger.Error("Failed to send initial heartbeat", "error", err)
		return
	}

	c.mu.Lock()
	c.lastHeartbeatAck = time.Now()
	c.heartbeatTicker = time.NewTicker(interval)
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		if c.heartbeatTicker != nil {
			c.heartbeatTicker.Stop()
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-stopChan:
			return
		case <-ctx.Done():
			return
		case <-c.heartbeatTicker.C:
			// Check if we missed an ACK
			c.mu.RLock()
			lastAck := c.lastHeartbeatAck
			c.mu.RUnlock()

			if time.Since(lastAck) > interval*2 {
				c.logger.Warn("Missed heartbeat ACK, connection may be dead")
				if c.OnDisconnect != nil {
					c.OnDisconnect(0, "missed heartbeat ACK")
				}
				return
			}

			if err := c.SendHeartbeat(ctx); err != nil {
				c.logger.Error("Failed to send heartbeat", "error", err)
				return
			}
		}
	}
}

// setState updates the connection state.
func (c *Client) setState(state int) {
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()
}

// notifyStateChange calls the state change callback.
func (c *Client) notifyStateChange(state int) {
	if c.OnStateChange != nil {
		c.OnStateChange(state)
	}
}

// State returns the current connection state.
func (c *Client) State() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// SessionID returns the current session ID.
func (c *Client) SessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

// Sequence returns the current sequence number.
func (c *Client) Sequence() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sequence
}
