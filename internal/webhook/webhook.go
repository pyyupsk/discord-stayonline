// Package webhook provides Discord webhook notifications for server status changes.
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

// Notifier sends Discord webhook notifications.
type Notifier struct {
	webhookURL string
	client     *http.Client
	logger     *slog.Logger
}

// Embed represents a Discord embed object.
type Embed struct {
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Color       int     `json:"color,omitempty"`
	Timestamp   string  `json:"timestamp,omitempty"`
	Fields      []Field `json:"fields,omitempty"`
}

// Field represents a Discord embed field.
type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// WebhookPayload represents a Discord webhook message.
type WebhookPayload struct {
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Content   string  `json:"content,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

// Colors for different notification types.
const (
	ColorRed    = 0xFF0000 // Error/Down
	ColorGreen  = 0x00FF00 // Connected/Up
	ColorYellow = 0xFFFF00 // Warning/Reconnecting
)

// Webhook identity.
const (
	WebhookUsername  = "Stay Online Monitor"
	WebhookAvatarURL = "https://raw.githubusercontent.com/pyyupsk/discord-stayonline/main/web/public/android-chrome-512x512.png"
)

// Field names.
const (
	FieldServerID = "Server ID"
	FieldGuild    = "Guild"
	FieldChannel  = "Channel"
)

// NewNotifier creates a new webhook notifier.
// Returns nil if webhookURL is empty.
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

// NotifyDown sends a notification when a server connection goes down.
func (n *Notifier) NotifyDown(serverID, guildName, channelName, reason string) {
	if n == nil {
		return
	}

	embed := Embed{
		Title:       "🔴 Connection Lost",
		Description: "Server connection has been lost and will attempt to reconnect.",
		Color:       ColorRed,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []Field{
			{Name: FieldServerID, Value: serverID, Inline: true},
			{Name: FieldGuild, Value: guildName, Inline: true},
			{Name: FieldChannel, Value: channelName, Inline: true},
			{Name: "Reason", Value: reason, Inline: false},
		},
	}

	n.send(embed)
}

// NotifyReconnecting sends a notification when reconnecting.
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

// NotifyUp sends a notification when connection is restored.
func (n *Notifier) NotifyUp(serverID, guildName, channelName string) {
	if n == nil {
		return
	}

	embed := Embed{
		Title:       "🟢 Connection Restored",
		Description: "Server connection has been successfully restored.",
		Color:       ColorGreen,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []Field{
			{Name: FieldServerID, Value: serverID, Inline: true},
			{Name: FieldGuild, Value: guildName, Inline: true},
			{Name: FieldChannel, Value: channelName, Inline: true},
		},
	}

	n.send(embed)
}

// send sends the webhook payload to Discord.
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
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		n.logger.Error("Webhook returned error", "status", resp.StatusCode)
		return
	}

	n.logger.Debug("Webhook sent successfully")
}
