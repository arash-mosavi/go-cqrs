package application

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerMiddleware implements circuit breaker pattern
type CircuitBreakerMiddleware struct {
	failureThreshold int
	resetTimeout     time.Duration
	state            string // "closed", "open", "half-open"
	failures         int
	lastFailure      time.Time
	mutex            sync.RWMutex
}

func NewCircuitBreakerMiddleware(failureThreshold int, resetTimeout time.Duration) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            "closed",
	}
}

func (cb *CircuitBreakerMiddleware) ForCommands() CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
			return cb.execute(func() (interface{}, error) {
				return next.Handle(ctx, cmd)
			})
		})
	}
}

func (cb *CircuitBreakerMiddleware) execute(operation func() (interface{}, error)) (interface{}, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case "open":
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = "half-open"
		} else {
			return nil, fmt.Errorf("circuit breaker is open")
		}
	}

	result, err := operation()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		if cb.failures >= cb.failureThreshold {
			cb.state = "open"
		}
		return nil, err
	}

	// Success - reset failure count and close circuit if it was half-open
	cb.failures = 0
	cb.state = "closed"

	return result, nil
}

func (cb *CircuitBreakerMiddleware) GetState() string {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// RateLimitingMiddleware implements simple rate limiting
type RateLimitingMiddleware struct {
	maxRequests int
	window      time.Duration
	requests    map[string][]time.Time
	mutex       sync.RWMutex
}

func NewRateLimitingMiddleware(maxRequests int, window time.Duration) *RateLimitingMiddleware {
	return &RateLimitingMiddleware{
		maxRequests: maxRequests,
		window:      window,
		requests:    make(map[string][]time.Time),
	}
}

func (rl *RateLimitingMiddleware) ForCommands() CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
			key := fmt.Sprintf("cmd:%s", cmd.CommandName())
			if !rl.allowRequest(key) {
				return nil, fmt.Errorf("rate limit exceeded for command: %s", cmd.CommandName())
			}
			return next.Handle(ctx, cmd)
		})
	}
}

func (rl *RateLimitingMiddleware) allowRequest(key string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Initialize if not exists
	if _, exists := rl.requests[key]; !exists {
		rl.requests[key] = make([]time.Time, 0, rl.maxRequests)
	}

	// Clean old requests outside the window
	requests := rl.requests[key]
	validRequests := make([]time.Time, 0, len(requests))

	for _, reqTime := range requests {
		if now.Sub(reqTime) <= rl.window {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if we can allow this request
	if len(validRequests) >= rl.maxRequests {
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests

	return true
}
