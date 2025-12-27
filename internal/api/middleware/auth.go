// Package middleware provides HTTP middleware components.
package middleware

import (
	"crypto/subtle"
	"errors"
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

// ErrAPIKeyRequired is returned when API_KEY environment variable is not set.
var ErrAPIKeyRequired = errors.New("API_KEY environment variable is required for security")

// Auth provides API key authentication.
type Auth struct {
	apiKey string
	logger *slog.Logger
}

// NewAuth creates a new auth middleware.
// Returns an error if API_KEY environment variable is not set.
func NewAuth(logger *slog.Logger) (*Auth, error) {
	if logger == nil {
		logger = slog.Default()
	}
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}
	return &Auth{
		apiKey: apiKey,
		logger: logger.With("middleware", "auth"),
	}, nil
}

// ValidateKey checks if the provided key matches the configured API key.
func (m *Auth) ValidateKey(key string) bool {
	return subtle.ConstantTimeCompare([]byte(key), []byte(m.apiKey)) == 1
}

// Protect wraps a handler to require valid API key.
func (m *Auth) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		cookie, err := r.Cookie(CookieName)
		if err != nil || !m.ValidateKey(cookie.Value) {
			responses.Error(w, http.StatusUnauthorized, "unauthorized", "Valid API key required")
			return
		}

		next.ServeHTTP(w, r)
	})
}
