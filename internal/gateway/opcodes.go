// Package gateway provides the Discord Gateway WebSocket client.
package gateway

import "encoding/json"

// Gateway opcodes as defined by Discord.
// See: https://discord.com/developers/docs/topics/opcodes-and-status-codes#gateway-opcodes
const (
	OpDispatch         = 0  // Dispatch: An event was dispatched (S→C)
	OpHeartbeat        = 1  // Heartbeat: Fired periodically to keep connection alive (C→S)
	OpIdentify         = 2  // Identify: Starts a new session (C→S)
	OpPresenceUpdate   = 3  // Presence Update: Update client's presence (C→S)
	OpVoiceStateUpdate = 4  // Voice State Update: Join/leave voice channel (C→S)
	OpResume           = 6  // Resume: Resume a previous session (C→S)
	OpReconnect        = 7  // Reconnect: Server requests client reconnect (S→C)
	OpRequestMembers   = 8  // Request Guild Members: Request guild members (C→S)
	OpInvalidSession   = 9  // Invalid Session: Session invalidated (S→C)
	OpHello            = 10 // Hello: Sent after connecting, contains heartbeat interval (S→C)
	OpHeartbeatAck     = 11 // Heartbeat ACK: Sent in response to receiving heartbeat (S→C)
)

// Gateway close codes.
// See: https://discord.com/developers/docs/topics/opcodes-and-status-codes#gateway-close-event-codes
const (
	CloseUnknownError         = 4000 // Unknown error
	CloseUnknownOpcode        = 4001 // Unknown opcode
	CloseDecodeError          = 4002 // Decode error
	CloseNotAuthenticated     = 4003 // Not authenticated
	CloseAuthenticationFailed = 4004 // Authentication failed (FATAL)
	CloseAlreadyAuthenticated = 4005 // Already authenticated
	CloseInvalidSeq           = 4007 // Invalid seq
	CloseRateLimited          = 4008 // Rate limited
	CloseSessionTimedOut      = 4009 // Session timed out
	CloseInvalidShard         = 4010 // Invalid shard (FATAL)
	CloseShardingRequired     = 4011 // Sharding required (FATAL)
	CloseInvalidAPIVersion    = 4012 // Invalid API version (FATAL)
	CloseInvalidIntents       = 4013 // Invalid intent(s) (FATAL)
	CloseDisallowedIntents    = 4014 // Disallowed intent(s) (FATAL)
)

// IsFatalCloseCode returns true if the close code indicates a non-recoverable error.
func IsFatalCloseCode(code int) bool {
	switch code {
	case CloseAuthenticationFailed,
		CloseInvalidShard,
		CloseShardingRequired,
		CloseInvalidAPIVersion,
		CloseInvalidIntents,
		CloseDisallowedIntents:
		return true
	default:
		return false
	}
}

// GatewayMessage represents a generic Gateway protocol message.
type GatewayMessage struct {
	Op       int             `json:"op"`          // Opcode for the payload
	Data     json.RawMessage `json:"d"`           // Event data (varies by opcode)
	Sequence *int            `json:"s,omitempty"` // Sequence number (dispatch only)
	Type     string          `json:"t,omitempty"` // Event name (dispatch only)
}

// HelloData is the payload for OP 10 (Hello).
type HelloData struct {
	HeartbeatInterval int `json:"heartbeat_interval"` // Interval in milliseconds
}

// ReadyData is the payload for the READY dispatch event.
type ReadyData struct {
	Version   int    `json:"v"`
	SessionID string `json:"session_id"`
	ResumeURL string `json:"resume_gateway_url"`
}

// IdentifyData is the payload for OP 2 (Identify).
type IdentifyData struct {
	Token          string             `json:"token"`
	Properties     IdentifyProperties `json:"properties"`
	Presence       *PresenceData      `json:"presence,omitempty"`
	Compress       bool               `json:"compress,omitempty"`
	LargeThreshold int                `json:"large_threshold,omitempty"`
}

// IdentifyProperties contains client identification info.
// Note: User tokens use keys without $ prefix (unlike bot tokens).
type IdentifyProperties struct {
	OS      string `json:"os"`      // Operating system
	Browser string `json:"browser"` // Browser identification
	Device  string `json:"device"`  // Device identification
}

// PresenceData is the payload for OP 3 (Presence Update).
// MANUAL REVIEW: Verify status values are valid Discord presence states.
type PresenceData struct {
	Since      *int64     `json:"since"`      // Unix time in ms of when the client went idle, null if not idle
	Activities []Activity `json:"activities"` // Array of activity objects
	Status     string     `json:"status"`     // Status: online, dnd, idle, invisible, offline
	AFK        bool       `json:"afk"`        // Whether the client is AFK
}

// Activity represents a Discord activity (game, streaming, etc).
type Activity struct {
	Name string `json:"name"`
	Type int    `json:"type"` // 0=Playing, 1=Streaming, 2=Listening, 3=Watching, 4=Custom, 5=Competing
}

// VoiceStateData is the payload for OP 4 (Voice State Update).
// MANUAL REVIEW: Verify guild_id and channel_id are correct snowflake formats.
type VoiceStateData struct {
	GuildID   string  `json:"guild_id"`   // ID of the guild
	ChannelID *string `json:"channel_id"` // ID of the voice channel to join (null to disconnect)
	SelfMute  bool    `json:"self_mute"`  // Is the client muted
	SelfDeaf  bool    `json:"self_deaf"`  // Is the client deafened
}

// ResumeData is the payload for OP 6 (Resume).
type ResumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int    `json:"seq"`
}
