package circuitbreaker

import (
	"sync"
	"time"
)

type Breaker struct {
	mu        sync.Mutex
	failures  int
	openUntil time.Time
	maxFails  int
	openTime  time.Duration
}

func NewCircuitBreaker(maxFails int, openTime time.Duration) *Breaker {
	return &Breaker{
		maxFails: maxFails,
		openTime: openTime,
	}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return !time.Now().Before(b.openUntil)
}

func (b *Breaker) MarkFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures++
	if b.failures >= b.maxFails {
		b.openUntil = time.Now().Add(b.openTime)
		b.failures = 0
	}
}

func (b *Breaker) MarkSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures = 0
	b.openUntil = time.Time{}
}

func (b *Breaker) RemainingOpen() time.Duration {
	b.mu.Lock()
	until := b.openUntil
	b.mu.Unlock()

	if until.IsZero() {
		return 0
	}
	d := time.Until(until)
	if d < 0 {
		return 0
	}
	return d
}
