# 🚀 CQRS Usage Guide

## Table of Contents

- [Getting Started](#getting-started)
- [Basic Usage](#basic-usage)
- [Advanced Features](#advanced-features)
- [Production Setup](#production-setup)
- [Troubleshooting](#troubleshooting)
- [Migration Guide](#migration-guide)
- [Performance Tuning](#performance-tuning)

---

## Getting Started

### Installation

```bash
go mod init your-project
go get github.com/your-org/cqrs
```

### Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "your-project/application"
)

// Define your command
type CreateUserCommand struct {
    Name  string
    Email string
}

func (c CreateUserCommand) CommandName() string {
    return "CreateUser"
}

// Define your query
type GetUserQuery struct {
    UserID int
}

func (q GetUserQuery) QueryName() string {
    return "GetUser"
}

// Define your handlers
type UserHandler struct {
    users map[int]*User
}

func (h *UserHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
    switch c := cmd.(type) {
    case CreateUserCommand:
        user := &User{
            ID:    len(h.users) + 1,
            Name:  c.Name,
            Email: c.Email,
        }
        h.users[user.ID] = user
        return user.ID, nil
    default:
        return nil, fmt.Errorf("unknown command: %T", cmd)
    }
}

func (h *UserHandler) HandleQuery(ctx context.Context, query application.Query) (interface{}, error) {
    switch q := query.(type) {
    case GetUserQuery:
        if user, exists := h.users[q.UserID]; exists {
            return user, nil
        }
        return nil, fmt.Errorf("user not found: %d", q.UserID)
    default:
        return nil, fmt.Errorf("unknown query: %T", query)
    }
}

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // Create buses
    commandBus := application.NewCommandBus(application.CommandBusOptions{
        EnableMetrics: true,
    })
    
    queryBus := application.NewQueryBus(application.QueryBusOptions{
        EnableMetrics: true,
    })
    
    // Register handlers
    userHandler := &UserHandler{users: make(map[int]*User)}
    
    commandBus.RegisterHandler("CreateUser", application.CommandHandlerFunc(userHandler.Handle))
    queryBus.RegisterHandler("GetUser", application.QueryHandlerFunc(userHandler.HandleQuery))
    
    ctx := context.Background()
    
    // Execute command
    userID, err := commandBus.ExecuteCommand(ctx, CreateUserCommand{
        Name:  "John Doe",
        Email: "john@example.com",
    })
    if err != nil {
        log.Fatalf("Failed to create user: %v", err)
    }
    
    // Execute query
    user, err := queryBus.Execute(ctx, GetUserQuery{UserID: userID.(int)})
    if err != nil {
        log.Fatalf("Failed to get user: %v", err)
    }
    
    fmt.Printf("Created user: %+v\n", user)
}
```

---

## Basic Usage

### 1. Commands and Queries

#### Command Design Patterns

```go
// Simple command
type CreateOrderCommand struct {
    CustomerID int
    ProductID  int
    Quantity   int
}

func (c CreateOrderCommand) CommandName() string {
    return "CreateOrder"
}

// Command with validation
type UpdateUserCommand struct {
    UserID int    `validate:"required,min=1"`
    Name   string `validate:"required,min=1,max=100"`
    Email  string `validate:"required,email"`
}

func (c UpdateUserCommand) CommandName() string {
    return "UpdateUser"
}

func (c UpdateUserCommand) Validate() error {
    if c.UserID <= 0 {
        return fmt.Errorf("invalid user ID: %d", c.UserID)
    }
    if c.Name == "" {
        return fmt.Errorf("name is required")
    }
    if !strings.Contains(c.Email, "@") {
        return fmt.Errorf("invalid email format")
    }
    return nil
}

// Query with parameters
type FindUsersQuery struct {
    Status   string
    Page     int
    PageSize int
}

func (q FindUsersQuery) QueryName() string {
    return "FindUsers"
}
```

#### Handler Implementation Patterns

```go
// Single responsibility handler
type CreateOrderHandler struct {
    orderRepo OrderRepository
    eventBus  EventBus
}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
    createCmd := cmd.(CreateOrderCommand)
    
    // Business logic
    order := &Order{
        ID:         generateOrderID(),
        CustomerID: createCmd.CustomerID,
        ProductID:  createCmd.ProductID,
        Quantity:   createCmd.Quantity,
        Status:     "pending",
        CreatedAt:  time.Now(),
    }
    
    // Persist
    if err := h.orderRepo.Save(ctx, order); err != nil {
        return nil, fmt.Errorf("failed to save order: %w", err)
    }
    
    // Publish event
    h.eventBus.Publish(OrderCreatedEvent{OrderID: order.ID})
    
    return order.ID, nil
}

// Composite handler for related operations
type UserHandler struct {
    userRepo UserRepository
    cache    Cache
}

func (h *UserHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
    switch c := cmd.(type) {
    case CreateUserCommand:
        return h.createUser(ctx, c)
    case UpdateUserCommand:
        return h.updateUser(ctx, c)
    case DeleteUserCommand:
        return h.deleteUser(ctx, c)
    default:
        return nil, fmt.Errorf("unsupported command: %T", cmd)
    }
}

func (h *UserHandler) HandleQuery(ctx context.Context, query application.Query) (interface{}, error) {
    switch q := query.(type) {
    case GetUserQuery:
        return h.getUser(ctx, q)
    case FindUsersQuery:
        return h.findUsers(ctx, q)
    default:
        return nil, fmt.Errorf("unsupported query: %T", query)
    }
}
```

### 2. Middleware Configuration

#### Basic Middleware Setup

```go
// Command bus with middleware
commandBus := application.NewCommandBus(application.CommandBusOptions{
    InitialCapacity: 128,
    EnableMetrics:   true,
    Middleware: []application.CommandMiddleware{
        application.CommandLoggingMiddleware("CMD"),
        application.ValidationMiddleware(),
        application.TimeoutMiddleware(30 * time.Second),
    },
})

// Query bus with caching and retry
queryBus := application.NewQueryBus(application.QueryBusOptions{
    InitialCapacity: 128,
    EnableMetrics:   true,
    Middleware: []application.QueryMiddleware{
        application.CachingMiddleware(10 * time.Minute),
        application.RetryMiddleware(3, 100*time.Millisecond),
        application.QueryLoggingMiddleware("QRY"),
    },
})
```

#### Custom Middleware

```go
// Custom authentication middleware
func AuthenticationMiddleware(authService AuthService) application.CommandMiddleware {
    return func(next application.CommandHandler) application.CommandHandler {
        return application.CommandHandlerFunc(func(ctx context.Context, cmd application.Command) (interface{}, error) {
            // Extract authentication token from context
            token := ctx.Value("auth_token")
            if token == nil {
                return nil, fmt.Errorf("authentication required")
            }
            
            // Validate token
            userID, err := authService.ValidateToken(token.(string))
            if err != nil {
                return nil, fmt.Errorf("invalid authentication: %w", err)
            }
            
            // Add user context
            ctx = context.WithValue(ctx, "user_id", userID)
            
            return next.Handle(ctx, cmd)
        })
    }
}

// Custom audit middleware
func AuditMiddleware(auditLogger AuditLogger) application.CommandMiddleware {
    return func(next application.CommandHandler) application.CommandHandler {
        return application.CommandHandlerFunc(func(ctx context.Context, cmd application.Command) (interface{}, error) {
            start := time.Now()
            
            // Execute command
            result, err := next.Handle(ctx, cmd)
            
            // Log audit trail
            auditLogger.Log(AuditEvent{
                CommandName: cmd.CommandName(),
                UserID:      ctx.Value("user_id"),
                Timestamp:   start,
                Duration:    time.Since(start),
                Success:     err == nil,
                Error:       err,
            })
            
            return result, err
        })
    }
}
```

---

## Advanced Features

### 1. Type-Safe Query Execution

```go
// Type-safe query execution
user, err := application.ExecuteTyped[*User](ctx, queryBus, GetUserQuery{UserID: 123})
if err != nil {
    return fmt.Errorf("failed to get user: %w", err)
}

// user is automatically typed as *User
fmt.Printf("User name: %s", user.Name)

// Batch type-safe execution
type UserBatch struct {
    Users []User
    Total int
}

batch, err := application.ExecuteTyped[*UserBatch](ctx, queryBus, FindUsersQuery{
    Status:   "active",
    Page:     1,
    PageSize: 10,
})
```

### 2. Circuit Breaker Pattern

```go
// Configure circuit breaker
circuitBreaker := application.NewCircuitBreakerMiddleware(
    5,                    // failure threshold
    30*time.Second,       // timeout
)

commandBus := application.NewCommandBus(application.CommandBusOptions{
    Middleware: []application.CommandMiddleware{
        circuitBreaker.ForCommands(),
        application.TimeoutMiddleware(10 * time.Second),
    },
})

// Monitor circuit breaker state
go func() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        state := circuitBreaker.GetState()
        fmt.Printf("Circuit breaker state: %s\n", state)
    }
}()
```

### 3. Rate Limiting

```go
// Configure rate limiter
rateLimiter := application.NewRateLimitingMiddleware(
    1000,               // requests per window
    1*time.Minute,      // window duration
)

queryBus := application.NewQueryBus(application.QueryBusOptions{
    Middleware: []application.QueryMiddleware{
        rateLimiter.ForQueries(),
        application.CachingMiddleware(5 * time.Minute),
    },
})

// Check rate limit status
fmt.Printf("Remaining tokens: %d\n", rateLimiter.RemainingTokens())
```

### 4. Advanced Caching Strategies

```go
// Cache with custom key generation
func CacheKeyMiddleware() application.QueryMiddleware {
    cache := make(map[string]interface{})
    mutex := sync.RWMutex{}
    
    return func(next application.QueryHandler) application.QueryHandler {
        return application.QueryHandlerFunc(func(ctx context.Context, query application.Query) (interface{}, error) {
            // Generate custom cache key
            key := fmt.Sprintf("%s:%v", query.QueryName(), query)
            
            // Check cache
            mutex.RLock()
            if cached, exists := cache[key]; exists {
                mutex.RUnlock()
                return cached, nil
            }
            mutex.RUnlock()
            
            // Execute query
            result, err := next.Handle(ctx, query)
            if err != nil {
                return nil, err
            }
            
            // Cache result
            mutex.Lock()
            cache[key] = result
            mutex.Unlock()
            
            return result, nil
        })
    }
}

// Conditional caching
func ConditionalCacheMiddleware(shouldCache func(query application.Query) bool) application.QueryMiddleware {
    return func(next application.QueryHandler) application.QueryHandler {
        return application.QueryHandlerFunc(func(ctx context.Context, query application.Query) (interface{}, error) {
            if !shouldCache(query) {
                return next.Handle(ctx, query)
            }
            
            // Apply caching logic only for selected queries
            return application.CachingMiddleware(10*time.Minute)(next).Handle(ctx, query)
        })
    }
}
```

---

## Production Setup

### 1. Configuration Management

#### Environment-Specific Configurations

```bash
# Directory structure
configs/
├── production.json
├── staging.json
├── development.json
└── test.json
```

```go
// Load configuration based on environment
func LoadEnvironmentConfig() (*application.Config, error) {
    env := os.Getenv("ENV")
    if env == "" {
        env = "development"
    }
    
    configPath := fmt.Sprintf("configs/%s.json", env)
    return application.LoadConfig(configPath)
}

// Configuration with environment overrides
func LoadConfigWithOverrides() (*application.Config, error) {
    // Load base configuration
    config, err := application.LoadConfig("configs/production.json")
    if err != nil {
        return nil, err
    }
    
    // Apply environment variable overrides
    if envConfig, err := application.LoadConfigFromEnv(); err == nil {
        config = mergeConfigs(config, envConfig)
    }
    
    return config, nil
}
```

#### Dynamic Configuration

```go
// Configuration hot-reload
type ConfigManager struct {
    config     *application.Config
    configPath string
    mutex      sync.RWMutex
    callbacks  []func(*application.Config)
}

func NewConfigManager(configPath string) *ConfigManager {
    return &ConfigManager{
        configPath: configPath,
        callbacks:  make([]func(*application.Config), 0),
    }
}

func (cm *ConfigManager) LoadConfig() error {
    config, err := application.LoadConfig(cm.configPath)
    if err != nil {
        return err
    }
    
    cm.mutex.Lock()
    cm.config = config
    cm.mutex.Unlock()
    
    // Notify callbacks
    for _, callback := range cm.callbacks {
        callback(config)
    }
    
    return nil
}

func (cm *ConfigManager) WatchConfig() {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Printf("Failed to create config watcher: %v", err)
        return
    }
    defer watcher.Close()
    
    watcher.Add(cm.configPath)
    
    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                if err := cm.LoadConfig(); err != nil {
                    log.Printf("Failed to reload config: %v", err)
                }
            }
        case err := <-watcher.Errors:
            log.Printf("Config watcher error: %v", err)
        }
    }
}
```

### 2. Monitoring and Observability

#### Comprehensive Monitoring Setup

```go
// Production monitoring setup
func SetupMonitoring(config *application.Config) (*application.MonitoringServer, error) {
    // Create metrics collector
    collector := application.NewMetricsCollector()
    
    // Create health checker
    healthChecker := application.NewHealthChecker()
    
    // Add health checks
    healthChecker.AddCheck("database", func() application.HealthStatus {
        if err := db.Ping(); err != nil {
            return application.HealthStatus{Healthy: false, Message: err.Error()}
        }
        return application.HealthStatus{Healthy: true, Message: "OK"}
    })
    
    healthChecker.AddCheck("cache", func() application.HealthStatus {
        if err := cache.Ping(); err != nil {
            return application.HealthStatus{Healthy: false, Message: err.Error()}
        }
        return application.HealthStatus{Healthy: true, Message: "OK"}
    })
    
    // Create monitoring server
    monitoringServer := application.NewMonitoringServer(config.Monitoring, collector)
    
    return monitoringServer, nil
}

// Custom metrics collection
func CustomMetricsMiddleware(collector *application.MetricsCollector) application.CommandMiddleware {
    return func(next application.CommandHandler) application.CommandHandler {
        return application.CommandHandlerFunc(func(ctx context.Context, cmd application.Command) (interface{}, error) {
            start := time.Now()
            
            result, err := next.Handle(ctx, cmd)
            
            duration := time.Since(start)
            
            // Collect custom metrics
            collector.RecordCustomMetric("command_duration", map[string]interface{}{
                "command_name": cmd.CommandName(),
                "duration_ms":  duration.Milliseconds(),
                "success":      err == nil,
            })
            
            return result, err
        })
    }
}
```

#### Distributed Tracing

```go
// OpenTelemetry integration
func TracingMiddleware(tracer trace.Tracer) application.CommandMiddleware {
    return func(next application.CommandHandler) application.CommandHandler {
        return application.CommandHandlerFunc(func(ctx context.Context, cmd application.Command) (interface{}, error) {
            ctx, span := tracer.Start(ctx, cmd.CommandName())
            defer span.End()
            
            span.SetAttributes(
                attribute.String("command.name", cmd.CommandName()),
                attribute.String("command.type", "command"),
            )
            
            result, err := next.Handle(ctx, cmd)
            
            if err != nil {
                span.RecordError(err)
                span.SetStatus(codes.Error, err.Error())
            }
            
            return result, err
        })
    }
}

// Jaeger setup
func SetupJaeger(serviceName string) (trace.Tracer, func(), error) {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
    if err != nil {
        return nil, nil, err
    }
    
    tp := tracesdk.NewTracerProvider(
        tracesdk.WithBatcher(exporter),
        tracesdk.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )
    
    otel.SetTracerProvider(tp)
    
    cleanup := func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        tp.Shutdown(ctx)
    }
    
    return tp.Tracer("cqrs"), cleanup, nil
}
```

### 3. Graceful Shutdown

```go
// Production server with graceful shutdown
func RunProductionServer(config *application.Config) error {
    // Setup components
    commandBus := application.NewCommandBusFromConfig(config)
    queryBus := application.NewQueryBusFromConfig(config)
    monitoringServer, err := SetupMonitoring(config)
    if err != nil {
        return fmt.Errorf("failed to setup monitoring: %w", err)
    }
    
    // Start monitoring server
    go func() {
        if err := monitoringServer.Start(); err != nil {
            log.Printf("Monitoring server error: %v", err)
        }
    }()
    
    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Main application loop
    go func() {
        // Your application logic here
        runApplication(commandBus, queryBus)
    }()
    
    // Wait for signal
    sig := <-sigChan
    log.Printf("Received signal: %v", sig)
    
    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    var wg sync.WaitGroup
    
    // Stop monitoring server
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := monitoringServer.Stop(); err != nil {
            log.Printf("Error stopping monitoring server: %v", err)
        }
    }()
    
    // Wait for shutdown to complete
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        log.Println("Graceful shutdown completed")
    case <-ctx.Done():
        log.Println("Shutdown timeout exceeded")
    }
    
    return nil
}
```

---

## Troubleshooting

### Common Issues and Solutions

#### 1. Handler Not Found

```go
// Problem: ErrHandlerNotFound
result, err := commandBus.ExecuteCommand(ctx, MyCommand{})
if errors.Is(err, application.ErrHandlerNotFound) {
    // Check if handler is registered
    registered := commandBus.GetRegisteredCommands()
    log.Printf("Registered commands: %v", registered)
    
    // Ensure CommandName() returns correct value
    cmd := MyCommand{}
    log.Printf("Command name: %s", cmd.CommandName())
}
```

#### 2. Memory Leaks

```go
// Problem: Memory growing over time
// Solution: Check cache configuration and implement cleanup

// Add cache cleanup
func CachingMiddlewareWithCleanup(ttl time.Duration) application.QueryMiddleware {
    cache := make(map[string]cacheEntry)
    mutex := sync.RWMutex{}
    
    // Cleanup goroutine
    go func() {
        ticker := time.NewTicker(ttl / 2)
        defer ticker.Stop()
        
        for range ticker.C {
            now := time.Now()
            mutex.Lock()
            for key, entry := range cache {
                if now.After(entry.expiry) {
                    delete(cache, key)
                }
            }
            mutex.Unlock()
        }
    }()
    
    return func(next application.QueryHandler) application.QueryHandler {
        // Implementation...
    }
}
```

#### 3. Performance Issues

```go
// Problem: Slow query execution
// Solution: Add performance debugging

func PerformanceDebugMiddleware() application.QueryMiddleware {
    return func(next application.QueryHandler) application.QueryHandler {
        return application.QueryHandlerFunc(func(ctx context.Context, query application.Query) (interface{}, error) {
            start := time.Now()
            
            result, err := next.Handle(ctx, query)
            
            duration := time.Since(start)
            if duration > 100*time.Millisecond {
                log.Printf("SLOW QUERY: %s took %v", query.QueryName(), duration)
            }
            
            return result, err
        })
    }
}
```

#### 4. Circuit Breaker Issues

```go
// Problem: Circuit breaker stuck open
// Solution: Monitor and reset manually if needed

func MonitorCircuitBreaker(cb *application.CircuitBreakerMiddleware) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        state := cb.GetState()
        if state == "open" {
            log.Printf("Circuit breaker is OPEN - investigating...")
            
            // Check underlying service health
            if isServiceHealthy() {
                log.Printf("Service is healthy, resetting circuit breaker")
                cb.Reset()
            }
        }
    }
}
```

### Debug Mode

```go
// Enable debug logging
func EnableDebugMode(commandBus *application.CommandBus, queryBus *application.QueryBus) {
    // Add debug middleware
    debugMiddleware := func(next application.CommandHandler) application.CommandHandler {
        return application.CommandHandlerFunc(func(ctx context.Context, cmd application.Command) (interface{}, error) {
            log.Printf("DEBUG: Executing command %s with data: %+v", cmd.CommandName(), cmd)
            
            start := time.Now()
            result, err := next.Handle(ctx, cmd)
            duration := time.Since(start)
            
            if err != nil {
                log.Printf("DEBUG: Command %s failed after %v: %v", cmd.CommandName(), duration, err)
            } else {
                log.Printf("DEBUG: Command %s completed after %v, result: %+v", cmd.CommandName(), duration, result)
            }
            
            return result, err
        })
    }
    
    // This would require middleware reregistration in a real implementation
    // commandBus.AddMiddleware(debugMiddleware)
}
```

---

## Migration Guide

### From Version 1.x to 2.x

#### Interface Changes

```go
// Old version (1.x)
type CommandHandler interface {
    Execute(cmd Command) (interface{}, error)
}

// New version (2.x)
type CommandHandler interface {
    Handle(ctx context.Context, cmd Command) (interface{}, error)
}

// Migration
func migrateHandler(oldHandler OldCommandHandler) application.CommandHandler {
    return application.CommandHandlerFunc(func(ctx context.Context, cmd application.Command) (interface{}, error) {
        return oldHandler.Execute(cmd)
    })
}
```

#### Configuration Changes

```go
// Old configuration
type OldConfig struct {
    MaxHandlers int
    Timeout     int // seconds
}

// New configuration
func migrateConfig(old OldConfig) application.CommandBusOptions {
    return application.CommandBusOptions{
        InitialCapacity: old.MaxHandlers,
        EnableMetrics:   true,
        Middleware: []application.CommandMiddleware{
            application.TimeoutMiddleware(time.Duration(old.Timeout) * time.Second),
        },
    }
}
```

---

## Performance Tuning

### 1. Optimal Configuration

```go
// High-throughput configuration
func HighThroughputConfig() *application.Config {
    config := application.DefaultConfig()
    
    // Increase capacities
    config.CommandBus.InitialCapacity = 1024
    config.QueryBus.InitialCapacity = 1024
    
    // Optimize caching
    config.Middleware.Caching.MaxSize = 100000
    config.Middleware.Caching.DefaultTTLSec = 300 // 5 minutes
    
    // Optimize retry strategy
    config.Middleware.Retry.MaxAttempts = 2
    config.Middleware.Retry.BaseDelayMs = 10
    
    // Enable performance features
    config.Performance.EnablePooling = true
    config.Performance.PoolSize = 500
    config.Performance.EnablePreallocation = true
    config.Performance.GCPercent = 50 // Reduce GC pressure
    
    return config
}

// Low-latency configuration
func LowLatencyConfig() *application.Config {
    config := application.DefaultConfig()
    
    // Disable expensive middleware
    config.Middleware.Logging.Enabled = false
    config.Middleware.Validation.Enabled = false
    
    // Minimal retry
    config.Middleware.Retry.MaxAttempts = 1
    
    // Aggressive caching
    config.Middleware.Caching.DefaultTTLSec = 60
    
    return config
}
```

### 2. Benchmarking and Profiling

```go
// Performance testing
func BenchmarkProductionWorkload(b *testing.B) {
    config := HighThroughputConfig()
    commandBus := application.NewCommandBusFromConfig(config)
    queryBus := application.NewQueryBusFromConfig(config)
    
    // Register handlers
    setupHandlers(commandBus, queryBus)
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        ctx := context.Background()
        
        for pb.Next() {
            // Mixed workload
            if rand.Intn(3) == 0 {
                commandBus.ExecuteCommand(ctx, CreateUserCommand{
                    Name:  "Test User",
                    Email: "test@example.com",
                })
            } else {
                queryBus.Execute(ctx, GetUserQuery{UserID: rand.Intn(1000)})
            }
        }
    })
}

// Memory profiling
func ProfileMemoryUsage() {
    f, err := os.Create("mem.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    runtime.GC()
    if err := pprof.WriteHeapProfile(f); err != nil {
        log.Fatal(err)
    }
}
```

### 3. Resource Optimization

```go
// Pool-based handler optimization
type PooledHandler struct {
    handlerPool sync.Pool
}

func NewPooledHandler() *PooledHandler {
    return &PooledHandler{
        handlerPool: sync.Pool{
            New: func() interface{} {
                return &actualHandler{
                    // Initialize expensive resources
                }
            },
        },
    }
}

func (ph *PooledHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
    handler := ph.handlerPool.Get().(*actualHandler)
    defer ph.handlerPool.Put(handler)
    
    return handler.Process(ctx, cmd)
}

// Connection pooling
type DatabaseHandler struct {
    dbPool *sql.DB
}

func (h *DatabaseHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
    // Use connection from pool
    conn, err := h.dbPool.Conn(ctx)
    if err != nil {
        return nil, err
    }
    defer conn.Close()
    
    // Execute with pooled connection
    return h.executeWithConnection(ctx, conn, cmd)
}
```

This comprehensive usage guide provides practical examples and patterns for effectively using the CQRS implementation in real-world scenarios, from basic usage to advanced production deployments.
