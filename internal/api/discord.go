package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// DiscordHandler handles Discord API lookups.
type DiscordHandler struct {
	token  string
	client *http.Client
	cache  map[string]*cacheEntry
	mu     sync.RWMutex
	logger *slog.Logger
}

type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// GuildInfo contains basic guild information.
type GuildInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

// ChannelInfo contains basic channel information.
type ChannelInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	GuildID string `json:"guild_id,omitempty"`
	Type    int    `json:"type"`
}

// ServerInfo combines guild and channel info for a server entry.
type ServerInfo struct {
	GuildID     string `json:"guild_id"`
	GuildName   string `json:"guild_name"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
}

// NewDiscordHandler creates a new Discord API handler.
func NewDiscordHandler(logger *slog.Logger) *DiscordHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &DiscordHandler{
		token: os.Getenv("DISCORD_TOKEN"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:  make(map[string]*cacheEntry),
		logger: logger.With("handler", "discord"),
	}
}

const (
	discordAPIBase = "https://discord.com/api/v10"
	cacheTTL       = 5 * time.Minute
)

func (h *DiscordHandler) getFromCache(key string) (interface{}, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if entry, ok := h.cache[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.data, true
		}
	}
	return nil, false
}

func (h *DiscordHandler) setCache(key string, data interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cache[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

func (h *DiscordHandler) fetchFromDiscord(endpoint string, result interface{}) error {
	req, err := http.NewRequest("GET", discordAPIBase+endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", h.token)

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discord API returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// GetGuild fetches guild info from Discord API.
func (h *DiscordHandler) GetGuild(guildID string) (*GuildInfo, error) {
	cacheKey := "guild:" + guildID
	if cached, ok := h.getFromCache(cacheKey); ok {
		return cached.(*GuildInfo), nil
	}

	var guild GuildInfo
	if err := h.fetchFromDiscord("/guilds/"+guildID, &guild); err != nil {
		return nil, err
	}

	h.setCache(cacheKey, &guild)
	return &guild, nil
}

// GetChannel fetches channel info from Discord API.
func (h *DiscordHandler) GetChannel(channelID string) (*ChannelInfo, error) {
	cacheKey := "channel:" + channelID
	if cached, ok := h.getFromCache(cacheKey); ok {
		return cached.(*ChannelInfo), nil
	}

	var channel ChannelInfo
	if err := h.fetchFromDiscord("/channels/"+channelID, &channel); err != nil {
		return nil, err
	}

	h.setCache(cacheKey, &channel)
	return &channel, nil
}

// GetServerInfo handles GET /api/discord/server-info?guild_id=...&channel_id=...
func (h *DiscordHandler) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	guildID := r.URL.Query().Get("guild_id")
	channelID := r.URL.Query().Get("channel_id")

	if guildID == "" || channelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "guild_id and channel_id are required",
		})
		return
	}

	info := ServerInfo{
		GuildID:   guildID,
		ChannelID: channelID,
	}

	// Fetch guild info
	guild, err := h.GetGuild(guildID)
	if err != nil {
		h.logger.Warn("Failed to fetch guild info", "guild_id", guildID, "error", err)
		info.GuildName = ""
	} else {
		info.GuildName = guild.Name
	}

	// Fetch channel info
	channel, err := h.GetChannel(channelID)
	if err != nil {
		h.logger.Warn("Failed to fetch channel info", "channel_id", channelID, "error", err)
		info.ChannelName = ""
	} else {
		info.ChannelName = channel.Name
	}

	writeJSON(w, http.StatusOK, info)
}

// GetBulkServerInfo handles POST /api/discord/bulk-info with array of {guild_id, channel_id}
func (h *DiscordHandler) GetBulkServerInfo(w http.ResponseWriter, r *http.Request) {
	var requests []struct {
		GuildID   string `json:"guild_id"`
		ChannelID string `json:"channel_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON request body",
		})
		return
	}

	results := make([]ServerInfo, len(requests))

	for i, req := range requests {
		info := ServerInfo{
			GuildID:   req.GuildID,
			ChannelID: req.ChannelID,
		}

		// Fetch guild info
		guild, err := h.GetGuild(req.GuildID)
		if err != nil {
			h.logger.Debug("Failed to fetch guild info", "guild_id", req.GuildID, "error", err)
		} else {
			info.GuildName = guild.Name
		}

		// Fetch channel info
		channel, err := h.GetChannel(req.ChannelID)
		if err != nil {
			h.logger.Debug("Failed to fetch channel info", "channel_id", req.ChannelID, "error", err)
		} else {
			info.ChannelName = channel.Name
		}

		results[i] = info
	}

	writeJSON(w, http.StatusOK, results)
}
