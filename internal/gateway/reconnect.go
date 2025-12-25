package gateway

import (
	"context"
	"log/slog"
	"time"
)

// Reconnector manages automatic reconnection with exponential backoff.
type Reconnector struct {
	client *Client
	token  string
	logger *slog.Logger

	attempt    int
	maxAttempt int
	stopChan   chan struct{}
	stopped    bool
}

// NewReconnector creates a new reconnector for the given client.
func NewReconnector(client *Client, token string, logger *slog.Logger) *Reconnector {
	if logger == nil {
		logger = slog.Default()
	}
	return &Reconnector{
		client:     client,
		token:      token,
		logger:     logger.With("component", "reconnector"),
		maxAttempt: 10, // Stop after 10 attempts
		stopChan:   make(chan struct{}),
	}
}

// Start begins the reconnection process.
// It will attempt to reconnect with exponential backoff until stopped or max attempts reached.
func (r *Reconnector) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		default:
		}

		// Check if we should give up
		if r.attempt >= r.maxAttempt {
			r.logger.Error("Max reconnection attempts reached", "attempts", r.attempt)
			return
		}

		// Calculate backoff delay
		delay := CalculateBackoff(r.attempt)
		r.logger.Info("Waiting before reconnect attempt",
			"attempt", r.attempt+1,
			"delay", delay.String())

		// Wait for backoff period
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-time.After(delay):
		}

		// Attempt to reconnect
		r.logger.Info("Attempting to reconnect", "attempt", r.attempt+1)

		if err := r.client.Connect(ctx); err != nil {
			r.logger.Error("Reconnection failed", "error", err, "attempt", r.attempt+1)
			r.attempt++
			continue
		}

		// Successfully connected - reset attempt counter
		r.logger.Info("Reconnection successful")
		r.attempt = 0
		return
	}
}

// Stop halts the reconnection process.
func (r *Reconnector) Stop() {
	if !r.stopped {
		r.stopped = true
		close(r.stopChan)
	}
}

// ResetAttempts resets the attempt counter (call on successful connection).
func (r *Reconnector) ResetAttempts() {
	r.attempt = 0
}

// Attempt returns the current attempt count.
func (r *Reconnector) Attempt() int {
	return r.attempt
}
