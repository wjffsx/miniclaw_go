package llm

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu           sync.Mutex
	requests     []time.Time
	maxRequests  int
	timeWindow   time.Duration
}

func NewRateLimiter(maxRequests int, timeWindow time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:   make([]time.Time, 0, maxRequests),
		maxRequests: maxRequests,
		timeWindow: timeWindow,
	}
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	cutoff := now.Add(-r.timeWindow)

	validRequests := make([]time.Time, 0, r.maxRequests)
	for _, req := range r.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}

	r.requests = validRequests

	if len(r.requests) >= r.maxRequests {
		return false
	}

	r.requests = append(r.requests, now)
	return true
}

func (r *RateLimiter) Wait() {
	for !r.Allow() {
		time.Sleep(100 * time.Millisecond)
	}
}

func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = make([]time.Time, 0, r.maxRequests)
}