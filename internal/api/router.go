package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/pyyupsk/discord-stayonline/internal/api/handlers"
	"github.com/pyyupsk/discord-stayonline/internal/api/middleware"
	"github.com/pyyupsk/discord-stayonline/internal/config"
	"github.com/pyyupsk/discord-stayonline/internal/manager"
	"github.com/pyyupsk/discord-stayonline/internal/ui"
	"github.com/pyyupsk/discord-stayonline/internal/ws"
)

type Router struct {
	mux     *http.ServeMux
	store   config.ConfigStore
	manager *manager.SessionManager
	hub     *ws.Hub
	webFS   fs.FS
	logger  *slog.Logger
	auth    *middleware.Auth
}

func NewRouter(store config.ConfigStore, mgr *manager.SessionManager, hub *ws.Hub, webFS fs.FS, logger *slog.Logger) (*Router, error) {
	if logger == nil {
		logger = slog.Default()
	}
	auth, err := middleware.NewAuth(logger)
	if err != nil {
		return nil, err
	}
	logger.Info("API key authentication enabled")
	return &Router{
		mux:     http.NewServeMux(),
		store:   store,
		manager: mgr,
		hub:     hub,
		webFS:   webFS,
		logger:  logger,
		auth:    auth,
	}, nil
}

func (r *Router) Setup() http.Handler {
	healthHandler := handlers.NewHealthHandler(r.manager, r.hub)
	r.mux.HandleFunc("GET /health", healthHandler.Health)
	r.mux.HandleFunc("HEAD /health", healthHandler.Health)

	authHandler := handlers.NewAuthHandler(r.auth, r.logger)
	r.mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	r.mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	r.mux.HandleFunc("GET /api/auth/check", authHandler.Check)

	tosHandler := handlers.NewTOSHandler(r.store, r.logger)
	r.mux.HandleFunc("POST /api/acknowledge-tos", r.auth.Protect(tosHandler.AcknowledgeTOS))

	configHandler := handlers.NewConfigHandler(r.store, r.logger)
	r.mux.HandleFunc("GET /api/config", r.auth.Protect(configHandler.GetConfig))
	r.mux.HandleFunc("POST /api/config", r.auth.Protect(configHandler.ReplaceConfig))
	r.mux.HandleFunc("PUT /api/config", r.auth.Protect(configHandler.UpdateConfig))

	if r.manager != nil {
		serversHandler := handlers.NewServersHandler(r.manager, r.logger)
		r.mux.HandleFunc("GET /api/statuses", r.auth.Protect(serversHandler.GetStatuses))
		r.mux.HandleFunc("POST /api/servers/", r.auth.Protect(serversHandler.ExecuteAction))
	}

	discordHandler := handlers.NewDiscordHandler(r.logger)
	r.mux.HandleFunc("GET /api/discord/user", r.auth.Protect(discordHandler.GetCurrentUser))
	r.mux.HandleFunc("GET /api/discord/server-info", r.auth.Protect(discordHandler.GetServerInfo))
	r.mux.HandleFunc("POST /api/discord/bulk-info", r.auth.Protect(discordHandler.GetBulkServerInfo))
	r.mux.HandleFunc("GET /api/discord/guilds", r.auth.Protect(discordHandler.GetUserGuilds))
	r.mux.HandleFunc("GET /api/discord/guilds/", r.auth.Protect(discordHandler.GetGuildChannels))

	if r.hub != nil {
		logsHandler := handlers.NewLogsHandler(r.hub, r.logger)
		r.mux.HandleFunc("GET /api/logs", r.auth.Protect(logsHandler.GetLogs))
	}

	if r.hub != nil {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		wsHandler := ws.NewHandler(r.hub, allowedOrigins, r.logger)
		r.mux.Handle("/ws", r.auth.ProtectHandler(http.HandlerFunc(wsHandler.ServeHTTP)))
	}

	if r.webFS != nil {
		r.mux.Handle("/", ui.SPAHandler(r.webFS))
	}

	return r.mux
}

func (r *Router) Handler() http.Handler {
	return r.mux
}
