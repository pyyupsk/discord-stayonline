package tests

import (
	"testing"
	"time"

	"github.com/pyyupsk/discord-stayonline/internal/gateway"
)

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name          string
		attempt       int
		wantBaseDelay time.Duration
		wantMaxDelay  time.Duration
	}{
		{
			name:          "first attempt (0) should be around 1s",
			attempt:       0,
			wantBaseDelay: 1 * time.Second,
			wantMaxDelay:  1500 * time.Millisecond, // 1s + 50% jitter max
		},
		{
			name:          "second attempt (1) should be around 2s",
			attempt:       1,
			wantBaseDelay: 2 * time.Second,
			wantMaxDelay:  3 * time.Second, // 2s + 50% jitter max
		},
		{
			name:          "third attempt (2) should be around 4s",
			attempt:       2,
			wantBaseDelay: 4 * time.Second,
			wantMaxDelay:  6 * time.Second, // 4s + 50% jitter max
		},
		{
			name:          "fourth attempt (3) should be around 8s",
			attempt:       3,
			wantBaseDelay: 8 * time.Second,
			wantMaxDelay:  12 * time.Second, // 8s + 50% jitter max
		},
		{
			name:          "fifth attempt (4) should be around 16s",
			attempt:       4,
			wantBaseDelay: 16 * time.Second,
			wantMaxDelay:  24 * time.Second, // 16s + 50% jitter max
		},
		{
			name:          "sixth attempt (5) should be around 32s",
			attempt:       5,
			wantBaseDelay: 32 * time.Second,
			wantMaxDelay:  48 * time.Second, // 32s + 50% jitter max
		},
		{
			name:          "seventh attempt (6) should be capped at 60s",
			attempt:       6,
			wantBaseDelay: 60 * time.Second,
			wantMaxDelay:  90 * time.Second, // 60s + 50% jitter max
		},
		{
			name:          "large attempt should still be capped at 60s",
			attempt:       100,
			wantBaseDelay: 60 * time.Second,
			wantMaxDelay:  90 * time.Second, // 60s + 50% jitter max
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to account for jitter
			for range 10 {
				got := gateway.CalculateBackoff(tt.attempt)

				if got < tt.wantBaseDelay {
					t.Errorf("CalculateBackoff(%d) = %v, want >= %v", tt.attempt, got, tt.wantBaseDelay)
				}
				if got > tt.wantMaxDelay {
					t.Errorf("CalculateBackoff(%d) = %v, want <= %v", tt.attempt, got, tt.wantMaxDelay)
				}
			}
		})
	}
}

func TestCalculateBackoffExponentialDoubling(t *testing.T) {
	// Verify the exponential doubling pattern without jitter
	// We test that the minimum delay doubles each attempt
	prevMin := gateway.BaseDelay

	for attempt := 1; attempt <= 5; attempt++ {
		// Get the theoretical minimum (base * 2^attempt, capped at max)
		expectedMin := min(gateway.BaseDelay*time.Duration(1<<uint(attempt)), gateway.MaxDelay)

		// The actual delay should be at least the expected minimum
		for range 10 {
			got := gateway.CalculateBackoff(attempt)
			if got < expectedMin {
				t.Errorf("attempt %d: got %v, expected >= %v", attempt, got, expectedMin)
			}
		}

		// Verify it's roughly doubling (allowing for cap)
		if expectedMin < gateway.MaxDelay {
			if expectedMin < prevMin*2 {
				t.Errorf("attempt %d: expected min %v is not ~2x previous min %v", attempt, expectedMin, prevMin)
			}
		}
		prevMin = expectedMin
	}
}

func TestCalculateBackoffJitterRange(t *testing.T) {
	// Test that jitter is actually adding variability
	results := make(map[time.Duration]bool)

	for range 100 {
		delay := gateway.CalculateBackoff(2) // 4s base
		results[delay] = true
	}

	// With 50% jitter on 4s base, we should get values between 4s and 6s
	// With 100 samples, we should have at least 5 unique values if jitter is working
	if len(results) < 5 {
		t.Errorf("Expected jitter to produce at least 5 unique values, got %d", len(results))
	}
}

func TestCalculateBackoffCapAt60Seconds(t *testing.T) {
	// Verify the cap is enforced at 60 seconds
	for attempt := 6; attempt <= 20; attempt++ {
		for range 10 {
			got := gateway.CalculateBackoff(attempt)
			maxWithJitter := gateway.MaxDelay + time.Duration(float64(gateway.MaxDelay)*gateway.JitterFactor)
			if got > maxWithJitter {
				t.Errorf("attempt %d: got %v, want <= %v (cap + jitter)", attempt, got, maxWithJitter)
			}
			if got < gateway.MaxDelay {
				t.Errorf("attempt %d: got %v, want >= %v (cap)", attempt, got, gateway.MaxDelay)
			}
		}
	}
}

func TestIsFatalCloseCode(t *testing.T) {
	tests := []struct {
		code      int
		wantFatal bool
	}{
		{gateway.CloseUnknownError, false},
		{gateway.CloseUnknownOpcode, false},
		{gateway.CloseDecodeError, false},
		{gateway.CloseNotAuthenticated, false},
		{gateway.CloseAuthenticationFailed, true}, // FATAL
		{gateway.CloseAlreadyAuthenticated, false},
		{gateway.CloseInvalidSeq, false},
		{gateway.CloseRateLimited, false},
		{gateway.CloseSessionTimedOut, false},
		{gateway.CloseInvalidShard, true},      // FATAL
		{gateway.CloseShardingRequired, true},  // FATAL
		{gateway.CloseInvalidAPIVersion, true}, // FATAL
		{gateway.CloseInvalidIntents, true},    // FATAL
		{gateway.CloseDisallowedIntents, true}, // FATAL
		{1000, false},                          // Normal closure
		{1001, false},                          // Going away
		{0, false},                             // Unknown
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := gateway.IsFatalCloseCode(tt.code)
			if got != tt.wantFatal {
				t.Errorf("IsFatalCloseCode(%d) = %v, want %v", tt.code, got, tt.wantFatal)
			}
		})
	}
}
