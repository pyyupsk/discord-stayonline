package config

import "time"

// Setting represents the global settings table (single row with id=1).
type Setting struct {
	ID              int       `gorm:"primaryKey;default:1"`
	Status          string    `gorm:"type:varchar(10);not null;default:'online'"`
	TOSAcknowledged bool      `gorm:"column:tos_acknowledged;not null;default:false"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM.
func (Setting) TableName() string {
	return "settings"
}

// Server represents a configured Discord server/channel connection.
type Server struct {
	ID             string    `gorm:"type:varchar(32);primaryKey"`
	GuildID        string    `gorm:"type:varchar(20);not null;index:idx_servers_guild_id"`
	GuildName      *string   `gorm:"type:varchar(100)"`
	GuildIcon      *string   `gorm:"type:varchar(64)"`
	ChannelID      string    `gorm:"type:varchar(20);not null"`
	ChannelName    *string   `gorm:"type:varchar(100)"`
	ConnectOnStart bool      `gorm:"column:connect_on_start;not null;default:false"`
	Priority       int       `gorm:"not null;default:1;index:idx_servers_priority"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM.
func (Server) TableName() string {
	return "servers"
}

// Log represents a stored log entry.
type Log struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Level     string    `gorm:"type:varchar(10);not null;index:idx_logs_level"`
	Message   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_logs_created_at"`
}

// TableName specifies the table name for GORM.
func (Log) TableName() string {
	return "logs"
}

// Session holds Discord Gateway session data for resumption.
type Session struct {
	ServerID  string    `gorm:"type:varchar(32);primaryKey"`
	SessionID string    `gorm:"column:session_id;type:varchar(64);not null"`
	Sequence  int       `gorm:"not null;default:0"`
	ResumeURL string    `gorm:"column:resume_url;type:varchar(255);not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM.
func (Session) TableName() string {
	return "sessions"
}
