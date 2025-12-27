package gateway

import (
	"crypto/rand"
	"encoding/binary"
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

func randomJitter(delay time.Duration) time.Duration {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}

	randUint := binary.BigEndian.Uint64(buf[:])
	randFloat := float64(randUint) / float64(^uint64(0))
	jitterNanos := randFloat * JitterFactor * float64(delay.Nanoseconds())
	return time.Duration(jitterNanos)
}

func ResetBackoff() int {
	return 0
}
