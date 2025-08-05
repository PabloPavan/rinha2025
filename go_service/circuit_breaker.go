package main

import (
	"sync"
	"time"
)

type CircuitBreaker struct {
	mu        sync.Mutex
	failures  int
	openUntil time.Time
	maxFails  int
	openTime  time.Duration
}

func NewCircuitBreaker(maxFails int, openTime time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFails: maxFails,
		openTime: openTime,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return !time.Now().Before(cb.openUntil)
}

func (cb *CircuitBreaker) MarkFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	if cb.failures >= cb.maxFails {
		cb.openUntil = time.Now().Add(cb.openTime)
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) MarkSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.openUntil = time.Time{}
}
