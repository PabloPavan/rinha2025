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

func (cb *Breaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return !time.Now().Before(cb.openUntil)
}

func (cb *Breaker) MarkFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	if cb.failures >= cb.maxFails {
		cb.openUntil = time.Now().Add(cb.openTime)
		cb.failures = 0
	}
}

func (cb *Breaker) MarkSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.openUntil = time.Time{}
}
