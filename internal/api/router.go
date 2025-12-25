package api

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/pyyupsk/discord-stayonline/internal/config"
	"github.com/pyyupsk/discord-stayonline/internal/manager"
	"github.com/pyyupsk/discord-stayonline/internal/ws"
)

// Router sets up HTTP routes for the API.
type Router struct {
	mux     *http.ServeMux
	store   *config.Store
	manager *manager.SessionManager
	hub     *ws.Hub
	logger  *slog.Logger
}

// NewRouter creates a new API router.
func NewRouter(store *config.Store, mgr *manager.SessionManager, hub *ws.Hub, logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{
		mux:     http.NewServeMux(),
		store:   store,
		manager: mgr,
		hub:     hub,
		logger:  logger,
	}
}

// Setup configures all HTTP routes.
func (r *Router) Setup() http.Handler {
	// Health endpoint
	healthHandler := NewHealthHandler()
	r.mux.HandleFunc("GET /health", healthHandler.Health)

	// TOS acknowledgment
	tosHandler := NewTOSHandler(r.store, r.logger)
	r.mux.HandleFunc("POST /api/acknowledge-tos", tosHandler.AcknowledgeTOS)

	// Config handlers
	configHandler := NewConfigHandler(r.store, r.logger)
	r.mux.HandleFunc("GET /api/config", configHandler.GetConfig)
	r.mux.HandleFunc("POST /api/config", configHandler.ReplaceConfig)
	r.mux.HandleFunc("PUT /api/config", configHandler.UpdateConfig)

	// Server action handlers
	if r.manager != nil {
		serversHandler := NewServersHandler(r.manager, r.logger)
		r.mux.HandleFunc("POST /api/servers/", serversHandler.ExecuteAction)
	}

	// WebSocket handler
	if r.hub != nil {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		wsHandler := ws.NewHandler(r.hub, allowedOrigins, r.logger)
		r.mux.HandleFunc("/ws", wsHandler.ServeHTTP)
	}

	// TODO: Static file handler (/)

	return r.mux
}

// Handler returns the HTTP handler.
func (r *Router) Handler() http.Handler {
	return r.mux
}
