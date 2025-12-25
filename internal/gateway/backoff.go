package gateway

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

const (
	// BaseDelay is the initial backoff delay.
	BaseDelay = 1 * time.Second

	// MaxDelay is the maximum backoff delay (capped at 60 seconds per spec).
	MaxDelay = 60 * time.Second

	// JitterFactor is the maximum jitter percentage (0.5 = 50%).
	JitterFactor = 0.5
)

// CalculateBackoff computes the backoff delay for a given attempt.
// It uses exponential backoff with jitter:
// - Base delay: 1 second
// - Doubles each attempt: 1s, 2s, 4s, 8s, 16s, 32s, 60s (capped)
// - Adds 0-50% random jitter to prevent thundering herd
//
// The attempt parameter is 0-indexed (first retry is attempt 0).
func CalculateBackoff(attempt int) time.Duration {
	// Prevent overflow: cap the shift at a reasonable value
	// 2^6 = 64 seconds, which is already above MaxDelay (60s)
	if attempt > 6 {
		attempt = 6
	}

	// Calculate exponential delay: base * 2^attempt
	delay := BaseDelay * time.Duration(1<<uint(attempt))

	// Cap at maximum delay
	if delay > MaxDelay {
		delay = MaxDelay
	}

	// Add jitter: 0-50% of the delay
	jitter := randomJitter(delay)
	return delay + jitter
}

// randomJitter generates a random duration between 0 and JitterFactor * delay.
// Uses crypto/rand for randomness.
func randomJitter(delay time.Duration) time.Duration {
	// Read random bytes
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// If crypto/rand fails, return no jitter rather than panic
		return 0
	}

	// Convert to float64 in range [0, 1)
	randUint := binary.BigEndian.Uint64(buf[:])
	randFloat := float64(randUint) / float64(^uint64(0))

	// Calculate jitter: 0 to JitterFactor * delay
	jitterNanos := randFloat * JitterFactor * float64(delay.Nanoseconds())
	return time.Duration(jitterNanos)
}

// ResetBackoff returns the value to reset backoff attempt counter to.
func ResetBackoff() int {
	return 0
}
