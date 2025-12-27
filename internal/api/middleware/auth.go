// Package middleware provides HTTP middleware components.
package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"os"

	"github.com/pyyupsk/discord-stayonline/internal/api/responses"
)

const (
	// CookieName is the name of the authentication cookie.
	CookieName = "api_key"
	// CookieMaxAge is the cookie lifetime in seconds (7 days).
	CookieMaxAge = 7 * 24 * 60 * 60
)

// Auth provides API key authentication.
type Auth struct {
	apiKey string
	logger *slog.Logger
}

// NewAuth creates a new auth middleware.
func NewAuth(logger *slog.Logger) *Auth {
	if logger == nil {
		logger = slog.Default()
	}
	return &Auth{
		apiKey: os.Getenv("API_KEY"),
		logger: logger.With("middleware", "auth"),
	}
}

// IsEnabled returns true if API key authentication is configured.
func (m *Auth) IsEnabled() bool {
	return m.apiKey != ""
}

// ValidateKey checks if the provided key matches the configured API key.
func (m *Auth) ValidateKey(key string) bool {
	return subtle.ConstantTimeCompare([]byte(key), []byte(m.apiKey)) == 1
}

// Protect wraps a handler to require valid API key.
func (m *Auth) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.IsEnabled() {
			next(w, r)
			return
		}

		cookie, err := r.Cookie(CookieName)
		if err != nil || !m.ValidateKey(cookie.Value) {
			responses.Error(w, http.StatusUnauthorized, "unauthorized", "Valid API key required")
			return
		}

		next(w, r)
	}
}

// ProtectHandler wraps an http.Handler to require valid API key.
func (m *Auth) ProtectHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(CookieName)
		if err != nil || !m.ValidateKey(cookie.Value) {
			responses.Error(w, http.StatusUnauthorized, "unauthorized", "Valid API key required")
			return
		}

		next.ServeHTTP(w, r)
	})
}
