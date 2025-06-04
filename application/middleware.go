package application

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AuthenticationMiddleware provides authentication support for commands and queries
type AuthenticationMiddleware struct {
	userStore map[string]string // userID -> role
	mutex     sync.RWMutex
}

func NewAuthenticationMiddleware() *AuthenticationMiddleware {
	return &AuthenticationMiddleware{
		userStore: make(map[string]string),
	}
}

func (am *AuthenticationMiddleware) AddUser(userID, role string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.userStore[userID] = role
}

func (am *AuthenticationMiddleware) ForCommands(requiredRole string) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
			userID := ctx.Value("userID")
			if userID == nil {
				return nil, fmt.Errorf("authentication required")
			}

			am.mutex.RLock()
			userRole, exists := am.userStore[userID.(string)]
			am.mutex.RUnlock()

			if !exists {
				return nil, fmt.Errorf("user not found")
			}

			if userRole != requiredRole && requiredRole != "any" {
				return nil, fmt.Errorf("insufficient permissions: required %s, have %s", requiredRole, userRole)
			}

			return next.Handle(ctx, cmd)
		})
	}
}

func (am *AuthenticationMiddleware) ForQueries(requiredRole string) QueryMiddleware {
	return func(next QueryHandler) QueryHandler {
		return QueryHandlerFunc(func(ctx context.Context, query Query) (interface{}, error) {
			userID := ctx.Value("userID")
			if userID == nil {
				return nil, fmt.Errorf("authentication required")
			}

			am.mutex.RLock()
			userRole, exists := am.userStore[userID.(string)]
			am.mutex.RUnlock()

			if !exists {
				return nil, fmt.Errorf("user not found")
			}

			if userRole != requiredRole && requiredRole != "any" {
				return nil, fmt.Errorf("insufficient permissions: required %s, have %s", requiredRole, userRole)
			}

			return next.Handle(ctx, query)
		})
	}
}

// Note: CircuitBreakerMiddleware and RateLimitingMiddleware are implemented in advanced_middleware.go

// AuditMiddleware logs all commands and queries for auditing purposes
type AuditMiddleware struct {
	logger *log.Logger
}

func NewAuditMiddleware(logger *log.Logger) *AuditMiddleware {
	if logger == nil {
		logger = log.Default()
	}
	return &AuditMiddleware{logger: logger}
}

func (am *AuditMiddleware) ForCommands() CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
			userID := "anonymous"
			if uid := ctx.Value("userID"); uid != nil {
				userID = uid.(string)
			}

			start := time.Now()
			result, err := next.Handle(ctx, cmd)
			duration := time.Since(start)

			status := "SUCCESS"
			if err != nil {
				status = "FAILED"
			}

			am.logger.Printf("AUDIT: COMMAND [%s] User: %s, Command: %s, Status: %s, Duration: %v",
				time.Now().Format(time.RFC3339), userID, cmd.CommandName(), status, duration)

			return result, err
		})
	}
}

func (am *AuditMiddleware) ForQueries() QueryMiddleware {
	return func(next QueryHandler) QueryHandler {
		return QueryHandlerFunc(func(ctx context.Context, query Query) (interface{}, error) {
			userID := "anonymous"
			if uid := ctx.Value("userID"); uid != nil {
				userID = uid.(string)
			}

			start := time.Now()
			result, err := next.Handle(ctx, query)
			duration := time.Since(start)

			status := "SUCCESS"
			if err != nil {
				status = "FAILED"
			}

			am.logger.Printf("AUDIT: QUERY [%s] User: %s, Query: %s, Status: %s, Duration: %v",
				time.Now().Format(time.RFC3339), userID, query.QueryName(), status, duration)

			return result, err
		})
	}
}
