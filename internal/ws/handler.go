package ws

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub            *Hub
	allowedOrigins []string
	logger         *slog.Logger
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *Hub, allowedOrigins string, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	origins := []string{"localhost", "127.0.0.1"}
	if allowedOrigins != "" {
		for _, origin := range strings.Split(allowedOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				origins = append(origins, origin)
			}
		}
	}

	return &Handler{
		hub:            hub,
		allowedOrigins: origins,
		logger:         logger.With("handler", "websocket"),
	}
}

// ServeHTTP handles WebSocket upgrade requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate origin
	origin := r.Header.Get("Origin")
	if !h.isOriginAllowed(origin, r.Host) {
		h.logger.Warn("Origin not allowed", "origin", origin)
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	// Upgrade connection
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: false,
		OriginPatterns:     h.allowedOrigins,
	})
	if err != nil {
		h.logger.Error("Failed to upgrade connection", "error", err)
		return
	}

	h.logger.Info("WebSocket client connected", "remote_addr", r.RemoteAddr)

	// Create client
	client := NewClient(conn, h.hub, h.logger)
	h.hub.Register(client)

	// Create context for this connection
	ctx := r.Context()

	// Start read and write pumps
	go client.WritePump(ctx)
	client.ReadPump(ctx)
}

// isOriginAllowed checks if the origin is in the allowed list or matches the host (same-origin).
func (h *Handler) isOriginAllowed(origin, host string) bool {
	if origin == "" {
		return true // Allow requests without origin (non-browser clients)
	}

	// Remove protocol prefix from origin
	originHost := strings.TrimPrefix(origin, "http://")
	originHost = strings.TrimPrefix(originHost, "https://")

	// Allow same-origin requests (origin matches host)
	if originHost == host {
		return true
	}

	// Remove port if present for comparison
	if idx := strings.Index(originHost, ":"); idx != -1 {
		originHost = originHost[:idx]
	}

	// Also check host without port
	hostWithoutPort := host
	if idx := strings.Index(host, ":"); idx != -1 {
		hostWithoutPort = host[:idx]
	}

	if originHost == hostWithoutPort {
		return true
	}

	// Check against allowed origins list
	for _, allowed := range h.allowedOrigins {
		allowed = strings.TrimPrefix(allowed, "http://")
		allowed = strings.TrimPrefix(allowed, "https://")
		if idx := strings.Index(allowed, ":"); idx != -1 {
			allowed = allowed[:idx]
		}

		if originHost == allowed {
			return true
		}
	}

	return false
}

// Upgrade is an alias for ServeHTTP to match common naming.
func (h *Handler) Upgrade(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTP(w, r)
}

// Hub returns the underlying hub.
func (h *Handler) Hub() *Hub {
	return h.hub
}

// HandleWebSocket is a convenience function that upgrades and handles a WebSocket connection.
func HandleWebSocket(hub *Hub, allowedOrigins string, logger *slog.Logger) http.HandlerFunc {
	handler := NewHandler(hub, allowedOrigins, logger)
	return handler.ServeHTTP
}
