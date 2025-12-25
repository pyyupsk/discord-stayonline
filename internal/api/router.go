// Package api provides HTTP handlers for the Discord presence service REST API.
package api

import (
	"io/fs"
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
	store   config.ConfigStore
	manager *manager.SessionManager
	hub     *ws.Hub
	webFS   fs.FS
	logger  *slog.Logger
	auth    *AuthMiddleware
}

// NewRouter creates a new API router.
func NewRouter(store config.ConfigStore, mgr *manager.SessionManager, hub *ws.Hub, webFS fs.FS, logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	auth := NewAuthMiddleware(logger)
	if auth.IsEnabled() {
		logger.Info("API key authentication enabled")
	} else {
		logger.Warn("API key authentication disabled - set API_KEY environment variable to enable")
	}
	return &Router{
		mux:     http.NewServeMux(),
		store:   store,
		manager: mgr,
		hub:     hub,
		webFS:   webFS,
		logger:  logger,
		auth:    auth,
	}
}

// Setup configures all HTTP routes.
func (r *Router) Setup() http.Handler {
	// Health endpoint (public)
	healthHandler := NewHealthHandler(r.manager, r.hub)
	r.mux.HandleFunc("GET /health", healthHandler.Health)
	r.mux.HandleFunc("HEAD /health", healthHandler.Health)

	// Auth endpoints (public)
	authHandler := NewAuthHandler(r.auth, r.logger)
	r.mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	r.mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	r.mux.HandleFunc("GET /api/auth/check", authHandler.Check)

	// TOS acknowledgment (protected)
	tosHandler := NewTOSHandler(r.store, r.logger)
	r.mux.HandleFunc("POST /api/acknowledge-tos", r.auth.Protect(tosHandler.AcknowledgeTOS))

	// Config handlers (protected)
	configHandler := NewConfigHandler(r.store, r.logger)
	r.mux.HandleFunc("GET /api/config", r.auth.Protect(configHandler.GetConfig))
	r.mux.HandleFunc("POST /api/config", r.auth.Protect(configHandler.ReplaceConfig))
	r.mux.HandleFunc("PUT /api/config", r.auth.Protect(configHandler.UpdateConfig))

	// Server action handlers (protected)
	if r.manager != nil {
		serversHandler := NewServersHandler(r.manager, r.logger)
		r.mux.HandleFunc("GET /api/statuses", r.auth.Protect(serversHandler.GetStatuses))
		r.mux.HandleFunc("POST /api/servers/", r.auth.Protect(serversHandler.ExecuteAction))
	}

	// Discord info lookup handlers (protected)
	discordHandler := NewDiscordHandler(r.logger)
	r.mux.HandleFunc("GET /api/discord/server-info", r.auth.Protect(discordHandler.GetServerInfo))
	r.mux.HandleFunc("POST /api/discord/bulk-info", r.auth.Protect(discordHandler.GetBulkServerInfo))
	r.mux.HandleFunc("GET /api/discord/guilds", r.auth.Protect(discordHandler.GetUserGuilds))
	r.mux.HandleFunc("GET /api/discord/guilds/", r.auth.Protect(discordHandler.GetGuildChannels))

	// Logs handler (protected)
	if r.hub != nil {
		logsHandler := NewLogsHandler(r.hub, r.logger)
		r.mux.HandleFunc("GET /api/logs", r.auth.Protect(logsHandler.GetLogs))
	}

	// WebSocket handler (protected via middleware wrapper)
	if r.hub != nil {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		wsHandler := ws.NewHandler(r.hub, allowedOrigins, r.logger)
		r.mux.Handle("/ws", r.auth.ProtectHandler(http.HandlerFunc(wsHandler.ServeHTTP)))
	}

	// Static file handler (public - login page needs to load)
	if r.webFS != nil {
		r.mux.Handle("/", http.FileServer(http.FS(r.webFS)))
	}

	return r.mux
}

// Handler returns the HTTP handler.
func (r *Router) Handler() http.Handler {
	return r.mux
}
