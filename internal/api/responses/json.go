// Package responses provides HTTP response utilities.
package responses

import (
	"encoding/json"
	"net/http"
)

// maxBodySize is the maximum request body size (1 MB).
const maxBodySize = 1 << 20

// Common error messages for logging and responses.
const (
	ErrLoadConfig    = "Failed to load config"
	ErrLoadConfigMsg = "Failed to load configuration"
	ErrSaveConfig    = "Failed to save config"
	ErrSaveConfigMsg = "Failed to save configuration"
)

// LimitBody wraps the request body with a size limiter to prevent DoS attacks.
func LimitBody(r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodySize)
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Error writes a standardized JSON error response.
func Error(w http.ResponseWriter, status int, errCode, message string) {
	JSON(w, status, map[string]string{
		"error":   errCode,
		"message": message,
	})
}
