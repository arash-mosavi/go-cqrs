# go-cqrs

A grade CQRS framework in Go.

##  Features
###  Performance & Scalability
- Thread-safe with read-write mutexes for concurrent operations
- Minimal memory allocations with optimized zero-copy execution paths
- Benchmark suite included for evaluating performance

###  Design
- Error-first: No panics, just clean error handling
- Full `context.Context` support
- Middleware system with support for logging, validation, timeouts, etc.
- Circuit breaker and rate limiting support
- Real-time metrics and observability integrations

###  Developer Experience
- Type-safe interfaces using Go generics
- Clean architecture and SOLID principles
- Comprehensive test coverage: unit, integration, and benchmark
- Rich example set (from basic to production scenarios)
- Structured documentation with guides and API reference

##  Installation

```bash
git clone https://github.com/arash-mosavi/go-cqrs.git
cd go-cqrs
go mod tidy
go test ./...
```

##  Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/arash-mosavi/go-cqrs/application"
)

type CreateUserCommand struct {
    UserID   int    `json:"user_id" validate:"required,min=1"`
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
}

func (c CreateUserCommand) CommandName() string { return "CreateUser" }

type CreateUserHandler struct{}

func (h CreateUserHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
    user := cmd.(CreateUserCommand)
    return map[string]interface{}{
        "id": user.UserID,
        "username": user.Username,
        "email": user.Email,
        "created": time.Now(),
    }, nil
}

func main() {
    bus := application.NewCommandBus(application.CommandBusOptions{
        InitialCapacity: 64,
        EnableMetrics: true,
        Middleware: []application.CommandMiddleware{
            application.LoggingMiddleware(log.Default()),
            application.ValidationMiddleware(),
            application.TimeoutMiddleware(5 * time.Second),
        },
    })

    _ = bus.RegisterHandler("CreateUser", CreateUserHandler{})

    result, err := bus.ExecuteCommand(context.Background(), CreateUserCommand{
        UserID: 123, Username: "john_doe", Email: "john@example.com",
    })

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Created: %+v\n", result)
}
```

##  Architecture

```
Commands --> CommandBus --> Handlers
Queries  --> QueryBus   --> Handlers
Middleware Stack: Logging, Validation, Timeout, Circuit Breaker, Rate Limiting
```

##  Middleware System

### Command Middleware
- LoggingMiddleware
- ValidationMiddleware
- TimeoutMiddleware
- CircuitBreakerMiddleware
- RateLimitMiddleware

### Query Middleware
- CachingMiddleware
- RetryMiddleware
- LoggingMiddleware

##  Benchmark Results

| Operation               | ns/op | allocs/op |
|------------------------|-------|-----------|
| RegisterHandler        | 150   | 1         |
| ExecuteCommand         | 287   | 2         |
| ExecuteQuery           | 412   | 3         |
| ExecuteWithCache       | 89    | 1         |
| ConcurrentCommandExec  | 756   | 6         |

##  Examples

### Run all examples
```bash
make demo
```

Or manually:
```bash
go run cmd/demo/main.go
go run cmd/simplified-demo/main.go
go run cmd/complete-production-demo/main.go
```

##  Testing

```bash
go test ./...
go test -cover ./...
go test -bench=. ./...
go tool cover -html=coverage.out
```

Test files:
- `cqrs_test.go`
- `benchmark_test.go`
- `integration_test.go`
- `middleware_test.go`

##  Using as Dependency

```bash
go get github.com/arash-mosavi/go-cqrs
```

##  Contributing

Contributions are welcome! Please open issues or PRs if you have suggestions or improvements.

---

Built with performance, safety, and scalability in mind. Designed for real-world usage.

---

MIT License  
(c) Arash Mosavi