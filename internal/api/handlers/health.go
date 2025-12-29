package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/hako/durafmt"
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
		Uptime:      durafmt.Parse(uptime).String(),
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
			Alloc:      humanize.Bytes(memStats.Alloc),
			TotalAlloc: humanize.Bytes(memStats.TotalAlloc),
			Sys:        humanize.Bytes(memStats.Sys),
			NumGC:      memStats.NumGC,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
