package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pyyupsk/discord-stayonline/internal/api/responses"
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
	data      any
	expiresAt time.Time
}

// UserInfo contains basic Discord user information.
type UserInfo struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	GlobalName    string `json:"global_name,omitempty"`
	Avatar        string `json:"avatar,omitempty"`
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

// VoiceChannelInfo contains voice channel information.
type VoiceChannelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

const (
	discordAPIBase        = "https://discord.com/api/v10"
	cacheTTL              = 5 * time.Minute
	channelTypeGuildVoice = 2
	channelTypeGuildStage = 13
)

func NewDiscordHandler(logger *slog.Logger) *DiscordHandler {
	return &DiscordHandler{
		token: os.Getenv("DISCORD_TOKEN"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:  make(map[string]*cacheEntry),
		logger: logger.With("handler", "discord"),
	}
}

func (h *DiscordHandler) getFromCache(key string) (any, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if entry, ok := h.cache[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.data, true
		}
	}
	return nil, false
}

func (h *DiscordHandler) setCache(key string, data any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cache[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

func (h *DiscordHandler) fetchFromDiscord(endpoint string, result any) error {
	req, err := http.NewRequest("GET", discordAPIBase+endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", h.token)

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

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

// GetServerInfo handles GET /api/discord/server-info
func (h *DiscordHandler) GetServerInfo(w http.ResponseWriter, r *http.Request) {
	guildID := r.URL.Query().Get("guild_id")
	channelID := r.URL.Query().Get("channel_id")

	if guildID == "" || channelID == "" {
		responses.Error(w, http.StatusBadRequest, "invalid_request", "guild_id and channel_id are required")
		return
	}

	info := ServerInfo{
		GuildID:   guildID,
		ChannelID: channelID,
	}

	if guild, err := h.GetGuild(guildID); err != nil {
		h.logger.Warn("Failed to fetch guild info", "guild_id", guildID, "error", err)
	} else {
		info.GuildName = guild.Name
	}

	if channel, err := h.GetChannel(channelID); err != nil {
		h.logger.Warn("Failed to fetch channel info", "channel_id", channelID, "error", err)
	} else {
		info.ChannelName = channel.Name
	}

	responses.JSON(w, http.StatusOK, info)
}

// GetCurrentUser handles GET /api/discord/user
func (h *DiscordHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	cacheKey := "user:me"
	if cached, ok := h.getFromCache(cacheKey); ok {
		responses.JSON(w, http.StatusOK, cached)
		return
	}

	var user UserInfo
	if err := h.fetchFromDiscord("/users/@me", &user); err != nil {
		h.logger.Error("Failed to fetch current user", "error", err)
		responses.Error(w, http.StatusInternalServerError, "discord_error", "Failed to fetch user from Discord")
		return
	}

	h.setCache(cacheKey, user)
	responses.JSON(w, http.StatusOK, user)
}

// GetUserGuilds handles GET /api/discord/guilds
func (h *DiscordHandler) GetUserGuilds(w http.ResponseWriter, r *http.Request) {
	cacheKey := "user:guilds"
	if cached, ok := h.getFromCache(cacheKey); ok {
		responses.JSON(w, http.StatusOK, cached)
		return
	}

	var guilds []GuildInfo
	if err := h.fetchFromDiscord("/users/@me/guilds", &guilds); err != nil {
		h.logger.Error("Failed to fetch user guilds", "error", err)
		responses.Error(w, http.StatusInternalServerError, "discord_error", "Failed to fetch guilds from Discord")
		return
	}

	h.setCache(cacheKey, guilds)
	responses.JSON(w, http.StatusOK, guilds)
}

// GetGuildChannels handles GET /api/discord/guilds/{id}/channels
func (h *DiscordHandler) GetGuildChannels(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api/discord/guilds/")
	path = strings.TrimSuffix(path, "/channels")
	guildID := path

	if guildID == "" || strings.Contains(guildID, "/") {
		responses.Error(w, http.StatusBadRequest, "invalid_request", "Guild ID is required")
		return
	}

	cacheKey := "guild:channels:" + guildID
	if cached, ok := h.getFromCache(cacheKey); ok {
		responses.JSON(w, http.StatusOK, cached)
		return
	}

	var channels []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Type     int    `json:"type"`
		Position int    `json:"position"`
	}

	if err := h.fetchFromDiscord("/guilds/"+guildID+"/channels", &channels); err != nil {
		h.logger.Error("Failed to fetch guild channels", "guild_id", guildID, "error", err)
		responses.Error(w, http.StatusInternalServerError, "discord_error", "Failed to fetch channels from Discord")
		return
	}

	var voiceChannels []VoiceChannelInfo
	for _, ch := range channels {
		if ch.Type == channelTypeGuildVoice || ch.Type == channelTypeGuildStage {
			voiceChannels = append(voiceChannels, VoiceChannelInfo{
				ID:       ch.ID,
				Name:     ch.Name,
				Position: ch.Position,
			})
		}
	}

	h.setCache(cacheKey, voiceChannels)
	responses.JSON(w, http.StatusOK, voiceChannels)
}

// GetBulkServerInfo handles POST /api/discord/bulk-info
func (h *DiscordHandler) GetBulkServerInfo(w http.ResponseWriter, r *http.Request) {
	var requests []struct {
		GuildID   string `json:"guild_id"`
		ChannelID string `json:"channel_id"`
	}

	if !responses.DecodeJSON(w, r, h.logger, &requests) {
		return
	}

	results := make([]ServerInfo, len(requests))
	for i, req := range requests {
		info := ServerInfo{
			GuildID:   req.GuildID,
			ChannelID: req.ChannelID,
		}

		if guild, err := h.GetGuild(req.GuildID); err == nil {
			info.GuildName = guild.Name
		}
		if channel, err := h.GetChannel(req.ChannelID); err == nil {
			info.ChannelName = channel.Name
		}

		results[i] = info
	}

	responses.JSON(w, http.StatusOK, results)
}
