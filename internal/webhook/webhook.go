package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Notifier struct {
	webhookURL string
	client     *http.Client
	logger     *slog.Logger
}

type Embed struct {
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Color       int     `json:"color,omitempty"`
	Timestamp   string  `json:"timestamp,omitempty"`
	Fields      []Field `json:"fields,omitempty"`
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type WebhookPayload struct {
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Content   string  `json:"content,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

const (
	ColorRed    = 0xed4245
	ColorGreen  = 0x57f287
	ColorYellow = 0xfee75c
)

const (
	WebhookUsername  = "Discord Stay Online"
	WebhookAvatarURL = "https://raw.githubusercontent.com/pyyupsk/discord-stayonline/main/web/public/android-chrome-512x512.png"
)

const FieldServerID = "Server ID"

func NewNotifier(webhookURL string, logger *slog.Logger) *Notifier {
	if webhookURL == "" {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Notifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger.With("component", "webhook"),
	}
}

func (n *Notifier) NotifyDown(serverID, guildID, channelID, reason string) {
	if n == nil {
		return
	}

	embed := Embed{
		Title:       "🔴 Connection Lost",
		Description: fmt.Sprintf("Connection to <#%s> has been lost.", channelID),
		Color:       ColorRed,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []Field{
			{Name: FieldServerID, Value: serverID, Inline: true},
			{Name: "Reason", Value: reason, Inline: false},
		},
	}

	n.send(embed)
}

func (n *Notifier) NotifyReconnecting(serverID string, attempt int, delay time.Duration) {
	if n == nil {
		return
	}

	embed := Embed{
		Title:       "🟡 Reconnecting",
		Description: fmt.Sprintf("Attempting to reconnect (attempt #%d)", attempt),
		Color:       ColorYellow,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []Field{
			{Name: FieldServerID, Value: serverID, Inline: true},
			{Name: "Retry In", Value: delay.Round(time.Second).String(), Inline: true},
		},
	}

	n.send(embed)
}

func (n *Notifier) NotifyUp(serverID, guildID, channelID string) {
	if n == nil {
		return
	}

	embed := Embed{
		Title:       "🟢 Connection Restored",
		Description: fmt.Sprintf("Connection to <#%s> has been successfully restored.", channelID),
		Color:       ColorGreen,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []Field{
			{Name: FieldServerID, Value: serverID, Inline: true},
		},
	}

	n.send(embed)
}

func (n *Notifier) send(embed Embed) {
	payload := WebhookPayload{
		Username:  WebhookUsername,
		AvatarURL: WebhookAvatarURL,
		Embeds:    []Embed{embed},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		n.logger.Error("Failed to marshal webhook payload", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(data))
	if err != nil {
		n.logger.Error("Failed to create webhook request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		n.logger.Error("Failed to send webhook", "error", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		n.logger.Error("Webhook returned error", "status", resp.StatusCode)
		return
	}

	n.logger.Debug("Webhook sent successfully")
}
