package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/pyyupsk/discord-stayonline/internal/manager"
	"github.com/pyyupsk/discord-stayonline/internal/ws"
)

var startTime = time.Now()

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status      string          `json:"status"`
	Uptime      string          `json:"uptime"`
	UptimeSecs  int64           `json:"uptime_secs"`
	Timestamp   string          `json:"timestamp"`
	Connections ConnectionsInfo `json:"connections"`
	Runtime     RuntimeInfo     `json:"runtime"`
	Memory      MemoryInfo      `json:"memory"`
}

// ConnectionsInfo contains connection statistics.
type ConnectionsInfo struct {
	ActiveSessions   int               `json:"active_sessions"`
	WebSocketClients int               `json:"websocket_clients"`
	SessionStatuses  map[string]string `json:"session_statuses,omitempty"`
}

// RuntimeInfo contains Go runtime information.
type RuntimeInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
}

// MemoryInfo contains memory statistics.
type MemoryInfo struct {
	Alloc      string `json:"alloc"`
	TotalAlloc string `json:"total_alloc"`
	Sys        string `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// HealthHandler handles health check requests.
type HealthHandler struct {
	manager *manager.SessionManager
	hub     *ws.Hub
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(mgr *manager.SessionManager, hub *ws.Hub) *HealthHandler {
	return &HealthHandler{
		manager: mgr,
		hub:     hub,
	}
}

// Health handles GET/HEAD /health requests.
// Returns detailed health information as JSON.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	// For HEAD requests, just return 200
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	uptime := time.Since(startTime)

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Build connection info
	connInfo := ConnectionsInfo{
		ActiveSessions:   0,
		WebSocketClients: 0,
	}

	if h.manager != nil {
		statuses := h.manager.GetAllStatuses()
		connInfo.ActiveSessions = len(statuses)
		connInfo.SessionStatuses = make(map[string]string)
		for id, status := range statuses {
			connInfo.SessionStatuses[id] = string(status)
		}
	}

	if h.hub != nil {
		connInfo.WebSocketClients = h.hub.ClientCount()
	}

	response := HealthResponse{
		Status:      "healthy",
		Uptime:      formatDuration(uptime),
		UptimeSecs:  int64(uptime.Seconds()),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Connections: connInfo,
		Runtime: RuntimeInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			GOOS:         runtime.GOOS,
			GOARCH:       runtime.GOARCH,
		},
		Memory: MemoryInfo{
			Alloc:      formatBytes(memStats.Alloc),
			TotalAlloc: formatBytes(memStats.TotalAlloc),
			Sys:        formatBytes(memStats.Sys),
			NumGC:      memStats.NumGC,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return formatDurationWithDays(days, hours, minutes, seconds)
	}
	if hours > 0 {
		return formatDurationWithHours(hours, minutes, seconds)
	}
	if minutes > 0 {
		return formatDurationWithMinutes(minutes, seconds)
	}
	return formatDurationSecondsOnly(seconds)
}

func formatDurationWithDays(days, hours, minutes, seconds int) string {
	return formatInt(days) + "d " + formatInt(hours) + "h " + formatInt(minutes) + "m " + formatInt(seconds) + "s"
}

func formatDurationWithHours(hours, minutes, seconds int) string {
	return formatInt(hours) + "h " + formatInt(minutes) + "m " + formatInt(seconds) + "s"
}

func formatDurationWithMinutes(minutes, seconds int) string {
	return formatInt(minutes) + "m " + formatInt(seconds) + "s"
}

func formatDurationSecondsOnly(seconds int) string {
	return formatInt(seconds) + "s"
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}

// formatBytes formats bytes as a human-readable string.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return formatUint(b) + " B"
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return formatFloat(float64(b)/float64(div)) + " " + units[exp]
}

func formatUint(n uint64) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}

func formatFloat(f float64) string {
	intPart := int(f)
	decPart := int((f - float64(intPart)) * 100)
	if decPart == 0 {
		return formatInt(intPart)
	}
	return formatInt(intPart) + "." + formatInt(decPart)
}
