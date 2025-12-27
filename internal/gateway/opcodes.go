package gateway

import "encoding/json"

const (
	OpDispatch         = 0
	OpHeartbeat        = 1
	OpIdentify         = 2
	OpPresenceUpdate   = 3
	OpVoiceStateUpdate = 4
	OpResume           = 6
	OpReconnect        = 7
	OpRequestMembers   = 8
	OpInvalidSession   = 9
	OpHello            = 10
	OpHeartbeatAck     = 11
)

const (
	CloseUnknownError         = 4000
	CloseUnknownOpcode        = 4001
	CloseDecodeError          = 4002
	CloseNotAuthenticated     = 4003
	CloseAuthenticationFailed = 4004
	CloseAlreadyAuthenticated = 4005
	CloseInvalidSeq           = 4007
	CloseRateLimited          = 4008
	CloseSessionTimedOut      = 4009
	CloseInvalidShard         = 4010
	CloseShardingRequired     = 4011
	CloseInvalidAPIVersion    = 4012
	CloseInvalidIntents       = 4013
	CloseDisallowedIntents    = 4014
)

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

type GatewayMessage struct {
	Op       int             `json:"op"`
	Data     json.RawMessage `json:"d"`
	Sequence *int            `json:"s,omitempty"`
	Type     string          `json:"t,omitempty"`
}

type HelloData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type ReadyData struct {
	Version   int    `json:"v"`
	SessionID string `json:"session_id"`
	ResumeURL string `json:"resume_gateway_url"`
}

type IdentifyData struct {
	Token          string             `json:"token"`
	Properties     IdentifyProperties `json:"properties"`
	Presence       *PresenceData      `json:"presence,omitempty"`
	Compress       bool               `json:"compress,omitempty"`
	LargeThreshold int                `json:"large_threshold,omitempty"`
}

type IdentifyProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type PresenceData struct {
	Since      *int64     `json:"since"`
	Activities []Activity `json:"activities"`
	Status     string     `json:"status"`
	AFK        bool       `json:"afk"`
}

type Activity struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

type VoiceStateData struct {
	GuildID   string  `json:"guild_id"`
	ChannelID *string `json:"channel_id"`
	SelfMute  bool    `json:"self_mute"`
	SelfDeaf  bool    `json:"self_deaf"`
}

type ResumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int    `json:"seq"`
}
