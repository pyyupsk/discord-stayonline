package gateway

import (
	"context"
	"log/slog"
	"time"
)

type Reconnector struct {
	client *Client
	token  string
	logger *slog.Logger

	attempt    int
	maxAttempt int
	stopChan   chan struct{}
	stopped    bool
}

func NewReconnector(client *Client, token string, logger *slog.Logger) *Reconnector {
	if logger == nil {
		logger = slog.Default()
	}
	return &Reconnector{
		client:     client,
		token:      token,
		logger:     logger.With("component", "reconnector"),
		maxAttempt: 10,
		stopChan:   make(chan struct{}),
	}
}

func (r *Reconnector) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		default:
		}

		if r.attempt >= r.maxAttempt {
			r.logger.Error("Max reconnection attempts reached", "attempts", r.attempt)
			return
		}

		delay := CalculateBackoff(r.attempt)
		r.logger.Info("Waiting before reconnect attempt",
			"attempt", r.attempt+1,
			"delay", delay.String())

		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-time.After(delay):
		}

		r.logger.Info("Attempting to reconnect", "attempt", r.attempt+1)

		if err := r.client.Connect(ctx); err != nil {
			r.logger.Error("Reconnection failed", "error", err, "attempt", r.attempt+1)
			r.attempt++
			continue
		}

		r.logger.Info("Reconnection successful")
		r.attempt = 0
		return
	}
}

func (r *Reconnector) Stop() {
	if !r.stopped {
		r.stopped = true
		close(r.stopChan)
	}
}

func (r *Reconnector) ResetAttempts() {
	r.attempt = 0
}

func (r *Reconnector) Attempt() int {
	return r.attempt
}
