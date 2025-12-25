package gateway

import (
	"context"
	"testing"
	"time"
)

func TestNewReconnector(t *testing.T) {
	client := NewClient("test-token", nil)
	reconnector := NewReconnector(client, "test-token", nil)

	if reconnector == nil {
		t.Fatal("NewReconnector returned nil")
	}
	if reconnector.client != client {
		t.Error("client not set correctly")
	}
	if reconnector.token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", reconnector.token)
	}
	if reconnector.maxAttempt != 10 {
		t.Errorf("expected maxAttempt 10, got %d", reconnector.maxAttempt)
	}
}

func TestReconnectorAttempt(t *testing.T) {
	client := NewClient("test-token", nil)
	reconnector := NewReconnector(client, "test-token", nil)

	if reconnector.Attempt() != 0 {
		t.Errorf("expected initial attempt 0, got %d", reconnector.Attempt())
	}
}

func TestReconnectorResetAttempts(t *testing.T) {
	client := NewClient("test-token", nil)
	reconnector := NewReconnector(client, "test-token", nil)

	reconnector.attempt = 5
	reconnector.ResetAttempts()

	if reconnector.Attempt() != 0 {
		t.Errorf("expected attempt after reset to be 0, got %d", reconnector.Attempt())
	}
}

func TestReconnectorStop(t *testing.T) {
	client := NewClient("test-token", nil)
	reconnector := NewReconnector(client, "test-token", nil)

	// Stop should work without panicking
	reconnector.Stop()

	if !reconnector.stopped {
		t.Error("expected stopped to be true after Stop()")
	}

	// Double stop should not panic
	reconnector.Stop()
}

func TestReconnectorStartWithContextCancel(t *testing.T) {
	client := NewClient("test-token", nil)
	reconnector := NewReconnector(client, "test-token", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		reconnector.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success - Start returned when context was cancelled
	case <-time.After(2 * time.Second):
		t.Error("Start did not return when context was cancelled")
	}
}

func TestReconnectorStartWithStop(t *testing.T) {
	client := NewClient("test-token", nil)
	reconnector := NewReconnector(client, "test-token", nil)

	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		reconnector.Start(ctx)
		close(done)
	}()

	// Give it a moment to start the loop
	time.Sleep(50 * time.Millisecond)

	// Stop the reconnector
	reconnector.Stop()

	select {
	case <-done:
		// Success - Start returned when stopped
	case <-time.After(2 * time.Second):
		t.Error("Start did not return when stopped")
	}
}

func TestReconnectorMaxAttempts(t *testing.T) {
	client := NewClient("test-token", nil)
	reconnector := NewReconnector(client, "test-token", nil)
	reconnector.maxAttempt = 0 // Set to 0 so it exits immediately

	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		reconnector.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success - Start returned when max attempts reached
	case <-time.After(2 * time.Second):
		t.Error("Start did not return when max attempts reached")
	}
}
