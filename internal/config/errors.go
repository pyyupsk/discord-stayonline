package config

import "errors"

var (
	ErrEmptyID         = errors.New("server entry ID cannot be empty")
	ErrEmptyGuildID    = errors.New("guild_id cannot be empty")
	ErrEmptyChannelID  = errors.New("channel_id cannot be empty")
	ErrInvalidStatus   = errors.New("status must be online, idle, or dnd")
	ErrInvalidPriority = errors.New("priority must be a positive integer")
	ErrTooManyServers  = errors.New("maximum 35 server entries allowed")
	ErrConfigNotFound  = errors.New("configuration file not found")
)
