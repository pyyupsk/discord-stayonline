package responses

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

const maxBodySize = 1 << 20

const (
	ErrLoadConfig    = "Failed to load config"
	ErrLoadConfigMsg = "Failed to load configuration"
	ErrSaveConfig    = "Failed to save config"
	ErrSaveConfigMsg = "Failed to save configuration"
)

func LimitBody(r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodySize)
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, errCode, message string) {
	JSON(w, status, map[string]string{
		"error":   errCode,
		"message": message,
	})
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, logger *slog.Logger, v any) bool {
	LimitBody(r)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		logger.Error("Failed to decode request", "error", err)
		Error(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return false
	}
	return true
}
