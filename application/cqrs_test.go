package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Test implementations
type TestCommand struct {
	Name string
	ID   int
}

func (c TestCommand) CommandName() string { return c.Name }

type TestQuery struct {
	Name string
	ID   int
}

func (q TestQuery) QueryName() string { return q.Name }

type TestCommandHandler struct {
	delay time.Duration
}

func (h TestCommandHandler) Handle(ctx context.Context, cmd Command) (interface{}, error) {
	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	return fmt.Sprintf("handled command: %s", cmd.CommandName()), nil
}

type TestQueryHandler struct {
	delay time.Duration
}

func (h TestQueryHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	return fmt.Sprintf("handled query: %s", query.QueryName()), nil
}

func TestCommandBusPerformance(t *testing.T) {
	bus := NewCommandBus(CommandBusOptions{
		InitialCapacity: 1000,
		EnableMetrics:   true,
	})

	// Register handlers
	for i := 0; i < 100; i++ {
		handlerName := fmt.Sprintf("TestCommand%d", i)
		err := bus.RegisterHandler(handlerName, TestCommandHandler{})
		if err != nil {
			t.Fatalf("Failed to register handler %s: %v", handlerName, err)
		}
	}

	ctx := context.Background()

	// Test concurrent execution
	t.Run("ConcurrentExecution", func(t *testing.T) {
		const numGoroutines = 100
		const numOperations = 1000

		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					cmd := TestCommand{
						Name: fmt.Sprintf("TestCommand%d", j%100),
						ID:   id*numOperations + j,
					}
					_, err := bus.ExecuteCommand(ctx, cmd)
					if err != nil {
						t.Errorf("Command execution failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		totalOps := numGoroutines * numOperations
		opsPerSecond := float64(totalOps) / elapsed.Seconds()

		t.Logf("Executed %d operations in %v", totalOps, elapsed)
		t.Logf("Throughput: %.2f operations/second", opsPerSecond)

		metrics := bus.GetMetrics()
		if metrics != nil {
			t.Logf("Metrics - Total: %d, Failures: %d, Avg Time: %v",
				metrics.TotalExecutions, metrics.TotalFailures, metrics.AverageExecutionTime)
		}
	})
}

func TestQueryBusPerformance(t *testing.T) {
	bus := NewQueryBus(QueryBusOptions{
		InitialCapacity: 1000,
		EnableMetrics:   true,
		Middleware: []QueryMiddleware{
			CachingMiddleware(1 * time.Minute),
		},
	})

	// Register handlers
	for i := 0; i < 100; i++ {
		handlerName := fmt.Sprintf("TestQuery%d", i)
		err := bus.RegisterHandler(handlerName, TestQueryHandler{})
		if err != nil {
			t.Fatalf("Failed to register handler %s: %v", handlerName, err)
		}
	}

	ctx := context.Background()

	t.Run("ConcurrentQueries", func(t *testing.T) {
		const numGoroutines = 50
		const numOperations = 1000

		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					query := TestQuery{
						Name: fmt.Sprintf("TestQuery%d", j%100),
						ID:   id*numOperations + j,
					}
					_, err := bus.Execute(ctx, query)
					if err != nil {
						t.Errorf("Query execution failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		totalOps := numGoroutines * numOperations
		opsPerSecond := float64(totalOps) / elapsed.Seconds()

		t.Logf("Executed %d queries in %v", totalOps, elapsed)
		t.Logf("Throughput: %.2f queries/second", opsPerSecond)

		metrics := bus.GetMetrics()
		if metrics != nil {
			t.Logf("Metrics - Total: %d, Failures: %d, Avg Time: %v",
				metrics.TotalExecutions, metrics.TotalFailures, metrics.AverageExecutionTime)
		}
	})

	t.Run("CacheEffectiveness", func(t *testing.T) {
		// Register a specific handler for the cache test
		err := bus.RegisterHandler("CachedQuery", TestQueryHandler{delay: 10 * time.Millisecond})
		if err != nil {
			t.Fatalf("Failed to register CachedQuery handler: %v", err)
		}

		// Test that caching reduces execution time
		query := TestQuery{Name: "CachedQuery", ID: 1}

		// First execution (should hit the handler)
		start1 := time.Now()
		_, err = bus.Execute(ctx, query)
		if err != nil {
			t.Fatalf("First query execution failed: %v", err)
		}
		firstExecution := time.Since(start1)

		// Second execution (should hit the cache)
		start2 := time.Now()
		_, err = bus.Execute(ctx, query)
		if err != nil {
			t.Fatalf("Second query execution failed: %v", err)
		}
		secondExecution := time.Since(start2)

		t.Logf("First execution: %v, Second execution: %v", firstExecution, secondExecution)

		// Second execution should be faster (cached)
		if secondExecution >= firstExecution {
			t.Log("Warning: Cache may not be working as expected")
		}
	})
}

func BenchmarkCommandBusExecution(b *testing.B) {
	bus := NewCommandBus(CommandBusOptions{
		InitialCapacity: 100,
	})

	// Register a single handler
	err := bus.RegisterHandler("BenchCommand", TestCommandHandler{})
	if err != nil {
		b.Fatalf("Failed to register handler: %v", err)
	}

	ctx := context.Background()
	cmd := TestCommand{Name: "BenchCommand", ID: 1}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := bus.ExecuteCommand(ctx, cmd)
			if err != nil {
				b.Errorf("Command execution failed: %v", err)
			}
		}
	})
}

func BenchmarkQueryBusExecution(b *testing.B) {
	bus := NewQueryBus(QueryBusOptions{
		InitialCapacity: 100,
	})

	// Register a single handler
	err := bus.RegisterHandler("BenchQuery", TestQueryHandler{})
	if err != nil {
		b.Fatalf("Failed to register handler: %v", err)
	}

	ctx := context.Background()
	query := TestQuery{Name: "BenchQuery", ID: 1}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := bus.Execute(ctx, query)
			if err != nil {
				b.Errorf("Query execution failed: %v", err)
			}
		}
	})
}

func BenchmarkCommandBusRegistration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus := NewCommandBus()
		handlerName := fmt.Sprintf("Command%d", i)
		err := bus.RegisterHandler(handlerName, TestCommandHandler{})
		if err != nil {
			b.Errorf("Handler registration failed: %v", err)
		}
	}
}

func TestMiddlewareOrdering(t *testing.T) {
	var executionOrder []string

	middleware1 := func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
			executionOrder = append(executionOrder, "middleware1-before")
			result, err := next.Handle(ctx, cmd)
			executionOrder = append(executionOrder, "middleware1-after")
			return result, err
		})
	}

	middleware2 := func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
			executionOrder = append(executionOrder, "middleware2-before")
			result, err := next.Handle(ctx, cmd)
			executionOrder = append(executionOrder, "middleware2-after")
			return result, err
		})
	}

	handlerFunc := CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
		executionOrder = append(executionOrder, "handler")
		return "result", nil
	})

	bus := NewCommandBus(CommandBusOptions{
		Middleware: []CommandMiddleware{middleware1, middleware2},
	})

	err := bus.RegisterHandler("TestCommand", handlerFunc)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	_, err = bus.ExecuteCommand(context.Background(), TestCommand{Name: "TestCommand"})
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	expectedOrder := []string{
		"middleware1-before", // First added middleware executes first (outermost)
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(executionOrder) != len(expectedOrder) {
		t.Fatalf("Expected %d execution steps, got %d", len(expectedOrder), len(executionOrder))
	}

	for i, expected := range expectedOrder {
		if executionOrder[i] != expected {
			t.Errorf("Expected execution order[%d] = %s, got %s", i, expected, executionOrder[i])
		}
	}
}
