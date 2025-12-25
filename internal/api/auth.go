package api

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	cookieName   = "api_key"
	cookieMaxAge = 24 * 60 * 60 // 24 hours in seconds
)

// AuthMiddleware provides API key authentication.
type AuthMiddleware struct {
	apiKey string
	logger *slog.Logger
}

// NewAuthMiddleware creates a new auth middleware.
func NewAuthMiddleware(logger *slog.Logger) *AuthMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthMiddleware{
		apiKey: os.Getenv("API_KEY"),
		logger: logger.With("middleware", "auth"),
	}
}

// IsEnabled returns true if API key authentication is configured.
func (m *AuthMiddleware) IsEnabled() bool {
	return m.apiKey != ""
}

// Protect wraps a handler to require valid API key.
func (m *AuthMiddleware) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no API key configured, allow all requests
		if !m.IsEnabled() {
			next(w, r)
			return
		}

		// Check cookie for API key
		cookie, err := r.Cookie(cookieName)
		if err != nil || !m.validateKey(cookie.Value) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error":   "unauthorized",
				"message": "Valid API key required",
			})
			return
		}

		next(w, r)
	}
}

// ProtectHandler wraps an http.Handler to require valid API key.
func (m *AuthMiddleware) ProtectHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no API key configured, allow all requests
		if !m.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Check cookie for API key
		cookie, err := r.Cookie(cookieName)
		if err != nil || !m.validateKey(cookie.Value) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error":   "unauthorized",
				"message": "Valid API key required",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) validateKey(key string) bool {
	return subtle.ConstantTimeCompare([]byte(key), []byte(m.apiKey)) == 1
}

// AuthHandler handles login/logout requests.
type AuthHandler struct {
	middleware *AuthMiddleware
	logger     *slog.Logger
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(middleware *AuthMiddleware, logger *slog.Logger) *AuthHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthHandler{
		middleware: middleware,
		logger:     logger.With("handler", "auth"),
	}
}

// Login handles POST /api/auth/login requests.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey string `json:"api_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON request body",
		})
		return
	}

	if !h.middleware.validateKey(req.APIKey) {
		h.logger.Warn("Failed login attempt")
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "Invalid API key",
		})
		return
	}

	// Set HTTP-only cookie with API key
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    req.APIKey,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
	})

	h.logger.Info("Successful login")
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Logged in successfully",
	})
}

// Logout handles POST /api/auth/logout requests.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})

	h.logger.Info("User logged out")
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Logged out successfully",
	})
}

// Check handles GET /api/auth/check requests.
func (h *AuthHandler) Check(w http.ResponseWriter, r *http.Request) {
	// If auth not enabled, always authenticated
	if !h.middleware.IsEnabled() {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": true,
			"auth_required": false,
		})
		return
	}

	// Check if valid cookie exists
	cookie, err := r.Cookie(cookieName)
	authenticated := err == nil && h.middleware.validateKey(cookie.Value)

	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": authenticated,
		"auth_required": true,
	})
}
