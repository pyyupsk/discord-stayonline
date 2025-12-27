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
	CookieName   = "api_key"
	CookieMaxAge = 7 * 24 * 60 * 60
)

var ErrAPIKeyRequired = errors.New("API_KEY environment variable is required for security")

type Auth struct {
	apiKey string
	logger *slog.Logger
}

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

func (m *Auth) ValidateKey(key string) bool {
	return subtle.ConstantTimeCompare([]byte(key), []byte(m.apiKey)) == 1
}

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
