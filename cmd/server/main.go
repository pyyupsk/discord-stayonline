// Package main provides the entry point for the Discord Stay Online service.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	discordstayonline "github.com/pyyupsk/discord-stayonline"
	"github.com/pyyupsk/discord-stayonline/internal/api"
	"github.com/pyyupsk/discord-stayonline/internal/config"
	"github.com/pyyupsk/discord-stayonline/internal/manager"
	"github.com/pyyupsk/discord-stayonline/internal/webhook"
	"github.com/pyyupsk/discord-stayonline/internal/ws"
)

func main() {
	_ = godotenv.Load()

	logger := initLogger()
	token := getEnvOrDefault("DISCORD_TOKEN", "")
	port := getEnvOrDefault("PORT", "8080")
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")

	if token == "" {
		slog.Warn("DISCORD_TOKEN not set - connections will fail until token is configured")
	}

	webhookNotifier := webhook.NewNotifier(webhookURL, logger)
	if webhookNotifier != nil {
		slog.Info("Discord webhook notifications enabled")
	}

	store, dbStore := initStore()
	cfg, err := store.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("Configuration loaded", "servers", len(cfg.Servers), "tos_acknowledged", cfg.TOSAcknowledged)

	hub := initHub(logger, dbStore)
	sessionMgr := initSessionManager(token, store, dbStore, hub, webhookNotifier, logger)

	webFS, err := discordstayonline.GetWebFS()
	if err != nil {
		slog.Error("Failed to get web filesystem", "error", err)
		os.Exit(1)
	}

	router := api.NewRouter(store, sessionMgr, hub, webFS, logger)
	srv := createServer(port, router.Setup())

	go startSessionManager(sessionMgr)
	go startHTTPServer(srv, port)

	waitForShutdown()
	shutdown(srv, sessionMgr, hub, dbStore)
}

func initLogger() *slog.Logger {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	return logger
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initStore() (config.ConfigStore, *config.DBStore) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		slog.Info("Using PostgreSQL for configuration storage")
		dbStore, err := config.NewDBStore(databaseURL)
		if err != nil {
			slog.Error("Failed to connect to database", "error", err)
			os.Exit(1)
		}
		return dbStore, dbStore
	}

	slog.Info("Using file for configuration storage")
	configPath := getEnvOrDefault("CONFIG_PATH", "config.json")
	return config.NewStore(configPath), nil
}

func initHub(logger *slog.Logger, dbStore *config.DBStore) *ws.Hub {
	var logStore ws.LogStore
	if dbStore != nil {
		logStore = &dbLogStore{db: dbStore}
	}
	hub := ws.NewHub(logger, logStore)
	go hub.Run()
	return hub
}

func initSessionManager(token string, store config.ConfigStore, dbStore *config.DBStore, hub *ws.Hub, webhookNotifier *webhook.Notifier, logger *slog.Logger) *manager.SessionManager {
	var sessionStore manager.SessionStore
	if dbStore != nil {
		sessionStore = &dbSessionStore{db: dbStore}
	}
	sessionMgr := manager.NewSessionManager(token, store, sessionStore, webhookNotifier, logger)
	sessionMgr.OnStatusChange = func(serverID string, status manager.ConnectionStatus, message string) {
		hub.BroadcastStatus(serverID, string(status), message)
	}
	return sessionMgr
}

func createServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func startSessionManager(sessionMgr *manager.SessionManager) {
	if err := sessionMgr.Start(); err != nil {
		slog.Error("Failed to start session manager", "error", err)
	}
}

func startHTTPServer(srv *http.Server, port string) {
	slog.Info("Starting server", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

func shutdown(srv *http.Server, sessionMgr *manager.SessionManager, hub *ws.Hub, dbStore *config.DBStore) {
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessionMgr.Stop()
	hub.Close()

	if dbStore != nil {
		dbStore.Close()
	}

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server stopped")
}

// dbLogStore adapts config.DBStore to ws.LogStore interface.
type dbLogStore struct {
	db *config.DBStore
}

func (s *dbLogStore) AddLog(level, message string) error {
	return s.db.AddLog(level, message)
}

func (s *dbLogStore) GetLogs(level string) ([]ws.LogEntry, error) {
	logs, err := s.db.GetLogs(level)
	if err != nil {
		return nil, err
	}

	// Convert config.LogEntry to ws.LogEntry
	result := make([]ws.LogEntry, len(logs))
	for i, log := range logs {
		result[i] = ws.LogEntry{
			Level:     log.Level,
			Message:   log.Message,
			Timestamp: log.Timestamp,
		}
	}
	return result, nil
}

// dbSessionStore adapts config.DBStore to manager.SessionStore interface.
type dbSessionStore struct {
	db *config.DBStore
}

func (s *dbSessionStore) SaveSession(state config.SessionState) error {
	return s.db.SaveSession(state)
}

func (s *dbSessionStore) LoadSession(serverID string) (*config.SessionState, error) {
	return s.db.LoadSession(serverID)
}

func (s *dbSessionStore) DeleteSession(serverID string) error {
	return s.db.DeleteSession(serverID)
}

func (s *dbSessionStore) UpdateSessionSequence(serverID string, sequence int) error {
	return s.db.UpdateSessionSequence(serverID, sequence)
}
