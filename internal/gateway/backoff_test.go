package gateway

import (
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name         string
		attempt      int
		wantMinDelay time.Duration
		wantMaxDelay time.Duration
	}{
		{
			name:         "first attempt (0) should be around 1s",
			attempt:      0,
			wantMinDelay: 1 * time.Second,
			wantMaxDelay: 1500 * time.Millisecond,
		},
		{
			name:         "second attempt (1) should be around 2s",
			attempt:      1,
			wantMinDelay: 2 * time.Second,
			wantMaxDelay: 3 * time.Second,
		},
		{
			name:         "third attempt (2) should be around 4s",
			attempt:      2,
			wantMinDelay: 4 * time.Second,
			wantMaxDelay: 6 * time.Second,
		},
		{
			name:         "seventh attempt (6) should be capped at 60s",
			attempt:      6,
			wantMinDelay: 60 * time.Second,
			wantMaxDelay: 90 * time.Second,
		},
		{
			name:         "large attempt should still be capped at 60s",
			attempt:      100,
			wantMinDelay: 60 * time.Second,
			wantMaxDelay: 90 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 10 {
				got := CalculateBackoff(tt.attempt)
				if got < tt.wantMinDelay {
					t.Errorf("CalculateBackoff(%d) = %v, want >= %v", tt.attempt, got, tt.wantMinDelay)
				}
				if got > tt.wantMaxDelay {
					t.Errorf("CalculateBackoff(%d) = %v, want <= %v", tt.attempt, got, tt.wantMaxDelay)
				}
			}
		})
	}
}

func TestCalculateBackoffJitterVariability(t *testing.T) {
	results := make(map[time.Duration]bool)
	for range 100 {
		delay := CalculateBackoff(2)
		results[delay] = true
	}
	if len(results) < 5 {
		t.Errorf("Expected jitter to produce at least 5 unique values, got %d", len(results))
	}
}

func TestResetBackoff(t *testing.T) {
	result := ResetBackoff()
	if result != 0 {
		t.Errorf("ResetBackoff did not return 0, got %d", result)
	}
}

func TestIsFatalCloseCode(t *testing.T) {
	tests := []struct {
		code      int
		wantFatal bool
	}{
		{CloseUnknownError, false},
		{CloseUnknownOpcode, false},
		{CloseDecodeError, false},
		{CloseNotAuthenticated, false},
		{CloseAuthenticationFailed, true},
		{CloseAlreadyAuthenticated, false},
		{CloseInvalidSeq, false},
		{CloseRateLimited, false},
		{CloseSessionTimedOut, false},
		{CloseInvalidShard, true},
		{CloseShardingRequired, true},
		{CloseInvalidAPIVersion, true},
		{CloseInvalidIntents, true},
		{CloseDisallowedIntents, true},
		{1000, false},
		{0, false},
	}

	for _, tt := range tests {
		got := IsFatalCloseCode(tt.code)
		if got != tt.wantFatal {
			t.Errorf("IsFatalCloseCode(%d) = %v, want %v", tt.code, got, tt.wantFatal)
		}
	}
}
