package ws

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

type Handler struct {
	hub            *Hub
	allowedOrigins []string
	logger         *slog.Logger
}

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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if !h.isOriginAllowed(origin, r.Host) {
		h.logger.Warn("Origin not allowed", "origin", origin)
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: false,
		OriginPatterns:     h.allowedOrigins,
	})
	if err != nil {
		h.logger.Error("Failed to upgrade connection", "error", err)
		return
	}

	h.logger.Info("WebSocket client connected", "remote_addr", r.RemoteAddr)

	client := NewClient(conn, h.hub, h.logger)
	h.hub.Register(client)

	ctx := r.Context()

	go client.WritePump(ctx)
	client.ReadPump(ctx)
}

func (h *Handler) isOriginAllowed(origin, host string) bool {
	if origin == "" {
		return true
	}

	originHost := strings.TrimPrefix(origin, "http://")
	originHost = strings.TrimPrefix(originHost, "https://")

	if originHost == host {
		return true
	}

	if hostPart, _, found := strings.Cut(originHost, ":"); found {
		originHost = hostPart
	}

	hostWithoutPort := host
	if hostPart, _, found := strings.Cut(host, ":"); found {
		hostWithoutPort = hostPart
	}

	if originHost == hostWithoutPort {
		return true
	}

	for _, allowed := range h.allowedOrigins {
		allowed = strings.TrimPrefix(allowed, "http://")
		allowed = strings.TrimPrefix(allowed, "https://")
		if hostPart, _, found := strings.Cut(allowed, ":"); found {
			allowed = hostPart
		}

		if originHost == allowed {
			return true
		}
	}

	return false
}

func (h *Handler) Upgrade(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTP(w, r)
}

func (h *Handler) Hub() *Hub {
	return h.hub
}

func HandleWebSocket(hub *Hub, allowedOrigins string, logger *slog.Logger) http.HandlerFunc {
	handler := NewHandler(hub, allowedOrigins, logger)
	return handler.ServeHTTP
}
