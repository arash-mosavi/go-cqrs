package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Query is the base interface for any query
type Query interface {
	QueryName() string
}

// QueryHandler handles a query and returns the result
type QueryHandler interface {
	Handle(ctx context.Context, query Query) (interface{}, error)
}

// QueryMiddleware is a function that wraps query handlers
type QueryMiddleware func(QueryHandler) QueryHandler

// QueryBus routes queries to their respective handlers with improved performance and O(1) operations.
type QueryBus struct {
	handlers   map[string]QueryHandler
	middleware []QueryMiddleware
	mutex      sync.RWMutex
	metrics    *BusMetrics
}

// QueryBusOptions defines configuration options for the query bus
type QueryBusOptions struct {
	InitialCapacity int
	EnableMetrics   bool
	Middleware      []QueryMiddleware
}

// NewQueryBus creates a new QueryBus with improved algorithms and O(1) operations.
func NewQueryBus(options ...QueryBusOptions) *QueryBus {
	opts := QueryBusOptions{
		InitialCapacity: 32, // Default capacity for better performance
		EnableMetrics:   false,
	}

	if len(options) > 0 {
		opts = options[0]
	}

	qb := &QueryBus{
		handlers:   make(map[string]QueryHandler, opts.InitialCapacity),
		middleware: opts.Middleware,
	}

	if opts.EnableMetrics {
		qb.metrics = &BusMetrics{}
	}

	return qb
}

// RegisterHandler associates a query name with a handler with O(1) complexity.
// Returns error instead of panicking for better production robustness.
func (qb *QueryBus) RegisterHandler(queryName string, handler QueryHandler) error {
	if handler == nil {
		return errors.New("handler cannot be nil")
	}

	if queryName == "" {
		return errors.New("query name cannot be empty")
	}

	qb.mutex.Lock()
	defer qb.mutex.Unlock()

	if _, exists := qb.handlers[queryName]; exists {
		return fmt.Errorf("query handler already registered: %s", queryName)
	}

	// Apply middleware chain in reverse order (first added, outermost wrapper)
	finalHandler := handler
	for i := len(qb.middleware) - 1; i >= 0; i-- {
		finalHandler = qb.middleware[i](finalHandler)
	}

	qb.handlers[queryName] = finalHandler
	return nil
}

// MustRegisterHandler is a convenience method that panics on registration error.
func (qb *QueryBus) MustRegisterHandler(queryName string, handler QueryHandler) {
	if err := qb.RegisterHandler(queryName, handler); err != nil {
		panic(err)
	}
}

// Execute dispatches a query to its handler with O(1) lookup complexity.
func (qb *QueryBus) Execute(ctx context.Context, query Query) (interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if query == nil {
		return nil, errors.New("query cannot be nil")
	}

	start := time.Now()

	// Fast path: check for deadline exceeded before handler lookup
	if deadline, ok := ctx.Deadline(); ok && time.Now().After(deadline) {
		return nil, errors.New("query execution timeout before handler lookup")
	}

	// O(1) handler lookup with read lock
	qb.mutex.RLock()
	handler, exists := qb.handlers[query.QueryName()]
	qb.mutex.RUnlock()

	if !exists {
		qb.recordMetrics(start, true)
		return nil, fmt.Errorf("no handler registered for query: %s", query.QueryName())
	}

	// Execute handler
	result, err := handler.Handle(ctx, query)

	// Record metrics if enabled
	qb.recordMetrics(start, err != nil)

	return result, err
}

// ExecuteTyped executes a query and attempts to cast the result to type T.
// This provides type safety and eliminates the need for manual type assertions.
func ExecuteTyped[T any](ctx context.Context, qb *QueryBus, query Query) (T, error) {
	var zero T

	result, err := qb.Execute(ctx, query)
	if err != nil {
		return zero, err
	}

	if result == nil {
		return zero, nil
	}

	typed, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("cannot cast query result to %T, got %T", zero, result)
	}

	return typed, nil
}

// GetRegisteredQueries returns a list of all registered query names.
// Time complexity: O(n) where n is the number of registered queries.
func (qb *QueryBus) GetRegisteredQueries() []string {
	qb.mutex.RLock()
	defer qb.mutex.RUnlock()

	queries := make([]string, 0, len(qb.handlers))
	for name := range qb.handlers {
		queries = append(queries, name)
	}
	return queries
}

// GetMetrics returns current bus metrics (if enabled)
func (qb *QueryBus) GetMetrics() *BusMetrics {
	if qb.metrics == nil {
		return nil
	}

	qb.metrics.mutex.RLock()
	defer qb.metrics.mutex.RUnlock()

	// Return a copy to prevent external modification
	return &BusMetrics{
		TotalExecutions:      qb.metrics.TotalExecutions,
		TotalFailures:        qb.metrics.TotalFailures,
		AverageExecutionTime: qb.metrics.AverageExecutionTime,
	}
}

// recordMetrics updates performance metrics (internal method)
func (qb *QueryBus) recordMetrics(start time.Time, failed bool) {
	if qb.metrics == nil {
		return
	}

	duration := time.Since(start)

	qb.metrics.mutex.Lock()
	defer qb.metrics.mutex.Unlock()

	qb.metrics.TotalExecutions++
	if failed {
		qb.metrics.TotalFailures++
	}

	// Update rolling average (simple implementation)
	if qb.metrics.TotalExecutions == 1 {
		qb.metrics.AverageExecutionTime = duration
	} else {
		// Exponential moving average with alpha = 0.1
		alpha := 0.1
		qb.metrics.AverageExecutionTime = time.Duration(
			float64(qb.metrics.AverageExecutionTime)*(1-alpha) +
				float64(duration)*alpha,
		)
	}
}

// QueryHandlerFunc is an adapter to allow use of ordinary functions as QueryHandler
type QueryHandlerFunc func(ctx context.Context, query Query) (interface{}, error)

// Handle calls the function itself
func (f QueryHandlerFunc) Handle(ctx context.Context, query Query) (interface{}, error) {
	return f(ctx, query)
}

// CachingMiddleware returns middleware that caches query results
// Note: This is a simple in-memory cache. For production, consider using Redis or similar.
func CachingMiddleware(cacheTTL time.Duration) QueryMiddleware {
	cache := &sync.Map{}

	return func(next QueryHandler) QueryHandler {
		return QueryHandlerFunc(func(ctx context.Context, query Query) (interface{}, error) {
			cacheKey := fmt.Sprintf("%s:%s", query.QueryName(), getCacheKey(query))

			// Check cache first
			if cached, ok := cache.Load(cacheKey); ok {
				if cacheEntry, ok := cached.(*cacheEntry); ok && time.Now().Before(cacheEntry.expiresAt) {
					return cacheEntry.value, cacheEntry.err
				}
				// Remove expired entry
				cache.Delete(cacheKey)
			}

			// Execute query
			result, err := next.Handle(ctx, query)

			// Cache the result
			cache.Store(cacheKey, &cacheEntry{
				value:     result,
				err:       err,
				expiresAt: time.Now().Add(cacheTTL),
			})

			return result, err
		})
	}
}

// cacheEntry represents a cached query result
type cacheEntry struct {
	value     interface{}
	err       error
	expiresAt time.Time
}

// getCacheKey generates a cache key for a query (simple implementation)
func getCacheKey(query Query) string {
	// In a real implementation, you might want to serialize the query parameters
	// For now, we just use the query name
	return query.QueryName()
}

// RetryMiddleware returns middleware that retries failed queries
func RetryMiddleware(maxRetries int, retryDelay time.Duration) QueryMiddleware {
	return func(next QueryHandler) QueryHandler {
		return QueryHandlerFunc(func(ctx context.Context, query Query) (interface{}, error) {
			var lastErr error

			for attempt := 0; attempt <= maxRetries; attempt++ {
				result, err := next.Handle(ctx, query)
				if err == nil {
					return result, nil
				}

				lastErr = err

				// Don't retry on the last attempt
				if attempt < maxRetries {
					select {
					case <-time.After(retryDelay):
						// Continue to next attempt
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
			}

			return nil, fmt.Errorf("query %s failed after %d attempts: %w", query.QueryName(), maxRetries+1, lastErr)
		})
	}
}

// QueryLoggingMiddleware returns middleware that logs query execution
func QueryLoggingMiddleware(prefix string) QueryMiddleware {
	return func(next QueryHandler) QueryHandler {
		return QueryHandlerFunc(func(ctx context.Context, query Query) (interface{}, error) {
			start := time.Now()
			fmt.Printf("[%s] Executing: %s\n", prefix, query.QueryName())

			result, err := next.Handle(ctx, query)
			duration := time.Since(start)

			if err != nil {
				fmt.Printf("[%s] %s failed after %v: %v\n", prefix, query.QueryName(), duration, err)
			} else {
				fmt.Printf("[%s] %s completed successfully in %v\n", prefix, query.QueryName(), duration)
			}

			return result, err
		})
	}
}
