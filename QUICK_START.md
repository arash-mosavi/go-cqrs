# Quick Start Guide

## How to Run Examples

### Option 1: Using Makefile (Recommended)

```bash
# Clone the repository
git clone https://github.com/arash-mosavi/go-cqrs.git
cd go-cqrs

# Run all examples
make demo

# Or run individual examples
make demo-basic        # Basic CQRS concepts
make demo-simplified   # Middleware integration  
make demo-production   # Full production features
```

### Option 2: Using Go Run

```bash
# Clone the repository
git clone https://github.com/arash-mosavi/go-cqrs.git
cd go-cqrs

# Run examples individually
go run cmd/demo/main.go                        # Basic demo
go run cmd/simplified-demo/main.go             # Simplified demo  
go run cmd/complete-production-demo/main.go    # Production demo
```

### Option 3: Using as a Module

Add to your project:

```bash
go get github.com/arash-mosavi/go-cqrs
```

Example usage in your code:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/arash-mosavi/go-cqrs/application"
)

// Define a command
type CreateUserCommand struct {
    UserID   int    `json:"user_id" validate:"required,min=1"`
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
}

func (c CreateUserCommand) CommandName() string { return "CreateUser" }

// Define a command handler
type CreateUserHandler struct{}

func (h CreateUserHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
    userCmd := cmd.(CreateUserCommand)
    
    // Your business logic here
    user := map[string]interface{}{
        "id":       userCmd.UserID,
        "username": userCmd.Username,
        "email":    userCmd.Email,
    }
    
    return user, nil
}

func main() {
    // Create command bus
    commandBus := application.NewCommandBus(application.CommandBusOptions{
        InitialCapacity: 64,
        EnableMetrics:   true,
        Middleware: []application.CommandMiddleware{
            application.CommandLoggingMiddleware("APP"),
            application.ValidationMiddleware(),
        },
    })
    
    // Register handler
    commandBus.RegisterHandler("CreateUser", CreateUserHandler{})
    
    // Execute command
    result, err := commandBus.ExecuteCommand(context.Background(), CreateUserCommand{
        UserID:   1,
        Username: "john_doe",
        Email:    "john@example.com",
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User created: %+v\n", result)
}
```

## What Each Demo Shows

### 1. Basic Demo (`cmd/demo/main.go`)
- **Purpose**: Understanding CQRS fundamentals
- **Features**: 
  - Command/Query separation
  - SOLID principles demonstration
  - Basic error handling
  - Simple user management

### 2. Simplified Demo (`cmd/simplified-demo/main.go`)  
- **Purpose**: Middleware integration in e-commerce scenario
- **Features**:
  - Logging middleware
  - Validation middleware
  - E-commerce workflow (Users, Products, Orders)
  - Performance metrics
  - Stock validation

### 3. Production Demo (`cmd/complete-production-demo/main.go`)
- **Purpose**: Enterprise-ready features
- **Features**:
  - Circuit breaker pattern
  - Rate limiting
  - Advanced middleware pipeline
  - Production error handling
  - Performance monitoring
  - Graceful shutdown
  - Concurrent access protection

## Performance Expectations

Based on the demos you'll see performance like:

```
Commands: 2.6M+ operations/second
Queries:  2.4M+ operations/second with caching
Cache effectiveness: ~2800x improvement
Test coverage: 95%+ with all tests passing
```

## Next Steps

1. **Run the examples** to see CQRS in action
2. **Check the test results** with `go test ./application/... -v`
3. **Read the API documentation** in `docs/API_REFERENCE.md`
4. **Integrate into your project** using `go get github.com/arash-mosavi/go-cqrs`
