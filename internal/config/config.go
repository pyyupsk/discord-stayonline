package config

type Status string

const (
	StatusOnline Status = "online"
	StatusIdle   Status = "idle"
	StatusDND    Status = "dnd"
)

type ServerEntry struct {
	ID             string `json:"id"`
	GuildID        string `json:"guild_id"`
	GuildName      string `json:"guild_name,omitempty"`
	GuildIcon      string `json:"guild_icon,omitempty"`
	ChannelID      string `json:"channel_id"`
	ChannelName    string `json:"channel_name,omitempty"`
	ConnectOnStart bool   `json:"connect_on_start"`
	Priority       int    `json:"priority"`
}

type Configuration struct {
	Servers         []ServerEntry `json:"servers"`
	Status          Status        `json:"status"`
	TOSAcknowledged bool          `json:"tos_acknowledged"`
}

const MaxServerEntries = 35

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

func (c *Configuration) Validate() error {
	if len(c.Servers) > MaxServerEntries {
		return ErrTooManyServers
	}
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

func Default() *Configuration {
	return &Configuration{
		Servers:         []ServerEntry{},
		Status:          StatusOnline,
		TOSAcknowledged: false,
	}
}

type SessionState struct {
	ServerID  string `json:"server_id"`
	SessionID string `json:"session_id"`
	Sequence  int    `json:"sequence"`
	ResumeURL string `json:"resume_url"`
}
