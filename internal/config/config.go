// Package config provides configuration types and persistence for the Discord presence service.
package config

// Status represents the desired Discord presence status.
type Status string

const (
	StatusOnline Status = "online"
	StatusIdle   Status = "idle"
	StatusDND    Status = "dnd"
)

// ServerEntry represents a configured Discord server/channel connection.
type ServerEntry struct {
	// ID is the unique identifier for this entry (UUID or nanoid)
	ID string `json:"id"`

	// GuildID is the Discord server ID (snowflake, 17-19 digit string)
	GuildID string `json:"guild_id"`

	// GuildName is an optional friendly name for the server
	GuildName string `json:"guild_name,omitempty"`

	// ChannelID is the Discord voice channel ID (snowflake)
	ChannelID string `json:"channel_id"`

	// ChannelName is an optional friendly name for the channel
	ChannelName string `json:"channel_name,omitempty"`

	// ConnectOnStart determines if this connection should auto-connect when service starts
	ConnectOnStart bool `json:"connect_on_start"`

	// Priority is the connection priority (1 = highest)
	Priority int `json:"priority"`
}

// Configuration represents the complete persisted configuration state.
type Configuration struct {
	// Servers is the list of configured server entries (max 35)
	Servers []ServerEntry `json:"servers"`

	// Status is the global presence status for the account (online, idle, dnd)
	Status Status `json:"status"`

	// TOSAcknowledged indicates the user has acknowledged the TOS warning
	TOSAcknowledged bool `json:"tos_acknowledged"`
}

// MaxServerEntries is the maximum number of server entries allowed.
const MaxServerEntries = 35

// Validate checks if the server entry has valid values.
func (s *ServerEntry) Validate() error {
	if s.ID == "" {
		return ErrEmptyID
	}
	if s.GuildID == "" {
		return ErrEmptyGuildID
	}
	if s.ChannelID == "" {
		return ErrEmptyChannelID
	}
	if s.Priority < 1 {
		return ErrInvalidPriority
	}
	return nil
}

// Validate checks if the configuration is valid.
func (c *Configuration) Validate() error {
	if len(c.Servers) > MaxServerEntries {
		return ErrTooManyServers
	}
	// Validate status (allow empty for backwards compatibility, defaults to online)
	if c.Status != "" && c.Status != StatusOnline && c.Status != StatusIdle && c.Status != StatusDND {
		return ErrInvalidStatus
	}
	for i := range c.Servers {
		if err := c.Servers[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Default returns a default empty configuration.
func Default() *Configuration {
	return &Configuration{
		Servers:         []ServerEntry{},
		Status:          StatusOnline,
		TOSAcknowledged: false,
	}
}
