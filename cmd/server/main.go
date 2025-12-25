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

	discordstayonline "github.com/pyyupsk/discord-stayonline"
	"github.com/pyyupsk/discord-stayonline/internal/api"
	"github.com/pyyupsk/discord-stayonline/internal/config"
	"github.com/pyyupsk/discord-stayonline/internal/manager"
	"github.com/pyyupsk/discord-stayonline/internal/ws"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load DISCORD_TOKEN from environment
	// Token is loaded via environment variable (see .env.example)
	// For production, set DISCORD_TOKEN in your environment or .env file
	// NOTE: The token is NEVER sent to the client or logged
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		slog.Warn("DISCORD_TOKEN not set - connections will fail until token is configured")
	}
	_ = token // Will be used by SessionManager

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize ConfigStore
	store := config.NewStore("config.json")

	// Load existing config or create default
	cfg, err := store.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("Configuration loaded", "servers", len(cfg.Servers), "tos_acknowledged", cfg.TOSAcknowledged)

	// Initialize WebSocket Hub
	hub := ws.NewHub(logger)
	go hub.Run()

	// Initialize SessionManager
	sessionMgr := manager.NewSessionManager(token, store, logger)

	// Wire SessionManager status changes to WebSocket Hub
	sessionMgr.OnStatusChange = func(serverID string, status manager.ConnectionStatus, message string) {
		hub.BroadcastStatus(serverID, string(status), message)
	}

	// Get embedded web filesystem
	webFS, err := discordstayonline.GetWebFS()
	if err != nil {
		slog.Error("Failed to get web filesystem", "error", err)
		os.Exit(1)
	}

	// Set up HTTP router
	router := api.NewRouter(store, sessionMgr, hub, webFS, logger)
	handler := router.Setup()

	// Start SessionManager auto-connect for servers with connect_on_start=true
	go func() {
		if err := sessionMgr.Start(); err != nil {
			slog.Error("Failed to start session manager", "error", err)
		}
	}()

	// Create server with timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Starting server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Gracefully close all Gateway connections
	sessionMgr.Stop()

	// Close WebSocket hub
	hub.Close()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server stopped")
}
