package gateway

import (
	"math/rand"
	"time"
)

const (
	BaseDelay    = 1 * time.Second
	MaxDelay     = 60 * time.Second
	JitterFactor = 0.5
)

func CalculateBackoff(attempt int) time.Duration {
	if attempt > 6 {
		attempt = 6
	}

	delay := BaseDelay * time.Duration(1<<uint(attempt))
	delay = min(delay, MaxDelay)
	jitter := randomJitter(delay)
	return delay + jitter
}

func randomJitter(d time.Duration) time.Duration {
	return time.Duration(rand.Float64() * JitterFactor * float64(d))
}
