package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Client represents a connected WebSocket client.
type Client struct {
	conn       *websocket.Conn
	hub        *Hub
	send       chan []byte
	logger     *slog.Logger
	subscribed map[string]bool
	mu         sync.RWMutex
}

// NewClient creates a new WebSocket client.
func NewClient(conn *websocket.Conn, hub *Hub, logger *slog.Logger) *Client {
	return &Client{
		conn:       conn,
		hub:        hub,
		send:       make(chan []byte, 256),
		logger:     logger,
		subscribed: make(map[string]bool),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close(websocket.StatusGoingAway, "closing")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgType, data, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				c.logger.Debug("WebSocket closed", "status", websocket.CloseStatus(err))
			} else {
				c.logger.Error("Read error", "error", err)
			}
			return
		}

		if msgType != websocket.MessageText {
			continue
		}

		c.handleMessage(ctx, data)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusGoingAway, "closing")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusGoingAway, "hub closed")
				return
			}

			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := c.conn.Write(writeCtx, websocket.MessageText, message)
			cancel()

			if err != nil {
				c.logger.Error("Write error", "error", err)
				return
			}
		case <-ticker.C:
			// Send ping
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := c.conn.Ping(pingCtx)
			cancel()

			if err != nil {
				c.logger.Error("Ping error", "error", err)
				return
			}
		}
	}
}

// handleMessage processes an incoming client message.
func (c *Client) handleMessage(_ context.Context, data []byte) {
	var msg struct {
		Type     string `json:"type"`
		Channel  string `json:"channel,omitempty"`
		ServerID string `json:"server_id,omitempty"`
		Action   string `json:"action,omitempty"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.logger.Error("Failed to parse message", "error", err)
		return
	}

	switch msg.Type {
	case "subscribe":
		c.subscribe(msg.Channel)
	case "unsubscribe":
		c.unsubscribe(msg.Channel)
	case "action":
		// Actions are handled via REST API, not WebSocket
		c.logger.Debug("Action via WebSocket not supported, use REST API")
	}
}

// subscribe adds a channel subscription.
func (c *Client) subscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscribed[channel] = true
	c.logger.Debug("Subscribed to channel", "channel", channel)
}

// unsubscribe removes a channel subscription.
func (c *Client) unsubscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscribed, channel)
	c.logger.Debug("Unsubscribed from channel", "channel", channel)
}

// IsSubscribed checks if the client is subscribed to a channel.
func (c *Client) IsSubscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.subscribed[channel]
}

// Send queues a message to be sent to the client.
func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	default:
		// Buffer full, drop message
		c.logger.Warn("Client send buffer full, dropping message")
	}
}
