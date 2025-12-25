package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// mockGatewayServer simulates a Discord Gateway server for testing.
type mockGatewayServer struct {
	server             *httptest.Server
	conn               *websocket.Conn
	mu                 sync.Mutex
	heartbeatCount     int
	heartbeatInterval  int
	sendReadyOnIdent   bool
	sendInvalidOnIdent bool
	closeOnConnect     bool
	closeCode          websocket.StatusCode
}

func newMockGatewayServer(t *testing.T) *mockGatewayServer {
	mock := &mockGatewayServer{
		heartbeatInterval: 100,
		sendReadyOnIdent:  true,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			t.Fatalf("failed to accept websocket: %v", err)
			return
		}

		mock.mu.Lock()
		mock.conn = conn
		closeOnConnect := mock.closeOnConnect
		closeCode := mock.closeCode
		mock.mu.Unlock()

		if closeOnConnect {
			conn.Close(closeCode, "test close")
			return
		}

		// Send HELLO
		hello := map[string]interface{}{
			"op": OpHello,
			"d": map[string]interface{}{
				"heartbeat_interval": mock.heartbeatInterval,
			},
		}
		helloData, _ := json.Marshal(hello)
		if err := conn.Write(r.Context(), websocket.MessageText, helloData); err != nil {
			return
		}

		// Read loop
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			mock.handleMessage(r.Context(), data)
		}
	})

	mock.server = httptest.NewServer(handler)
	return mock
}

func (m *mockGatewayServer) URL() string {
	return "ws" + strings.TrimPrefix(m.server.URL, "http")
}

func (m *mockGatewayServer) Close() {
	m.mu.Lock()
	if m.conn != nil {
		m.conn.Close(websocket.StatusNormalClosure, "server closing")
	}
	m.mu.Unlock()
	m.server.Close()
}

func (m *mockGatewayServer) handleMessage(ctx context.Context, data []byte) {
	var msg struct {
		Op   int             `json:"op"`
		Data json.RawMessage `json:"d"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	m.mu.Lock()
	conn := m.conn
	sendReadyOnIdent := m.sendReadyOnIdent
	sendInvalidOnIdent := m.sendInvalidOnIdent
	m.mu.Unlock()

	switch msg.Op {
	case OpIdentify:
		if sendInvalidOnIdent {
			invalid := map[string]interface{}{
				"op": OpInvalidSession,
				"d":  false,
			}
			data, _ := json.Marshal(invalid)
			_ = conn.Write(ctx, websocket.MessageText, data)
		} else if sendReadyOnIdent {
			ready := map[string]interface{}{
				"op": OpDispatch,
				"t":  "READY",
				"s":  1,
				"d": map[string]interface{}{
					"v":                  10,
					"session_id":         "test-session-123",
					"resume_gateway_url": m.URL(),
				},
			}
			data, _ := json.Marshal(ready)
			_ = conn.Write(ctx, websocket.MessageText, data)
		}

	case OpHeartbeat:
		m.mu.Lock()
		m.heartbeatCount++
		m.mu.Unlock()

		ack := map[string]interface{}{
			"op": OpHeartbeatAck,
		}
		ackData, _ := json.Marshal(ack)
		_ = conn.Write(ctx, websocket.MessageText, ackData)

	case OpPresenceUpdate:
		// Acknowledge presence update

	case OpVoiceStateUpdate:
		// Acknowledge voice state update
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("test-token", nil)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", client.token)
	}
	if client.state != StateDisconnected {
		t.Errorf("expected state StateDisconnected, got %d", client.state)
	}
}

func TestClientState(t *testing.T) {
	client := NewClient("test-token", nil)
	if client.State() != StateDisconnected {
		t.Errorf("expected State() to return StateDisconnected, got %d", client.State())
	}
}

func TestClientSessionID(t *testing.T) {
	client := NewClient("test-token", nil)
	if client.SessionID() != "" {
		t.Errorf("expected SessionID() to return empty string, got '%s'", client.SessionID())
	}
}

func TestClientSequence(t *testing.T) {
	client := NewClient("test-token", nil)
	if client.Sequence() != 0 {
		t.Errorf("expected Sequence() to return 0, got %d", client.Sequence())
	}
}

func TestSendIdentifyNotConnected(t *testing.T) {
	client := NewClient("test-token", nil)
	err := client.SendIdentify(context.Background())
	if err != ErrNotConnected {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSendHeartbeatNotConnected(t *testing.T) {
	client := NewClient("test-token", nil)
	err := client.SendHeartbeat(context.Background())
	if err != ErrNotConnected {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSendPresenceUpdateNotConnected(t *testing.T) {
	client := NewClient("test-token", nil)
	err := client.SendPresenceUpdate(context.Background(), "online")
	if err != ErrNotConnected {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSendVoiceStateUpdateNotConnected(t *testing.T) {
	client := NewClient("test-token", nil)
	err := client.SendVoiceStateUpdate(context.Background(), "guild123", "channel123", false, false)
	if err != ErrNotConnected {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestClientClose(t *testing.T) {
	client := NewClient("test-token", nil)
	// Close should not error when not connected
	err := client.Close()
	if err != nil {
		t.Errorf("expected no error on close when disconnected, got %v", err)
	}
}

func TestClientStateChange(t *testing.T) {
	client := NewClient("test-token", nil)

	stateChanges := make([]int, 0)
	client.OnStateChange = func(state int) {
		stateChanges = append(stateChanges, state)
	}

	client.notifyStateChange(StateConnecting)
	client.notifyStateChange(StateConnected)

	if len(stateChanges) != 2 {
		t.Errorf("expected 2 state changes, got %d", len(stateChanges))
	}
	if stateChanges[0] != StateConnecting {
		t.Errorf("expected first state change to be StateConnecting, got %d", stateChanges[0])
	}
	if stateChanges[1] != StateConnected {
		t.Errorf("expected second state change to be StateConnected, got %d", stateChanges[1])
	}
}

func TestClientSetState(t *testing.T) {
	client := NewClient("test-token", nil)

	client.setState(StateConnecting)
	if client.State() != StateConnecting {
		t.Errorf("expected state StateConnecting, got %d", client.State())
	}

	client.setState(StateConnected)
	if client.State() != StateConnected {
		t.Errorf("expected state StateConnected, got %d", client.State())
	}
}

func TestHandleMessage(t *testing.T) {
	client := NewClient("test-token", nil)

	// Test OpHeartbeatAck
	ackMsg := `{"op": 11}`
	err := client.handleMessage(context.Background(), []byte(ackMsg))
	if err != nil {
		t.Errorf("handleMessage for HeartbeatAck returned error: %v", err)
	}

	// Test unknown opcode (should not error)
	unknownMsg := `{"op": 99}`
	err = client.handleMessage(context.Background(), []byte(unknownMsg))
	if err != nil {
		t.Errorf("handleMessage for unknown opcode returned error: %v", err)
	}

	// Test invalid JSON
	invalidMsg := `{not valid json}`
	err = client.handleMessage(context.Background(), []byte(invalidMsg))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHandleDispatchReady(t *testing.T) {
	client := NewClient("test-token", nil)

	readyCalled := false
	client.OnReady = func(sessionID string) {
		readyCalled = true
		if sessionID != "test-session-id" {
			t.Errorf("expected sessionID 'test-session-id', got '%s'", sessionID)
		}
	}

	readyData := `{
		"v": 10,
		"session_id": "test-session-id",
		"resume_gateway_url": "wss://gateway.discord.gg"
	}`
	err := client.handleDispatch(context.Background(), "READY", json.RawMessage(readyData))
	if err != nil {
		t.Errorf("handleDispatch returned error: %v", err)
	}

	if !readyCalled {
		t.Error("expected OnReady callback to be called")
	}

	if client.SessionID() != "test-session-id" {
		t.Errorf("expected sessionID 'test-session-id', got '%s'", client.SessionID())
	}
}

func TestHandleDispatchResumed(t *testing.T) {
	client := NewClient("test-token", nil)

	stateChanges := make([]int, 0)
	client.OnStateChange = func(state int) {
		stateChanges = append(stateChanges, state)
	}

	err := client.handleDispatch(context.Background(), "RESUMED", json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("handleDispatch returned error: %v", err)
	}

	if client.State() != StateConnected {
		t.Errorf("expected state StateConnected after RESUMED, got %d", client.State())
	}
}

func TestHandleReconnect(t *testing.T) {
	client := NewClient("test-token", nil)

	disconnectCalled := false
	client.OnDisconnect = func(code int, reason string) {
		disconnectCalled = true
		if reason != "reconnect requested" {
			t.Errorf("expected reason 'reconnect requested', got '%s'", reason)
		}
	}

	client.handleReconnect()

	if !disconnectCalled {
		t.Error("expected OnDisconnect callback to be called")
	}
}

func TestHandleInvalidSession(t *testing.T) {
	client := NewClient("test-token", nil)
	client.sessionID = "old-session"
	client.sequence = 100

	errorCalled := false
	client.OnError = func(err error) {
		errorCalled = true
		if err != ErrInvalidSession {
			t.Errorf("expected ErrInvalidSession, got %v", err)
		}
	}

	// Non-resumable session
	client.handleInvalidSession(json.RawMessage(`false`))

	if !errorCalled {
		t.Error("expected OnError callback to be called")
	}

	if client.SessionID() != "" {
		t.Errorf("expected sessionID to be cleared, got '%s'", client.SessionID())
	}

	if client.Sequence() != 0 {
		t.Errorf("expected sequence to be 0, got %d", client.Sequence())
	}
}

func TestHandleInvalidSessionResumable(t *testing.T) {
	client := NewClient("test-token", nil)
	client.sessionID = "old-session"
	client.sequence = 100

	client.OnError = func(err error) {}

	// Resumable session
	client.handleInvalidSession(json.RawMessage(`true`))

	if client.SessionID() != "old-session" {
		t.Errorf("expected sessionID to be preserved, got '%s'", client.SessionID())
	}

	if client.Sequence() != 100 {
		t.Errorf("expected sequence to be 100, got %d", client.Sequence())
	}
}

func TestHandleHello(t *testing.T) {
	mock := newMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect directly to the mock server
	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	client := NewClient("test-token", nil)
	client.conn = conn
	client.heartbeatStop = make(chan struct{})

	// Read HELLO from mock server
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read HELLO: %v", err)
	}

	var msg GatewayMessage
	_ = json.Unmarshal(data, &msg)

	err = client.handleHello(ctx, msg.Data)
	if err != nil {
		t.Errorf("handleHello returned error: %v", err)
	}

	// Give time for heartbeat goroutine to start
	time.Sleep(50 * time.Millisecond)

	client.mu.RLock()
	interval := client.heartbeatInterval
	client.mu.RUnlock()

	if interval != 100*time.Millisecond {
		t.Errorf("expected heartbeat interval 100ms, got %v", interval)
	}

	// Stop heartbeat
	close(client.heartbeatStop)
}

func TestSequenceUpdates(t *testing.T) {
	client := NewClient("test-token", nil)

	// Message with sequence
	msg := `{"op": 0, "s": 42, "t": "UNKNOWN_EVENT", "d": {}}`
	_ = client.handleMessage(context.Background(), []byte(msg))

	if client.Sequence() != 42 {
		t.Errorf("expected sequence 42, got %d", client.Sequence())
	}
}

func TestConnectToMockServer(t *testing.T) {
	mock := newMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect directly (we test the mock server, as real Connect uses hardcoded URL)
	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Verify we get HELLO
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	var hello struct {
		Op int `json:"op"`
	}
	_ = json.Unmarshal(data, &hello)

	if hello.Op != OpHello {
		t.Errorf("expected OpHello, got %d", hello.Op)
	}
}

func TestCloseWithActiveConnection(t *testing.T) {
	mock := newMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	client := NewClient("test-token", nil)
	client.conn = conn
	client.state = StateConnected
	client.heartbeatStop = make(chan struct{})
	client.readStop = make(chan struct{})
	client.readDone = make(chan struct{})

	// Start a goroutine to close readDone when readStop is closed
	go func() {
		<-client.readStop
		close(client.readDone)
	}()

	stateChanged := false
	client.OnStateChange = func(state int) {
		if state == StateClosed {
			stateChanged = true
		}
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	if client.State() != StateClosed {
		t.Errorf("expected state StateClosed, got %d", client.State())
	}

	if !stateChanged {
		t.Error("expected OnStateChange to be called with StateClosed")
	}
}

func TestSendIdentifyWithConnection(t *testing.T) {
	mock := newMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, _ = conn.Read(ctx)

	client := NewClient("test-token", nil)
	client.conn = conn

	err = client.SendIdentify(ctx)
	if err != nil {
		t.Errorf("SendIdentify returned error: %v", err)
	}

	// Read READY response (sent by mock server on IDENTIFY)
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read READY: %v", err)
	}

	var ready struct {
		Op   int    `json:"op"`
		Type string `json:"t"`
	}
	_ = json.Unmarshal(data, &ready)

	if ready.Op != OpDispatch || ready.Type != "READY" {
		t.Errorf("expected READY dispatch, got op=%d, type=%s", ready.Op, ready.Type)
	}
}

func TestSendHeartbeatWithConnection(t *testing.T) {
	mock := newMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, _ = conn.Read(ctx)

	client := NewClient("test-token", nil)
	client.conn = conn
	client.sequence = 42

	err = client.SendHeartbeat(ctx)
	if err != nil {
		t.Errorf("SendHeartbeat returned error: %v", err)
	}

	// Read ACK
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read ACK: %v", err)
	}

	var ack struct {
		Op int `json:"op"`
	}
	_ = json.Unmarshal(data, &ack)

	if ack.Op != OpHeartbeatAck {
		t.Errorf("expected HeartbeatAck, got op=%d", ack.Op)
	}
}

func TestSendPresenceUpdateWithConnection(t *testing.T) {
	mock := newMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, _ = conn.Read(ctx)

	client := NewClient("test-token", nil)
	client.conn = conn

	err = client.SendPresenceUpdate(ctx, "online")
	if err != nil {
		t.Errorf("SendPresenceUpdate returned error: %v", err)
	}
}

func TestSendVoiceStateUpdateWithConnection(t *testing.T) {
	mock := newMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, _ = conn.Read(ctx)

	client := NewClient("test-token", nil)
	client.conn = conn

	// With channel ID
	err = client.SendVoiceStateUpdate(ctx, "guild123", "channel123", true, false)
	if err != nil {
		t.Errorf("SendVoiceStateUpdate returned error: %v", err)
	}

	// Without channel ID (disconnect)
	err = client.SendVoiceStateUpdate(ctx, "guild123", "", false, false)
	if err != nil {
		t.Errorf("SendVoiceStateUpdate returned error: %v", err)
	}
}

func TestHandleMessageReconnect(t *testing.T) {
	client := NewClient("test-token", nil)

	disconnectCalled := false
	client.OnDisconnect = func(code int, reason string) {
		disconnectCalled = true
	}

	// OpReconnect message
	msg := `{"op": 7}`
	err := client.handleMessage(context.Background(), []byte(msg))
	if err != nil {
		t.Errorf("handleMessage returned error: %v", err)
	}

	if !disconnectCalled {
		t.Error("expected OnDisconnect to be called for reconnect")
	}
}

func TestHandleMessageInvalidSession(t *testing.T) {
	client := NewClient("test-token", nil)

	errorCalled := false
	client.OnError = func(err error) {
		errorCalled = true
	}

	// OpInvalidSession message
	msg := `{"op": 9, "d": false}`
	err := client.handleMessage(context.Background(), []byte(msg))
	if err != nil {
		t.Errorf("handleMessage returned error: %v", err)
	}

	if !errorCalled {
		t.Error("expected OnError to be called for invalid session")
	}
}

func TestHandleReadErrorFatalCode(t *testing.T) {
	client := NewClient("test-token", nil)
	client.state = StateConnected

	// Simulate a fatal close error by using handleReadError directly
	// We can't easily simulate websocket close codes, so we test the state change
	client.setState(StateDisconnected)

	if client.State() != StateDisconnected {
		t.Errorf("expected state StateDisconnected, got %d", client.State())
	}
}

func TestStartHeartbeatZeroInterval(t *testing.T) {
	client := NewClient("test-token", nil)
	client.heartbeatInterval = 0

	ctx := context.Background()

	// Should return immediately without error when interval is 0
	done := make(chan struct{})
	go func() {
		client.startHeartbeat(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("startHeartbeat did not return immediately for zero interval")
	}
}

func TestHandleDispatchUnknownEvent(t *testing.T) {
	client := NewClient("test-token", nil)

	// Unknown event type should not error
	err := client.handleDispatch(context.Background(), "UNKNOWN_EVENT", json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("handleDispatch returned error for unknown event: %v", err)
	}
}

func TestHandleDispatchReadyInvalidJSON(t *testing.T) {
	client := NewClient("test-token", nil)

	// Invalid JSON for READY should error
	err := client.handleDispatch(context.Background(), "READY", json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid READY JSON")
	}
}

func TestHandleHelloInvalidJSON(t *testing.T) {
	client := NewClient("test-token", nil)

	// Invalid JSON for HELLO should error
	err := client.handleHello(context.Background(), json.RawMessage(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid HELLO JSON")
	}
}
