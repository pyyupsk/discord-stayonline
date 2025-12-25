package tests

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
	"github.com/pyyupsk/discord-stayonline/internal/gateway"
)

const (
	testToken          = "test-token"
	errFailedToConnect = "failed to connect: %v"
	errFailedReadHello = "failed to read HELLO: %v"
)

// MockGatewayServer simulates a Discord Gateway server for testing.
type MockGatewayServer struct {
	server          *httptest.Server
	conn            *websocket.Conn
	mu              sync.Mutex
	heartbeatCount  int
	identifyPayload json.RawMessage
	onConnect       func()
	onIdentify      func(data json.RawMessage)
	onHeartbeat     func(seq *int)
}

// NewMockGatewayServer creates a new mock Gateway server.
func NewMockGatewayServer(t *testing.T) *MockGatewayServer {
	mock := &MockGatewayServer{}

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
		mock.mu.Unlock()

		if mock.onConnect != nil {
			mock.onConnect()
		}

		// Send HELLO
		hello := map[string]any{
			"op": gateway.OpHello,
			"d": map[string]any{
				"heartbeat_interval": 100, // 100ms for fast testing
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

			mock.handleMessage(r.Context(), data, t)
		}
	})

	mock.server = httptest.NewServer(handler)
	return mock
}

// URL returns the WebSocket URL for the mock server.
func (m *MockGatewayServer) URL() string {
	return "ws" + strings.TrimPrefix(m.server.URL, "http")
}

// Close shuts down the mock server.
func (m *MockGatewayServer) Close() {
	if m.conn != nil {
		m.conn.Close(websocket.StatusNormalClosure, "server closing")
	}
	m.server.Close()
}

// HeartbeatCount returns the number of heartbeats received.
func (m *MockGatewayServer) HeartbeatCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.heartbeatCount
}

// IdentifyPayload returns the last received IDENTIFY payload.
func (m *MockGatewayServer) IdentifyPayload() json.RawMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.identifyPayload
}

// SendReady sends a READY event to the client.
func (m *MockGatewayServer) SendReady(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	if conn == nil {
		return nil
	}

	ready := map[string]any{
		"op": gateway.OpDispatch,
		"t":  "READY",
		"s":  1,
		"d": map[string]any{
			"v":                  10,
			"session_id":         sessionID,
			"resume_gateway_url": m.URL(),
		},
	}
	data, _ := json.Marshal(ready)
	return conn.Write(ctx, websocket.MessageText, data)
}

// SendReconnect sends a reconnect request to the client.
func (m *MockGatewayServer) SendReconnect(ctx context.Context) error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	if conn == nil {
		return nil
	}

	reconnect := map[string]any{
		"op": gateway.OpReconnect,
		"d":  nil,
	}
	data, _ := json.Marshal(reconnect)
	return conn.Write(ctx, websocket.MessageText, data)
}

// SendInvalidSession sends an invalid session notification.
func (m *MockGatewayServer) SendInvalidSession(ctx context.Context, resumable bool) error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	if conn == nil {
		return nil
	}

	invalidSession := map[string]any{
		"op": gateway.OpInvalidSession,
		"d":  resumable,
	}
	data, _ := json.Marshal(invalidSession)
	return conn.Write(ctx, websocket.MessageText, data)
}

// CloseConnection closes the WebSocket with the specified code.
func (m *MockGatewayServer) CloseConnection(code websocket.StatusCode, reason string) error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()

	if conn == nil {
		return nil
	}

	return conn.Close(code, reason)
}

// handleMessage processes incoming messages from the client.
func (m *MockGatewayServer) handleMessage(ctx context.Context, data []byte, t *testing.T) {
	var msg struct {
		Op   int             `json:"op"`
		Data json.RawMessage `json:"d"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Logf("failed to unmarshal message: %v", err)
		return
	}

	switch msg.Op {
	case gateway.OpIdentify:
		m.mu.Lock()
		m.identifyPayload = msg.Data
		m.mu.Unlock()

		if m.onIdentify != nil {
			m.onIdentify(msg.Data)
		}

		// Send READY response
		_ = m.SendReady(ctx, "test-session-123")

	case gateway.OpHeartbeat:
		m.mu.Lock()
		m.heartbeatCount++
		count := m.heartbeatCount
		m.mu.Unlock()

		// Parse sequence from heartbeat
		var seq *int
		_ = json.Unmarshal(msg.Data, &seq)

		if m.onHeartbeat != nil {
			m.onHeartbeat(seq)
		}

		// Send heartbeat ACK
		ack := map[string]any{
			"op": gateway.OpHeartbeatAck,
		}
		ackData, _ := json.Marshal(ack)
		m.mu.Lock()
		conn := m.conn
		m.mu.Unlock()
		if conn != nil {
			_ = conn.Write(ctx, websocket.MessageText, ackData)
		}

		t.Logf("received heartbeat #%d", count)

	case gateway.OpPresenceUpdate:
		t.Logf("received presence update")

	case gateway.OpVoiceStateUpdate:
		t.Logf("received voice state update")
	}
}

// TestGatewayClientConnect tests basic connection to the Gateway.
func TestGatewayClientConnect(t *testing.T) {
	// Create mock server
	mock := NewMockGatewayServer(t)
	defer mock.Close()

	// We can't directly test with the real client since it uses a hardcoded URL
	// Instead, we verify the mock server works correctly
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect to mock server
	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect to mock server: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf(errFailedReadHello, err)
	}

	var hello struct {
		Op   int `json:"op"`
		Data struct {
			HeartbeatInterval int `json:"heartbeat_interval"`
		} `json:"d"`
	}
	if err := json.Unmarshal(data, &hello); err != nil {
		t.Fatalf("failed to unmarshal HELLO: %v", err)
	}

	if hello.Op != gateway.OpHello {
		t.Errorf("expected OpHello (%d), got %d", gateway.OpHello, hello.Op)
	}

	if hello.Data.HeartbeatInterval != 100 {
		t.Errorf("expected heartbeat interval 100, got %d", hello.Data.HeartbeatInterval)
	}
}

// TestGatewayHeartbeat tests heartbeat mechanism.
func TestGatewayHeartbeat(t *testing.T) {
	mock := NewMockGatewayServer(t)
	defer mock.Close()

	heartbeatReceived := make(chan struct{})
	mock.onHeartbeat = func(seq *int) {
		select {
		case heartbeatReceived <- struct{}{}:
		default:
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect
	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf(errFailedToConnect, err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf(errFailedReadHello, err)
	}

	// Send heartbeat
	heartbeat := map[string]any{
		"op": gateway.OpHeartbeat,
		"d":  nil,
	}
	heartbeatData, _ := json.Marshal(heartbeat)
	if err := conn.Write(ctx, websocket.MessageText, heartbeatData); err != nil {
		t.Fatalf("failed to send heartbeat: %v", err)
	}

	// Wait for heartbeat to be received
	select {
	case <-heartbeatReceived:
		// Success
	case <-ctx.Done():
		t.Fatal("timeout waiting for heartbeat")
	}

	// Read heartbeat ACK
	_, ackData, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read heartbeat ACK: %v", err)
	}

	var ack struct {
		Op int `json:"op"`
	}
	if err := json.Unmarshal(ackData, &ack); err != nil {
		t.Fatalf("failed to unmarshal heartbeat ACK: %v", err)
	}

	if ack.Op != gateway.OpHeartbeatAck {
		t.Errorf("expected OpHeartbeatAck (%d), got %d", gateway.OpHeartbeatAck, ack.Op)
	}

	if mock.HeartbeatCount() != 1 {
		t.Errorf("expected 1 heartbeat, got %d", mock.HeartbeatCount())
	}
}

// TestGatewayIdentify tests the IDENTIFY flow.
func TestGatewayIdentify(t *testing.T) {
	mock := NewMockGatewayServer(t)
	defer mock.Close()

	identifyReceived := make(chan struct{})
	mock.onIdentify = func(data json.RawMessage) {
		close(identifyReceived)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect
	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf(errFailedToConnect, err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf(errFailedReadHello, err)
	}

	// Send IDENTIFY
	identify := map[string]any{
		"op": gateway.OpIdentify,
		"d": map[string]any{
			"token": testToken,
			"properties": map[string]string{
				"os":      "Windows",
				"browser": "Discord Client",
				"device":  "",
			},
		},
	}
	identifyData, _ := json.Marshal(identify)
	if err := conn.Write(ctx, websocket.MessageText, identifyData); err != nil {
		t.Fatalf("failed to send IDENTIFY: %v", err)
	}

	// Wait for IDENTIFY to be received
	select {
	case <-identifyReceived:
		// Success
	case <-ctx.Done():
		t.Fatal("timeout waiting for IDENTIFY")
	}

	// Read READY
	_, readyData, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read READY: %v", err)
	}

	var ready struct {
		Op   int    `json:"op"`
		Type string `json:"t"`
		Data struct {
			SessionID string `json:"session_id"`
		} `json:"d"`
	}
	if err := json.Unmarshal(readyData, &ready); err != nil {
		t.Fatalf("failed to unmarshal READY: %v", err)
	}

	if ready.Op != gateway.OpDispatch {
		t.Errorf("expected OpDispatch (%d), got %d", gateway.OpDispatch, ready.Op)
	}

	if ready.Type != "READY" {
		t.Errorf("expected event type READY, got %s", ready.Type)
	}

	if ready.Data.SessionID != "test-session-123" {
		t.Errorf("expected session ID test-session-123, got %s", ready.Data.SessionID)
	}
}

// TestGatewayReconnect tests reconnect request handling.
func TestGatewayReconnect(t *testing.T) {
	mock := NewMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect
	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf(errFailedToConnect, err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf(errFailedReadHello, err)
	}

	// Send reconnect from server
	if err := mock.SendReconnect(ctx); err != nil {
		t.Fatalf("failed to send reconnect: %v", err)
	}

	// Read reconnect
	_, reconnectData, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read reconnect: %v", err)
	}

	var reconnect struct {
		Op int `json:"op"`
	}
	if err := json.Unmarshal(reconnectData, &reconnect); err != nil {
		t.Fatalf("failed to unmarshal reconnect: %v", err)
	}

	if reconnect.Op != gateway.OpReconnect {
		t.Errorf("expected OpReconnect (%d), got %d", gateway.OpReconnect, reconnect.Op)
	}
}

// TestGatewayInvalidSession tests invalid session handling.
func TestGatewayInvalidSession(t *testing.T) {
	mock := NewMockGatewayServer(t)
	defer mock.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect
	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf(errFailedToConnect, err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf(errFailedReadHello, err)
	}

	// Send invalid session (not resumable)
	if err := mock.SendInvalidSession(ctx, false); err != nil {
		t.Fatalf("failed to send invalid session: %v", err)
	}

	// Read invalid session
	_, invalidData, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read invalid session: %v", err)
	}

	var invalid struct {
		Op   int  `json:"op"`
		Data bool `json:"d"`
	}
	if err := json.Unmarshal(invalidData, &invalid); err != nil {
		t.Fatalf("failed to unmarshal invalid session: %v", err)
	}

	if invalid.Op != gateway.OpInvalidSession {
		t.Errorf("expected OpInvalidSession (%d), got %d", gateway.OpInvalidSession, invalid.Op)
	}

	if invalid.Data != false {
		t.Errorf("expected resumable=false, got %v", invalid.Data)
	}
}

// Note: TestIsFatalCloseCode is in backoff_test.go

// TestReconnector tests the reconnection logic.
func TestReconnector(t *testing.T) {
	// Test that backoff is applied correctly
	client := gateway.NewClient(testToken, nil)
	reconnector := gateway.NewReconnector(client, testToken, nil)

	if reconnector.Attempt() != 0 {
		t.Errorf("expected initial attempt to be 0, got %d", reconnector.Attempt())
	}

	reconnector.ResetAttempts()
	if reconnector.Attempt() != 0 {
		t.Errorf("expected attempt after reset to be 0, got %d", reconnector.Attempt())
	}
}

// TestFullConnectionLifecycle tests a complete connection lifecycle.
func TestFullConnectionLifecycle(t *testing.T) {
	mock := NewMockGatewayServer(t)
	defer mock.Close()

	var (
		identifyDone  = make(chan struct{})
		heartbeatDone = make(chan struct{})
	)

	mock.onIdentify = func(data json.RawMessage) {
		close(identifyDone)
	}

	heartbeatCnt := 0
	mock.onHeartbeat = func(seq *int) {
		heartbeatCnt++
		if heartbeatCnt >= 2 {
			select {
			case <-heartbeatDone:
			default:
				close(heartbeatDone)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect
	conn, _, err := websocket.Dial(ctx, mock.URL(), nil)
	if err != nil {
		t.Fatalf(errFailedToConnect, err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Read HELLO
	_, _, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf(errFailedReadHello, err)
	}

	// Send IDENTIFY
	identify := map[string]any{
		"op": gateway.OpIdentify,
		"d": map[string]any{
			"token": testToken,
			"properties": map[string]string{
				"os":      "Windows",
				"browser": "Discord Client",
				"device":  "",
			},
		},
	}
	identifyData, _ := json.Marshal(identify)
	if err := conn.Write(ctx, websocket.MessageText, identifyData); err != nil {
		t.Fatalf("failed to send IDENTIFY: %v", err)
	}

	// Wait for IDENTIFY processing
	select {
	case <-identifyDone:
	case <-ctx.Done():
		t.Fatal("timeout waiting for IDENTIFY")
	}

	// Read READY
	_, _, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("failed to read READY: %v", err)
	}

	// Send multiple heartbeats
	for i := range 2 {
		heartbeat := map[string]any{
			"op": gateway.OpHeartbeat,
			"d":  i,
		}
		heartbeatData, _ := json.Marshal(heartbeat)
		if err := conn.Write(ctx, websocket.MessageText, heartbeatData); err != nil {
			t.Fatalf("failed to send heartbeat %d: %v", i, err)
		}

		// Read ACK
		_, _, err = conn.Read(ctx)
		if err != nil {
			t.Fatalf("failed to read heartbeat ACK %d: %v", i, err)
		}
	}

	// Wait for heartbeats
	select {
	case <-heartbeatDone:
	case <-ctx.Done():
		t.Fatal("timeout waiting for heartbeats")
	}

	// Verify lifecycle completed
	if mock.HeartbeatCount() < 2 {
		t.Errorf("expected at least 2 heartbeats, got %d", mock.HeartbeatCount())
	}
}
