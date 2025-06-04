# 📚 CQRS API Reference

## Table of Contents

- [Core Components](#core-components)
- [Command Bus API](#command-bus-api)
- [Query Bus API](#query-bus-api)
- [Middleware API](#middleware-api)
- [Configuration API](#configuration-api)
- [Monitoring API](#monitoring-api)
- [Types and Interfaces](#types-and-interfaces)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

---

## Core Components

### Command Interface

```go
type Command interface {
    CommandName() string
}
```

**Description**: Interface that all commands must implement to identify themselves.

**Methods**:
- `CommandName() string`: Returns the unique name/identifier for the command

### Query Interface

```go
type Query interface {
    QueryName() string
}
```

**Description**: Interface that all queries must implement to identify themselves.

**Methods**:
- `QueryName() string`: Returns the unique name/identifier for the query

### CommandHandler Interface

```go
type CommandHandler interface {
    Handle(ctx context.Context, cmd Command) (interface{}, error)
}
```

**Description**: Interface for command handlers that process business logic.

**Methods**:
- `Handle(ctx context.Context, cmd Command) (interface{}, error)`: Processes the command and returns result or error

### QueryHandler Interface

```go
type QueryHandler interface {
    Handle(ctx context.Context, query Query) (interface{}, error)
}
```

**Description**: Interface for query handlers that retrieve data.

**Methods**:
- `Handle(ctx context.Context, query Query) (interface{}, error)`: Processes the query and returns result or error

---

## Command Bus API

### NewCommandBus

```go
func NewCommandBus(options CommandBusOptions) *CommandBus
```

**Description**: Creates a new command bus with specified configuration.

**Parameters**:
- `options CommandBusOptions`: Configuration options for the command bus

**Returns**: `*CommandBus` - Configured command bus instance

**Example**:
```go
commandBus := application.NewCommandBus(application.CommandBusOptions{
    InitialCapacity: 256,
    EnableMetrics:   true,
    Middleware: []application.CommandMiddleware{
        application.CommandLoggingMiddleware("COMMANDS"),
        application.ValidationMiddleware(),
        application.TimeoutMiddleware(30 * time.Second),
    },
})
```

### RegisterHandler

```go
func (cb *CommandBus) RegisterHandler(commandName string, handler CommandHandler) error
```

**Description**: Registers a command handler for a specific command type.

**Parameters**:
- `commandName string`: The name of the command to handle
- `handler CommandHandler`: The handler implementation

**Returns**: `error` - Error if registration fails (e.g., duplicate handler)

**Example**:
```go
err := commandBus.RegisterHandler("CreateUser", &CreateUserHandler{})
if err != nil {
    log.Fatalf("Failed to register handler: %v", err)
}
```

### ExecuteCommand

```go
func (cb *CommandBus) ExecuteCommand(ctx context.Context, command Command) (interface{}, error)
```

**Description**: Executes a command through the registered handler with middleware.

**Parameters**:
- `ctx context.Context`: Request context for cancellation and timeouts
- `command Command`: The command to execute

**Returns**: 
- `interface{}`: Result from the command handler
- `error`: Error if execution fails

**Example**:
```go
result, err := commandBus.ExecuteCommand(ctx, CreateUserCommand{
    Name:  "John Doe",
    Email: "john@example.com",
})
if err != nil {
    log.Printf("Command failed: %v", err)
    return
}
```

### GetMetrics

```go
func (cb *CommandBus) GetMetrics() *BusMetrics
```

**Description**: Returns performance metrics for the command bus.

**Returns**: `*BusMetrics` - Metrics object with execution statistics

**Example**:
```go
metrics := commandBus.GetMetrics()
if metrics != nil {
    log.Printf("Total executions: %d", metrics.TotalExecutions)
    log.Printf("Average time: %v", metrics.AverageExecutionTime)
}
```

### GetRegisteredCommands

```go
func (cb *CommandBus) GetRegisteredCommands() []string
```

**Description**: Returns list of all registered command names.

**Returns**: `[]string` - Slice of registered command names

**Example**:
```go
commands := commandBus.GetRegisteredCommands()
fmt.Printf("Registered commands: %v", commands)
```

---

## Query Bus API

### NewQueryBus

```go
func NewQueryBus(options QueryBusOptions) *QueryBus
```

**Description**: Creates a new query bus with specified configuration.

**Parameters**:
- `options QueryBusOptions`: Configuration options for the query bus

**Returns**: `*QueryBus` - Configured query bus instance

**Example**:
```go
queryBus := application.NewQueryBus(application.QueryBusOptions{
    InitialCapacity: 256,
    EnableMetrics:   true,
    Middleware: []application.QueryMiddleware{
        application.CachingMiddleware(10 * time.Minute),
        application.RetryMiddleware(3, 500*time.Millisecond),
        application.QueryLoggingMiddleware("QUERIES"),
    },
})
```

### RegisterHandler

```go
func (qb *QueryBus) RegisterHandler(queryName string, handler QueryHandler) error
```

**Description**: Registers a query handler for a specific query type.

**Parameters**:
- `queryName string`: The name of the query to handle
- `handler QueryHandler`: The handler implementation

**Returns**: `error` - Error if registration fails

**Example**:
```go
err := queryBus.RegisterHandler("GetUser", &GetUserHandler{})
if err != nil {
    log.Fatalf("Failed to register handler: %v", err)
}
```

### Execute

```go
func (qb *QueryBus) Execute(ctx context.Context, query Query) (interface{}, error)
```

**Description**: Executes a query through the registered handler with middleware.

**Parameters**:
- `ctx context.Context`: Request context for cancellation and timeouts
- `query Query`: The query to execute

**Returns**: 
- `interface{}`: Result from the query handler
- `error`: Error if execution fails

**Example**:
```go
result, err := queryBus.Execute(ctx, GetUserQuery{UserID: 123})
if err != nil {
    log.Printf("Query failed: %v", err)
    return
}
user := result.(*User)
```

### ExecuteTyped

```go
func ExecuteTyped[T any](ctx context.Context, bus *QueryBus, query Query) (T, error)
```

**Description**: Type-safe query execution with automatic type assertion.

**Type Parameters**:
- `T`: Expected return type

**Parameters**:
- `ctx context.Context`: Request context
- `bus *QueryBus`: Query bus instance
- `query Query`: Query to execute

**Returns**: 
- `T`: Typed result
- `error`: Error if execution or type assertion fails

**Example**:
```go
user, err := application.ExecuteTyped[*User](ctx, queryBus, GetUserQuery{UserID: 123})
if err != nil {
    log.Printf("Query failed: %v", err)
    return
}
// user is already typed as *User
```

---

## Middleware API

### Command Middleware

#### CommandLoggingMiddleware

```go
func CommandLoggingMiddleware(prefix string) CommandMiddleware
```

**Description**: Logs command execution with optional prefix.

**Parameters**:
- `prefix string`: Log prefix for identification

**Returns**: `CommandMiddleware` - Configured logging middleware

#### ValidationMiddleware

```go
func ValidationMiddleware() CommandMiddleware
```

**Description**: Validates commands that implement the `Validatable` interface.

**Returns**: `CommandMiddleware` - Validation middleware

#### TimeoutMiddleware

```go
func TimeoutMiddleware(timeout time.Duration) CommandMiddleware
```

**Description**: Enforces execution timeout for commands.

**Parameters**:
- `timeout time.Duration`: Maximum execution time

**Returns**: `CommandMiddleware` - Timeout middleware

#### Circuit Breaker Middleware

```go
func NewCircuitBreakerMiddleware(failureThreshold int, timeout time.Duration) *CircuitBreakerMiddleware
func (cb *CircuitBreakerMiddleware) ForCommands() CommandMiddleware
```

**Description**: Circuit breaker pattern implementation for fault tolerance.

**Parameters**:
- `failureThreshold int`: Number of failures before opening circuit
- `timeout time.Duration`: Time to wait before attempting recovery

### Query Middleware

#### CachingMiddleware

```go
func CachingMiddleware(ttl time.Duration) QueryMiddleware
```

**Description**: In-memory caching with configurable TTL.

**Parameters**:
- `ttl time.Duration`: Time-to-live for cached results

**Returns**: `QueryMiddleware` - Caching middleware

#### RetryMiddleware

```go
func RetryMiddleware(maxRetries int, baseDelay time.Duration) QueryMiddleware
```

**Description**: Automatic retry with exponential backoff.

**Parameters**:
- `maxRetries int`: Maximum number of retry attempts
- `baseDelay time.Duration`: Base delay between retries

**Returns**: `QueryMiddleware` - Retry middleware

#### Rate Limiting Middleware

```go
func NewRateLimitingMiddleware(limit int, window time.Duration) *RateLimitingMiddleware
func (rl *RateLimitingMiddleware) ForQueries() QueryMiddleware
```

**Description**: Token bucket rate limiting for resource protection.

**Parameters**:
- `limit int`: Maximum requests per window
- `window time.Duration`: Time window for rate limiting

---

## Configuration API

### LoadConfig

```go
func LoadConfig(configPath string) (*Config, error)
```

**Description**: Loads configuration from JSON file.

**Parameters**:
- `configPath string`: Path to configuration file

**Returns**: 
- `*Config`: Loaded configuration
- `error`: Error if loading fails

### LoadConfigFromEnv

```go
func LoadConfigFromEnv() (*Config, error)
```

**Description**: Loads configuration from environment variables.

**Returns**: 
- `*Config`: Configuration with environment overrides
- `error`: Error if loading fails

### DefaultConfig

```go
func DefaultConfig() *Config
```

**Description**: Returns production-ready default configuration.

**Returns**: `*Config` - Default configuration

### Config Structure

```go
type Config struct {
    CommandBus CommandBusConfig `json:"command_bus"`
    QueryBus   QueryBusConfig   `json:"query_bus"`
    Middleware MiddlewareConfig `json:"middleware"`
    Monitoring MonitoringConfig `json:"monitoring"`
    Performance PerformanceConfig `json:"performance"`
}
```

---

## Monitoring API

### MetricsCollector

```go
func NewMetricsCollector() *MetricsCollector
func (mc *MetricsCollector) CollectMetrics() map[string]interface{}
```

**Description**: Collects system and application metrics.

### PrometheusExporter

```go
func NewPrometheusExporter(collector *MetricsCollector) *PrometheusExporter
func (pe *PrometheusExporter) ExportMetrics() string
```

**Description**: Exports metrics in Prometheus format.

### HealthChecker

```go
func NewHealthChecker() *HealthChecker
func (hc *HealthChecker) AddCheck(name string, check HealthCheck)
func (hc *HealthChecker) CheckHealth() map[string]HealthStatus
```

**Description**: Manages application health checks.

### MonitoringServer

```go
func NewMonitoringServer(config MonitoringConfig, collector *MetricsCollector) *MonitoringServer
func (ms *MonitoringServer) Start() error
func (ms *MonitoringServer) Stop() error
```

**Description**: HTTP server for metrics and health endpoints.

---

## Types and Interfaces

### BusMetrics

```go
type BusMetrics struct {
    TotalExecutions      int64
    TotalFailures        int64
    AverageExecutionTime time.Duration
    mutex                sync.RWMutex
}
```

### CommandBusOptions

```go
type CommandBusOptions struct {
    InitialCapacity int
    EnableMetrics   bool
    Middleware      []CommandMiddleware
}
```

### QueryBusOptions

```go
type QueryBusOptions struct {
    InitialCapacity int
    EnableMetrics   bool
    Middleware      []QueryMiddleware
}
```

### Validatable Interface

```go
type Validatable interface {
    Validate() error
}
```

**Description**: Interface for commands that can be validated.

---

## Error Handling

### Common Errors

- `ErrHandlerNotFound`: Handler not registered for command/query
- `ErrHandlerAlreadyRegistered`: Attempt to register duplicate handler
- `ErrValidationFailed`: Command validation failed
- `ErrTimeout`: Operation exceeded timeout
- `ErrCircuitBreakerOpen`: Circuit breaker is open
- `ErrRateLimitExceeded`: Rate limit exceeded

### Error Types

```go
type CQRSError struct {
    Type    string
    Message string
    Cause   error
}

func (e *CQRSError) Error() string
func (e *CQRSError) Unwrap() error
```

---

## Best Practices

### Command Design

1. **Single Responsibility**: One command should perform one business operation
2. **Immutable**: Commands should be immutable after creation
3. **Validation**: Implement `Validatable` interface for input validation
4. **Naming**: Use imperative verbs (CreateUser, UpdateOrder, DeleteProduct)

### Query Design

1. **Read-Only**: Queries should not modify system state
2. **Cacheable**: Design queries to be cacheable when possible
3. **Specific**: Avoid generic "get everything" queries
4. **Naming**: Use descriptive names (GetUserByEmail, FindActiveOrders)

### Handler Implementation

1. **Error Handling**: Always return meaningful errors
2. **Context Awareness**: Respect context cancellation
3. **Idempotency**: Commands should be idempotent when possible
4. **Logging**: Include relevant context in logs

### Performance Optimization

1. **Pre-allocation**: Set appropriate initial capacities
2. **Middleware Order**: Place expensive middleware last
3. **Caching Strategy**: Cache read-heavy queries
4. **Resource Pooling**: Enable pooling for high-throughput scenarios

### Production Deployment

1. **Configuration**: Use environment-specific configurations
2. **Monitoring**: Enable comprehensive metrics and health checks
3. **Circuit Breakers**: Protect against cascading failures
4. **Rate Limiting**: Implement appropriate rate limits
5. **Graceful Shutdown**: Handle shutdown signals properly

---

## Example Integration

```go
package main

import (
    "context"
    "log"
    "time"
    
    "your-module/application"
)

func main() {
    // Load configuration
    config, err := application.LoadConfig("configs/production.json")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Create buses with configuration
    commandBus := application.NewCommandBusFromConfig(config)
    queryBus := application.NewQueryBusFromConfig(config)
    
    // Register handlers
    userService := &UserService{db: database}
    commandBus.RegisterHandler("CreateUser", userService)
    queryBus.RegisterHandler("GetUser", userService)
    
    // Start monitoring
    monitoring := application.NewMonitoringServer(config.Monitoring, collector)
    go monitoring.Start()
    
    // Execute operations
    ctx := context.Background()
    
    result, err := commandBus.ExecuteCommand(ctx, CreateUserCommand{
        Name:  "John Doe",
        Email: "john@example.com",
    })
    
    if err != nil {
        log.Printf("Command failed: %v", err)
        return
    }
    
    user, err := application.ExecuteTyped[*User](ctx, queryBus, GetUserQuery{
        UserID: result.(int),
    })
    
    if err != nil {
        log.Printf("Query failed: %v", err)
        return
    }
    
    log.Printf("Created user: %+v", user)
}
```

This API reference provides comprehensive documentation for all public interfaces and best practices for using the CQRS implementation effectively in production environments.
