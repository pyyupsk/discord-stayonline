package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/pyyupsk/discord-stayonline/internal/api/middleware"
	"github.com/pyyupsk/discord-stayonline/internal/api/responses"
)

// AuthHandler handles login/logout requests.
type AuthHandler struct {
	auth   *middleware.Auth
	logger *slog.Logger
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(auth *middleware.Auth, logger *slog.Logger) *AuthHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthHandler{
		auth:   auth,
		logger: logger.With("handler", "auth"),
	}
}

// Login handles POST /api/auth/login requests.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey string `json:"api_key"`
	}

	responses.LimitBody(r)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.Error(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	if !h.auth.ValidateKey(req.APIKey) {
		h.logger.Warn("Failed login attempt")
		responses.Error(w, http.StatusUnauthorized, "unauthorized", "Invalid API key")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    req.APIKey,
		Path:     "/",
		MaxAge:   middleware.CookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
	})

	h.logger.Info("Successful login")
	responses.JSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Logged in successfully",
	})
}

// Logout handles POST /api/auth/logout requests.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})

	h.logger.Info("User logged out")
	responses.JSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Logged out successfully",
	})
}

// Check handles GET /api/auth/check requests.
func (h *AuthHandler) Check(w http.ResponseWriter, r *http.Request) {
	if !h.auth.IsEnabled() {
		responses.JSON(w, http.StatusOK, map[string]any{
			"authenticated": true,
			"auth_required": false,
		})
		return
	}

	cookie, err := r.Cookie(middleware.CookieName)
	authenticated := err == nil && h.auth.ValidateKey(cookie.Value)

	responses.JSON(w, http.StatusOK, map[string]any{
		"authenticated": authenticated,
		"auth_required": true,
	})
}
