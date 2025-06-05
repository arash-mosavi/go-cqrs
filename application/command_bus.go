package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Command represents a strict interface for all commands.
type Command interface {
	CommandName() string
}

// CommandHandler defines a contract for executing commands.
type CommandHandler interface {
	Handle(ctx context.Context, command Command) (interface{}, error)
}

// CommandBus manages registration and execution of commands with improved algorithms and O(1) operations.
type CommandBus struct {
	handlers   map[string]CommandHandler
	middleware []CommandMiddleware
	mutex      sync.RWMutex
	metrics    *BusMetrics
}

// CommandMiddleware is a function that wraps command handlers
type CommandMiddleware func(CommandHandler) CommandHandler

// BusMetrics tracks performance and usage statistics
type BusMetrics struct {
	TotalExecutions      int64
	TotalFailures        int64
	AverageExecutionTime time.Duration
	mutex                sync.RWMutex
}

// CommandBusOptions defines configuration options for the command bus
type CommandBusOptions struct {
	InitialCapacity int
	EnableMetrics   bool
	Middleware      []CommandMiddleware
}

// NewCommandBus creates a new command bus with improved performance characteristics.
func NewCommandBus(options ...CommandBusOptions) *CommandBus {
	opts := CommandBusOptions{
		InitialCapacity: 32, // Default capacity for better performance
		EnableMetrics:   false,
	}

	if len(options) > 0 {
		opts = options[0]
	}

	cb := &CommandBus{
		handlers:   make(map[string]CommandHandler, opts.InitialCapacity),
		middleware: opts.Middleware,
	}

	if opts.EnableMetrics {
		cb.metrics = &BusMetrics{}
	}

	return cb
}

// RegisterHandler binds a command name to a handler with O(1) complexity.
// Returns error instead of panicking for better production robustness.
func (cb *CommandBus) RegisterHandler(commandName string, handler CommandHandler) error {
	if handler == nil {
		return errors.New("handler cannot be nil")
	}

	if commandName == "" {
		return errors.New("command name cannot be empty")
	}

	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if _, exists := cb.handlers[commandName]; exists {
		return fmt.Errorf("command handler for %s already registered", commandName)
	}

	finalHandler := handler
	for i := len(cb.middleware) - 1; i >= 0; i-- {
		finalHandler = cb.middleware[i](finalHandler)
	}

	cb.handlers[commandName] = finalHandler
	return nil
}

// MustRegisterHandler is a convenience method that panics on registration error.
func (cb *CommandBus) MustRegisterHandler(commandName string, handler CommandHandler) {
	if err := cb.RegisterHandler(commandName, handler); err != nil {
		panic(err)
	}
}

// ExecuteCommand dispatches a command to its corresponding handler with O(1) lookup complexity.
func (cb *CommandBus) ExecuteCommand(ctx context.Context, command Command) (interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if command == nil {
		return nil, errors.New("command cannot be nil")
	}

	start := time.Now()

	if deadline, ok := ctx.Deadline(); ok && time.Now().After(deadline) {
		return nil, errors.New("command execution timeout before handler lookup")
	}

	cb.mutex.RLock()
	handler, exists := cb.handlers[command.CommandName()]
	cb.mutex.RUnlock()

	if !exists {
		cb.recordMetrics(start, true)
		return nil, fmt.Errorf("no handler registered for command: %s", command.CommandName())
	}

	// Execute handler
	result, err := handler.Handle(ctx, command)

	// Record metrics if enabled
	cb.recordMetrics(start, err != nil)

	return result, err
}

// GetRegisteredCommands returns a list of all registered command names.
func (cb *CommandBus) GetRegisteredCommands() []string {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	commands := make([]string, 0, len(cb.handlers))
	for name := range cb.handlers {
		commands = append(commands, name)
	}
	return commands
}

// GetMetrics returns current bus metrics (if enabled)
func (cb *CommandBus) GetMetrics() *BusMetrics {
	if cb.metrics == nil {
		return nil
	}

	cb.metrics.mutex.RLock()
	defer cb.metrics.mutex.RUnlock()

	return &BusMetrics{
		TotalExecutions:      cb.metrics.TotalExecutions,
		TotalFailures:        cb.metrics.TotalFailures,
		AverageExecutionTime: cb.metrics.AverageExecutionTime,
	}
}

func (cb *CommandBus) recordMetrics(start time.Time, failed bool) {
	if cb.metrics == nil {
		return
	}

	duration := time.Since(start)

	cb.metrics.mutex.Lock()
	defer cb.metrics.mutex.Unlock()

	cb.metrics.TotalExecutions++
	if failed {
		cb.metrics.TotalFailures++
	}

	if cb.metrics.TotalExecutions == 1 {
		cb.metrics.AverageExecutionTime = duration
	} else {
		alpha := 0.1
		cb.metrics.AverageExecutionTime = time.Duration(
			float64(cb.metrics.AverageExecutionTime)*(1-alpha) +
				float64(duration)*alpha,
		)
	}
}

// CommandHandlerFunc is an adapter to allow use of ordinary functions as CommandHandler
type CommandHandlerFunc func(ctx context.Context, command Command) (interface{}, error)

// Handle calls the function itself
func (f CommandHandlerFunc) Handle(ctx context.Context, command Command) (interface{}, error) {
	return f(ctx, command)
}

// CommandLoggingMiddleware returns middleware that logs command execution
func CommandLoggingMiddleware(prefix string) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, command Command) (interface{}, error) {
			start := time.Now()
			fmt.Printf("[%s] Executing: %s\n", prefix, command.CommandName())

			result, err := next.Handle(ctx, command)
			duration := time.Since(start)

			if err != nil {
				fmt.Printf("[%s] %s failed after %v: %v\n", prefix, command.CommandName(), duration, err)
			} else {
				fmt.Printf("[%s] %s completed successfully in %v\n", prefix, command.CommandName(), duration)
			}

			return result, err
		})
	}
}

// ValidationMiddleware returns middleware that validates commands
func ValidationMiddleware() CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, command Command) (interface{}, error) {
			// Check if command implements a Validator interface
			if validator, ok := command.(interface{ Validate() error }); ok {
				if err := validator.Validate(); err != nil {
					return nil, fmt.Errorf("command validation failed: %w", err)
				}
			}

			return next.Handle(ctx, command)
		})
	}
}

// TimeoutMiddleware returns middleware that enforces command execution timeout
func TimeoutMiddleware(timeout time.Duration) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, command Command) (interface{}, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			resultChan := make(chan struct {
				result interface{}
				err    error
			}, 1)

			go func() {
				result, err := next.Handle(ctx, command)
				resultChan <- struct {
					result interface{}
					err    error
				}{result, err}
			}()

			select {
			case res := <-resultChan:
				return res.result, res.err
			case <-ctx.Done():
				return nil, fmt.Errorf("command %s execution timeout after %v", command.CommandName(), timeout)
			}
		})
	}
}
