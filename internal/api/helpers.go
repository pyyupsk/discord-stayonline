package api

import (
	"encoding/json"
	"net/http"
)

// maxBodySize is the maximum request body size (1 MB).
const maxBodySize = 1 << 20

// limitBody wraps the request body with a size limiter to prevent DoS attacks.
func limitBody(r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodySize)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
