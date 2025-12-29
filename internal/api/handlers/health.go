package handlers

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

type HealthResponse struct {
	Status      string          `json:"status"`
	Uptime      string          `json:"uptime"`
	UptimeSecs  int64           `json:"uptime_secs"`
	Timestamp   string          `json:"timestamp"`
	Connections ConnectionsInfo `json:"connections"`
	Runtime     RuntimeInfo     `json:"runtime"`
	Memory      MemoryInfo      `json:"memory"`
}

type ConnectionsInfo struct {
	ActiveSessions   int               `json:"active_sessions"`
	WebSocketClients int               `json:"websocket_clients"`
	SessionStatuses  map[string]string `json:"session_statuses,omitempty"`
}

type RuntimeInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
}

type MemoryInfo struct {
	Alloc      string `json:"alloc"`
	TotalAlloc string `json:"total_alloc"`
	Sys        string `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

type HealthHandler struct {
	manager *manager.SessionManager
	hub     *ws.Hub
}

func NewHealthHandler(mgr *manager.SessionManager, hub *ws.Hub) *HealthHandler {
	return &HealthHandler{
		manager: mgr,
		hub:     hub,
	}
}

// Health handles GET/HEAD /health requests.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	uptime := time.Since(startTime)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

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

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m " + strconv.Itoa(seconds) + "s"
	}
	if hours > 0 {
		return strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m " + strconv.Itoa(seconds) + "s"
	}
	if minutes > 0 {
		return strconv.Itoa(minutes) + "m " + strconv.Itoa(seconds) + "s"
	}
	return strconv.Itoa(seconds) + "s"
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return strconv.FormatUint(b, 10) + " B"
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	val := float64(b) / float64(div)
	return strconv.FormatFloat(val, 'f', 2, 64) + " " + units[exp]
}
