package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

const (
	GatewayURL     = "wss://gateway.discord.gg/?v=10&encoding=json"
	GatewayVersion = 10
)

var (
	clientCounter uint64
	osList        = []string{"Windows", "Linux", "Mac OS X", "iOS", "Android"}
	browserList   = []string{"Discord Client", "Chrome", "Firefox", "Safari", "Edge", "Opera", "Brave"}
)

func getClientProperties(index int) (os, browser, device string) {
	os = osList[index%len(osList)]
	browser = browserList[(index/len(osList))%len(browserList)]
	if index >= len(osList)*len(browserList) {
		device = fmt.Sprintf("device-%d", index)
	}
	return
}

const (
	StateDisconnected = iota
	StateConnecting
	StateConnected
	StateClosed
)

var (
	ErrNotConnected   = errors.New("not connected to gateway")
	ErrAlreadyClosed  = errors.New("connection already closed")
	ErrFatalClose     = errors.New("fatal close code received")
	ErrInvalidSession = errors.New("session is invalid")
)

type Client struct {
	token       string
	status      string
	clientIndex int

	conn  *websocket.Conn
	state int
	mu    sync.RWMutex

	sessionID        string
	sequence         int
	resumeURL        string
	resumeSessionID  string
	resumeSequence   int
	resumeGatewayURL string

	heartbeatInterval time.Duration
	heartbeatTicker   *time.Ticker
	lastHeartbeatAck  time.Time
	heartbeatStop     chan struct{}

	readStop     chan struct{}
	readDone     chan struct{}
	disconnected chan struct{}

	OnReady       func(sessionID string)
	OnDisconnect  func(code int, reason string)
	OnError       func(err error)
	OnStateChange func(state int)

	logger *slog.Logger
}

func NewClient(token string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	index := int(atomic.AddUint64(&clientCounter, 1) - 1)
	return &Client{
		token:       token,
		clientIndex: index,
		status:      "online",
		state:       StateDisconnected,
		logger:      logger.With("component", "gateway"),
	}
}

func (c *Client) SetStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

func (c *Client) SetResumeData(sessionID string, sequence int, resumeURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resumeSessionID = sessionID
	c.resumeSequence = sequence
	c.resumeGatewayURL = resumeURL
}

func (c *Client) GetSessionData() (sessionID string, sequence int, resumeURL string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID, c.sequence, c.resumeURL
}

func (c *Client) ClearResumeData() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resumeSessionID = ""
	c.resumeSequence = 0
	c.resumeGatewayURL = ""
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.state == StateConnected {
		c.mu.Unlock()
		return nil
	}
	c.state = StateConnecting
	resumeURL := c.resumeGatewayURL
	c.mu.Unlock()

	c.notifyStateChange(StateConnecting)

	gatewayURL := GatewayURL
	if resumeURL != "" {
		gatewayURL = resumeURL + "/?v=10&encoding=json"
		c.logger.Info("Resuming Discord Gateway session", "url", gatewayURL)
	} else {
		c.logger.Info("Connecting to Discord Gateway", "url", gatewayURL)
	}

	conn, _, err := websocket.Dial(ctx, gatewayURL, &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		c.setState(StateDisconnected)
		return fmt.Errorf("dial gateway: %w", err)
	}

	conn.SetReadLimit(1024 * 1024)

	c.mu.Lock()
	c.conn = conn
	c.heartbeatStop = make(chan struct{})
	c.readStop = make(chan struct{})
	c.readDone = make(chan struct{})
	c.disconnected = make(chan struct{})
	c.mu.Unlock()

	go c.readLoop(ctx)

	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()

	if c.state == StateClosed || c.state == StateDisconnected {
		c.mu.Unlock()
		return nil
	}

	c.state = StateClosed

	if c.heartbeatStop != nil {
		close(c.heartbeatStop)
		c.heartbeatStop = nil
	}

	if c.readStop != nil {
		close(c.readStop)
		c.readStop = nil
	}

	conn := c.conn
	c.conn = nil
	readDone := c.readDone

	c.mu.Unlock()

	if conn != nil {
		_ = conn.Close(websocket.StatusGoingAway, "client closing")
	}

	if readDone != nil {
		select {
		case <-readDone:
		case <-time.After(5 * time.Second):
		}
	}

	c.mu.Lock()
	c.disconnected = nil
	c.mu.Unlock()

	c.notifyStateChange(StateClosed)
	return nil
}

func (c *Client) SendIdentify(ctx context.Context) error {
	c.mu.RLock()
	status := c.status
	c.mu.RUnlock()
	if status == "" {
		status = "online"
	}
	return c.SendIdentifyWithStatus(ctx, status)
}

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
			Properties: func() IdentifyProperties {
				os, browser, device := getClientProperties(c.clientIndex)
				return IdentifyProperties{
					OS:      os,
					Browser: browser,
					Device:  device,
				}
			}(),
			Presence: &PresenceData{
				Status:     status,
				Since:      new(int64),
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

func (c *Client) sendResume(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	sessionID := c.resumeSessionID
	seq := c.resumeSequence
	c.mu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	resume := struct {
		Op   int        `json:"op"`
		Data ResumeData `json:"d"`
	}{
		Op: OpResume,
		Data: ResumeData{
			Token:     c.token,
			SessionID: sessionID,
			Sequence:  seq,
		},
	}

	data, err := json.Marshal(resume)
	if err != nil {
		return fmt.Errorf("marshal resume: %w", err)
	}

	c.logger.Info("Sending RESUME", "session_id", sessionID, "sequence", seq)
	return conn.Write(ctx, websocket.MessageText, data)
}

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

func (c *Client) readLoop(ctx context.Context) {
	defer func() {
		c.mu.Lock()
		if c.readDone != nil {
			close(c.readDone)
			c.readDone = nil
		}
		if c.heartbeatStop != nil {
			close(c.heartbeatStop)
			c.heartbeatStop = nil
		}
		if c.disconnected != nil {
			close(c.disconnected)
			c.disconnected = nil
		}
		c.mu.Unlock()
	}()

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

func (c *Client) handleMessage(ctx context.Context, data []byte) error {
	var msg GatewayMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

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

	case OpHeartbeat:
		c.logger.Debug("Received heartbeat request from Gateway")
		if err := c.SendHeartbeat(ctx); err != nil {
			c.logger.Error("Failed to send requested heartbeat", "error", err)
		}

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

func (c *Client) handleHello(ctx context.Context, data json.RawMessage) error {
	var hello HelloData
	if err := json.Unmarshal(data, &hello); err != nil {
		return fmt.Errorf("unmarshal hello: %w", err)
	}

	c.mu.Lock()
	c.heartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
	resumeSessionID := c.resumeSessionID
	c.mu.Unlock()

	c.logger.Info("Received HELLO", "heartbeat_interval_ms", hello.HeartbeatInterval)

	go c.startHeartbeat(ctx)

	if resumeSessionID != "" {
		return c.sendResume(ctx)
	}
	return c.SendIdentify(ctx)
}

func (c *Client) handleDispatch(_ context.Context, eventType string, data json.RawMessage) error {
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
		c.mu.Lock()
		c.sessionID = c.resumeSessionID
		c.sequence = c.resumeSequence
		c.state = StateConnected
		sessionID := c.sessionID
		c.mu.Unlock()

		c.logger.Info("Session resumed successfully", "session_id", sessionID)
		c.notifyStateChange(StateConnected)

		if c.OnReady != nil {
			c.OnReady(sessionID)
		}
	}

	return nil
}

func (c *Client) handleHeartbeatAck() {
	c.mu.Lock()
	c.lastHeartbeatAck = time.Now()
	c.mu.Unlock()
	c.logger.Debug("Received heartbeat ACK")
}

func (c *Client) handleReconnect() {
	if c.OnDisconnect != nil {
		c.OnDisconnect(0, "reconnect requested")
	}
}

func (c *Client) handleInvalidSession(data json.RawMessage) {
	var resumable bool
	_ = json.Unmarshal(data, &resumable)

	if !resumable {
		c.mu.Lock()
		c.sessionID = ""
		c.sequence = 0
		c.resumeSessionID = ""
		c.resumeSequence = 0
		c.resumeGatewayURL = ""
		conn := c.conn
		c.mu.Unlock()
		c.logger.Info("Session invalidated, closing connection to re-identify")

		if c.OnError != nil {
			c.OnError(ErrInvalidSession)
		}

		if conn != nil {
			_ = conn.Close(websocket.StatusNormalClosure, "invalid session - will reconnect")
		}
		return
	}

	if c.OnError != nil {
		c.OnError(ErrInvalidSession)
	}
}

func (c *Client) handleReadError(err error) {
	c.logger.Error("Read error", "error", err)

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

func (c *Client) startHeartbeat(ctx context.Context) {
	c.mu.RLock()
	interval := c.heartbeatInterval
	stopChan := c.heartbeatStop
	c.mu.RUnlock()

	if interval == 0 {
		return
	}

	jitterDuration := randomJitter(interval * 2)
	c.logger.Debug("Waiting before first heartbeat", "jitter", jitterDuration)

	select {
	case <-stopChan:
		return
	case <-ctx.Done():
		return
	case <-time.After(jitterDuration):
	}

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
			c.mu.RLock()
			lastAck := c.lastHeartbeatAck
			c.mu.RUnlock()

			if time.Since(lastAck) > interval*2 {
				c.logger.Warn("Missed heartbeat ACK, connection may be dead")
				c.mu.RLock()
				conn := c.conn
				c.mu.RUnlock()
				if conn != nil {
					_ = conn.Close(websocket.StatusProtocolError, "missed heartbeat ACK")
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

func (c *Client) setState(state int) {
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()
}

func (c *Client) notifyStateChange(state int) {
	if c.OnStateChange != nil {
		c.OnStateChange(state)
	}
}

func (c *Client) State() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Client) SessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

func (c *Client) Sequence() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sequence
}

func (c *Client) Disconnected() <-chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.disconnected
}
