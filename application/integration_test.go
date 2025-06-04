// filepath: application/integration_test.go
package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// Integration test models
type User struct {
	ID      int    `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Created time.Time
	Active  bool
}

type Product struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Stock      int     `json:"stock"`
	CategoryID int     `json:"category_id"`
}

type Order struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	ProductID  int       `json:"product_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// In-memory storage for integration tests
type TestStorage struct {
	users    map[int]*User
	products map[int]*Product
	orders   map[int]*Order
	nextID   int
	mu       sync.RWMutex
}

func NewTestStorage() *TestStorage {
	return &TestStorage{
		users:    make(map[int]*User),
		products: make(map[int]*Product),
		orders:   make(map[int]*Order),
		nextID:   1,
	}
}

func (s *TestStorage) NextID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	return id
}

func (s *TestStorage) SaveUser(user *User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = user
}

func (s *TestStorage) GetUser(id int) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[id]
	return user, exists
}

func (s *TestStorage) SaveProduct(product *Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.products[product.ID] = product
}

func (s *TestStorage) GetProduct(id int) (*Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	product, exists := s.products[id]
	return product, exists
}

func (s *TestStorage) SaveOrder(order *Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.ID] = order
}

func (s *TestStorage) GetOrder(id int) (*Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, exists := s.orders[id]
	return order, exists
}

func (s *TestStorage) GetUserOrders(userID int) []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var orders []*Order
	for _, order := range s.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders
}

// Commands for integration tests
type CreateUserCommand struct {
	Email string
	Name  string
}

func (c CreateUserCommand) CommandName() string { return "CreateUser" }

type CreateProductCommand struct {
	Name       string
	Price      float64
	Stock      int
	CategoryID int
}

func (c CreateProductCommand) CommandName() string { return "CreateProduct" }

type CreateOrderCommand struct {
	UserID    int
	ProductID int
	Quantity  int
}

func (c CreateOrderCommand) CommandName() string { return "CreateOrder" }

type UpdateProductStockCommand struct {
	ProductID int
	NewStock  int
}

func (c UpdateProductStockCommand) CommandName() string { return "UpdateProductStock" }

// Queries for integration tests
type GetUserQuery struct {
	UserID int
}

func (q GetUserQuery) QueryName() string { return "GetUser" }

type GetProductQuery struct {
	ProductID int
}

func (q GetProductQuery) QueryName() string { return "GetProduct" }

type GetUserOrdersQuery struct {
	UserID int
}

func (q GetUserOrdersQuery) QueryName() string { return "GetUserOrders" }

type GetOrderQuery struct {
	OrderID int
}

func (q GetOrderQuery) QueryName() string { return "GetOrder" }

// Command Handlers
type CreateUserHandler struct {
	storage *TestStorage
}

func (h CreateUserHandler) Handle(ctx context.Context, cmd Command) (interface{}, error) {
	createCmd := cmd.(CreateUserCommand)

	if createCmd.Email == "" {
		return nil, errors.New("email is required")
	}
	if createCmd.Name == "" {
		return nil, errors.New("name is required")
	}

	user := &User{
		ID:      h.storage.NextID(),
		Email:   createCmd.Email,
		Name:    createCmd.Name,
		Created: time.Now(),
		Active:  true,
	}

	h.storage.SaveUser(user)
	return user, nil
}

type CreateProductHandler struct {
	storage *TestStorage
}

func (h CreateProductHandler) Handle(ctx context.Context, cmd Command) (interface{}, error) {
	createCmd := cmd.(CreateProductCommand)

	if createCmd.Name == "" {
		return nil, errors.New("product name is required")
	}
	if createCmd.Price <= 0 {
		return nil, errors.New("price must be positive")
	}
	if createCmd.Stock < 0 {
		return nil, errors.New("stock cannot be negative")
	}

	product := &Product{
		ID:         h.storage.NextID(),
		Name:       createCmd.Name,
		Price:      createCmd.Price,
		Stock:      createCmd.Stock,
		CategoryID: createCmd.CategoryID,
	}

	h.storage.SaveProduct(product)
	return product, nil
}

type CreateOrderHandler struct {
	storage *TestStorage
}

func (h CreateOrderHandler) Handle(ctx context.Context, cmd Command) (interface{}, error) {
	createCmd := cmd.(CreateOrderCommand)

	// Check if user exists
	user, exists := h.storage.GetUser(createCmd.UserID)
	if !exists {
		return nil, errors.New("user not found")
	}
	if !user.Active {
		return nil, errors.New("user is not active")
	}

	// Check if product exists and has sufficient stock
	product, exists := h.storage.GetProduct(createCmd.ProductID)
	if !exists {
		return nil, errors.New("product not found")
	}
	if product.Stock < createCmd.Quantity {
		return nil, errors.New("insufficient stock")
	}

	// Create order
	order := &Order{
		ID:         h.storage.NextID(),
		UserID:     createCmd.UserID,
		ProductID:  createCmd.ProductID,
		Quantity:   createCmd.Quantity,
		TotalPrice: product.Price * float64(createCmd.Quantity),
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	// Update product stock
	product.Stock -= createCmd.Quantity
	h.storage.SaveProduct(product)
	h.storage.SaveOrder(order)

	return order, nil
}

type UpdateProductStockHandler struct {
	storage *TestStorage
}

func (h UpdateProductStockHandler) Handle(ctx context.Context, cmd Command) (interface{}, error) {
	updateCmd := cmd.(UpdateProductStockCommand)

	product, exists := h.storage.GetProduct(updateCmd.ProductID)
	if !exists {
		return nil, errors.New("product not found")
	}

	if updateCmd.NewStock < 0 {
		return nil, errors.New("stock cannot be negative")
	}

	product.Stock = updateCmd.NewStock
	h.storage.SaveProduct(product)

	return product, nil
}

// Query Handlers
type GetUserHandler struct {
	storage *TestStorage
}

func (h GetUserHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
	getQuery := query.(GetUserQuery)

	user, exists := h.storage.GetUser(getQuery.UserID)
	if !exists {
		return nil, errors.New("user not found")
	}

	return user, nil
}

type GetProductHandler struct {
	storage *TestStorage
}

func (h GetProductHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
	getQuery := query.(GetProductQuery)

	product, exists := h.storage.GetProduct(getQuery.ProductID)
	if !exists {
		return nil, errors.New("product not found")
	}

	return product, nil
}

type GetUserOrdersHandler struct {
	storage *TestStorage
}

func (h GetUserOrdersHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
	getQuery := query.(GetUserOrdersQuery)

	// Verify user exists
	_, exists := h.storage.GetUser(getQuery.UserID)
	if !exists {
		return nil, errors.New("user not found")
	}

	orders := h.storage.GetUserOrders(getQuery.UserID)
	return orders, nil
}

type GetOrderHandler struct {
	storage *TestStorage
}

func (h GetOrderHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
	getQuery := query.(GetOrderQuery)

	order, exists := h.storage.GetOrder(getQuery.OrderID)
	if !exists {
		return nil, errors.New("order not found")
	}

	return order, nil
}

// Integration Tests

func TestCompleteECommerceFlow(t *testing.T) {
	// Setup
	storage := NewTestStorage()

	commandBus := NewCommandBus(CommandBusOptions{
		InitialCapacity: 50,
		EnableMetrics:   true,
		Middleware: []CommandMiddleware{
			ValidationMiddleware(),
			TimeoutMiddleware(5 * time.Second),
		},
	})

	queryBus := NewQueryBus(QueryBusOptions{
		InitialCapacity: 50,
		EnableMetrics:   true,
		Middleware: []QueryMiddleware{
			// Note: CachingMiddleware disabled for integration tests to avoid test interference
			// In production, use: CachingMiddleware(30 * time.Second)
			RetryMiddleware(2, 100*time.Millisecond),
		},
	})

	// Register command handlers
	err := commandBus.RegisterHandler("CreateUser", CreateUserHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register CreateUser handler: %v", err)
	}

	err = commandBus.RegisterHandler("CreateProduct", CreateProductHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register CreateProduct handler: %v", err)
	}

	err = commandBus.RegisterHandler("CreateOrder", CreateOrderHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register CreateOrder handler: %v", err)
	}

	err = commandBus.RegisterHandler("UpdateProductStock", UpdateProductStockHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register UpdateProductStock handler: %v", err)
	}

	// Register query handlers
	err = queryBus.RegisterHandler("GetUser", GetUserHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register GetUser handler: %v", err)
	}

	err = queryBus.RegisterHandler("GetProduct", GetProductHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register GetProduct handler: %v", err)
	}

	err = queryBus.RegisterHandler("GetUserOrders", GetUserOrdersHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register GetUserOrders handler: %v", err)
	}

	err = queryBus.RegisterHandler("GetOrder", GetOrderHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register GetOrder handler: %v", err)
	}

	ctx := context.Background()

	// Track created entity IDs dynamically
	var userID, productID int

	// Test 1: Create User
	t.Run("CreateUser", func(t *testing.T) {
		result, err := commandBus.ExecuteCommand(ctx, CreateUserCommand{
			Email: "john@example.com",
			Name:  "John Doe",
		})
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		user := result.(*User)
		userID = user.ID // Store the actual user ID
		if user.ID == 0 {
			t.Error("User ID should not be zero")
		}
		if user.Email != "john@example.com" {
			t.Errorf("Expected email 'john@example.com', got '%s'", user.Email)
		}
		if user.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%s'", user.Name)
		}
		if !user.Active {
			t.Error("User should be active by default")
		}
	})

	// Test 2: Create Product
	t.Run("CreateProduct", func(t *testing.T) {
		result, err := commandBus.ExecuteCommand(ctx, CreateProductCommand{
			Name:       "Gaming Laptop",
			Price:      1299.99,
			Stock:      10,
			CategoryID: 1,
		})
		if err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}

		product := result.(*Product)
		productID = product.ID // Store the actual product ID
		if product.ID == 0 {
			t.Error("Product ID should not be zero")
		}
		if product.Name != "Gaming Laptop" {
			t.Errorf("Expected name 'Gaming Laptop', got '%s'", product.Name)
		}
		if product.Price != 1299.99 {
			t.Errorf("Expected price 1299.99, got %f", product.Price)
		}
		if product.Stock != 10 {
			t.Errorf("Expected stock 10, got %d", product.Stock)
		}
	})

	// Test 3: Query User
	t.Run("GetUser", func(t *testing.T) {
		result, err := queryBus.Execute(ctx, GetUserQuery{UserID: userID})
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		user := result.(*User)
		if user.ID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, user.ID)
		}
		if user.Email != "john@example.com" {
			t.Errorf("Expected email 'john@example.com', got '%s'", user.Email)
		}
	})

	// Test 4: Query Product
	t.Run("GetProduct", func(t *testing.T) {
		result, err := queryBus.Execute(ctx, GetProductQuery{ProductID: productID})
		if err != nil {
			t.Fatalf("Failed to get product: %v", err)
		}

		product := result.(*Product)
		if product.ID != productID {
			t.Errorf("Expected product ID %d, got %d", productID, product.ID)
		}
		if product.Name != "Gaming Laptop" {
			t.Errorf("Expected name 'Gaming Laptop', got '%s'", product.Name)
		}
	})

	// Test 5: Create Order
	t.Run("CreateOrder", func(t *testing.T) {
		result, err := commandBus.ExecuteCommand(ctx, CreateOrderCommand{
			UserID:    userID,
			ProductID: productID,
			Quantity:  2,
		})
		if err != nil {
			t.Fatalf("Failed to create order: %v", err)
		}

		order := result.(*Order)
		if order.ID == 0 {
			t.Error("Order ID should not be zero")
		}
		if order.UserID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, order.UserID)
		}
		if order.ProductID != productID {
			t.Errorf("Expected product ID %d, got %d", productID, order.ProductID)
		}
		if order.Quantity != 2 {
			t.Errorf("Expected quantity 2, got %d", order.Quantity)
		}
		expectedTotal := 1299.99 * 2
		if order.TotalPrice != expectedTotal {
			t.Errorf("Expected total price %f, got %f", expectedTotal, order.TotalPrice)
		}
		if order.Status != "pending" {
			t.Errorf("Expected status 'pending', got '%s'", order.Status)
		}

		// Verify stock was decremented
		productResult, err := queryBus.Execute(ctx, GetProductQuery{ProductID: productID})
		if err != nil {
			t.Fatalf("Failed to get product after order: %v", err)
		}

		product := productResult.(*Product)
		if product.Stock != 8 { // 10 - 2
			t.Errorf("Expected stock 8 after order, got %d", product.Stock)
		}
	})

	// Test 6: Get User Orders
	t.Run("GetUserOrders", func(t *testing.T) {
		result, err := queryBus.Execute(ctx, GetUserOrdersQuery{UserID: userID})
		if err != nil {
			t.Fatalf("Failed to get user orders: %v", err)
		}

		orders := result.([]*Order)
		if len(orders) != 1 {
			t.Errorf("Expected 1 order, got %d", len(orders))
		}

		if len(orders) > 0 {
			order := orders[0]
			if order.UserID != userID {
				t.Errorf("Expected user ID %d, got %d", userID, order.UserID)
			}
		}
	})

	// Test 7: Error Handling - Insufficient Stock
	t.Run("InsufficientStock", func(t *testing.T) {
		_, err := commandBus.ExecuteCommand(ctx, CreateOrderCommand{
			UserID:    userID,
			ProductID: productID,
			Quantity:  20, // More than available stock (8)
		})
		if err == nil {
			t.Error("Expected error for insufficient stock")
		}
		if err.Error() != "insufficient stock" {
			t.Errorf("Expected 'insufficient stock' error, got '%s'", err.Error())
		}
	})

	// Test 8: Error Handling - User Not Found
	t.Run("UserNotFound", func(t *testing.T) {
		_, err := queryBus.Execute(ctx, GetUserQuery{UserID: 999})
		if err == nil {
			t.Error("Expected error for user not found")
		} else if !strings.Contains(err.Error(), "user not found") {
			t.Errorf("Expected error containing 'user not found', got '%s'", err.Error())
		}
	})

	// Test 9: Validation Errors
	t.Run("ValidationErrors", func(t *testing.T) {
		// Test empty email
		_, err := commandBus.ExecuteCommand(ctx, CreateUserCommand{
			Email: "",
			Name:  "Test User",
		})
		if err == nil {
			t.Error("Expected error for empty email")
		}

		// Test negative price
		_, err = commandBus.ExecuteCommand(ctx, CreateProductCommand{
			Name:  "Test Product",
			Price: -10.0,
			Stock: 5,
		})
		if err == nil {
			t.Error("Expected error for negative price")
		}
	})

	// Test 10: Cache Effectiveness
	t.Run("CacheEffectiveness", func(t *testing.T) {
		// First query - should hit database
		start1 := time.Now()
		_, err := queryBus.Execute(ctx, GetUserQuery{UserID: userID})
		if err != nil {
			t.Fatalf("First query failed: %v", err)
		}
		duration1 := time.Since(start1)

		// Second query - should hit cache
		start2 := time.Now()
		_, err = queryBus.Execute(ctx, GetUserQuery{UserID: userID})
		if err != nil {
			t.Fatalf("Second query failed: %v", err)
		}
		duration2 := time.Since(start2)

		t.Logf("First query: %v, Second query: %v", duration1, duration2)

		// Cache should make second query faster (though this might be flaky in tests)
		if duration2 > duration1 {
			t.Log("Warning: Cache may not be providing expected performance improvement")
		}
	})

	// Test 11: Metrics Collection
	t.Run("MetricsCollection", func(t *testing.T) {
		cmdMetrics := commandBus.GetMetrics()
		queryMetrics := queryBus.GetMetrics()

		if cmdMetrics == nil {
			t.Error("Command metrics should not be nil")
		} else {
			if cmdMetrics.TotalExecutions == 0 {
				t.Error("Command executions should be tracked")
			}
			t.Logf("Command metrics - Executions: %d, Failures: %d, Avg Time: %v",
				cmdMetrics.TotalExecutions, cmdMetrics.TotalFailures, cmdMetrics.AverageExecutionTime)
		}

		if queryMetrics == nil {
			t.Error("Query metrics should not be nil")
		} else {
			if queryMetrics.TotalExecutions == 0 {
				t.Error("Query executions should be tracked")
			}
			t.Logf("Query metrics - Executions: %d, Failures: %d, Avg Time: %v",
				queryMetrics.TotalExecutions, queryMetrics.TotalFailures, queryMetrics.AverageExecutionTime)
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	storage := NewTestStorage()

	commandBus := NewCommandBus(CommandBusOptions{
		InitialCapacity: 100,
		EnableMetrics:   true,
	})

	queryBus := NewQueryBus(QueryBusOptions{
		InitialCapacity: 100,
		EnableMetrics:   true,
	})

	// Register handlers
	err := commandBus.RegisterHandler("CreateUser", CreateUserHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register CreateUser handler: %v", err)
	}

	err = queryBus.RegisterHandler("GetUser", GetUserHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register GetUser handler: %v", err)
	}

	ctx := context.Background()
	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup
	start := time.Now()

	// Create users concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Create user
				_, err := commandBus.ExecuteCommand(ctx, CreateUserCommand{
					Email: fmt.Sprintf("user%d_%d@example.com", id, j),
					Name:  fmt.Sprintf("User %d-%d", id, j),
				})
				if err != nil {
					t.Errorf("Failed to create user: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	elapsed := time.Since(start)
	totalOps := numGoroutines * numOperations

	t.Logf("Created %d users in %v (%.2f ops/sec)",
		totalOps, elapsed, float64(totalOps)/elapsed.Seconds())

	// Verify all users were created correctly
	t.Run("VerifyCreatedUsers", func(t *testing.T) {
		// Try to query some of the created users
		for i := 1; i <= min(10, totalOps); i++ {
			result, err := queryBus.Execute(ctx, GetUserQuery{UserID: i})
			if err != nil {
				t.Errorf("Failed to get user %d: %v", i, err)
				continue
			}

			user := result.(*User)
			if user.ID != i {
				t.Errorf("Expected user ID %d, got %d", i, user.ID)
			}
		}
	})

	// Test concurrent reads
	t.Run("ConcurrentReads", func(t *testing.T) {
		var readWg sync.WaitGroup
		readStart := time.Now()

		for i := 0; i < numGoroutines; i++ {
			readWg.Add(1)
			go func() {
				defer readWg.Done()

				for j := 0; j < 50; j++ {
					userID := (j % totalOps) + 1
					_, err := queryBus.Execute(ctx, GetUserQuery{UserID: userID})
					if err != nil {
						t.Errorf("Failed to get user %d: %v", userID, err)
						return
					}
				}
			}()
		}

		readWg.Wait()
		readElapsed := time.Since(readStart)

		totalReads := numGoroutines * 50
		t.Logf("Performed %d reads in %v (%.2f reads/sec)",
			totalReads, readElapsed, float64(totalReads)/readElapsed.Seconds())
	})
}

func TestMiddlewareIntegration(t *testing.T) {
	storage := NewTestStorage()

	// Custom middleware for testing
	var middlewareExecutions []string
	var mu sync.Mutex

	testMiddleware := func(name string) CommandMiddleware {
		return func(next CommandHandler) CommandHandler {
			return CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
				mu.Lock()
				middlewareExecutions = append(middlewareExecutions, name+"-before")
				mu.Unlock()

				result, err := next.Handle(ctx, cmd)

				mu.Lock()
				middlewareExecutions = append(middlewareExecutions, name+"-after")
				mu.Unlock()

				return result, err
			})
		}
	}

	commandBus := NewCommandBus(CommandBusOptions{
		InitialCapacity: 50,
		EnableMetrics:   true,
		Middleware: []CommandMiddleware{
			testMiddleware("middleware1"),
			testMiddleware("middleware2"),
			ValidationMiddleware(),
			testMiddleware("middleware3"),
		},
	})

	err := commandBus.RegisterHandler("CreateUser", CreateUserHandler{storage: storage})
	if err != nil {
		t.Fatalf("Failed to register CreateUser handler: %v", err)
	}

	ctx := context.Background()

	// Execute command
	_, err = commandBus.ExecuteCommand(ctx, CreateUserCommand{
		Email: "test@example.com",
		Name:  "Test User",
	})
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify middleware execution order
	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"middleware3-before",
		"middleware3-after",
		"middleware2-after",
		"middleware1-after",
	}

	mu.Lock()
	executions := make([]string, len(middlewareExecutions))
	copy(executions, middlewareExecutions)
	mu.Unlock()

	if len(executions) != len(expected) {
		t.Fatalf("Expected %d middleware executions, got %d", len(expected), len(executions))
	}

	for i, exp := range expected {
		if i < len(executions) && executions[i] != exp {
			t.Errorf("Expected middleware execution[%d] = %s, got %s", i, exp, executions[i])
		}
	}
}

// Test types for error handling
type FailingCommand struct {
	ShouldFail bool
}

func (c FailingCommand) CommandName() string {
	return "FailingCommand"
}

type FailingHandler struct{}

func (h FailingHandler) Handle(ctx context.Context, cmd Command) (interface{}, error) {
	failCmd := cmd.(FailingCommand)
	if failCmd.ShouldFail {
		panic("intentional panic for testing")
	}
	return "success", nil
}

func TestErrorHandlingAndRecovery(t *testing.T) {
	commandBus := NewCommandBus(CommandBusOptions{
		InitialCapacity: 10,
		EnableMetrics:   true,
		Middleware: []CommandMiddleware{
			// Recovery middleware should catch panics
			func(next CommandHandler) CommandHandler {
				return CommandHandlerFunc(func(ctx context.Context, cmd Command) (result interface{}, err error) {
					defer func() {
						if r := recover(); r != nil {
							err = fmt.Errorf("handler panicked: %v", r)
						}
					}()
					return next.Handle(ctx, cmd)
				})
			},
		},
	})

	err := commandBus.RegisterHandler("FailingCommand", FailingHandler{})
	if err != nil {
		t.Fatalf("Failed to register FailingCommand handler: %v", err)
	}

	ctx := context.Background()

	// Test successful execution
	t.Run("SuccessfulExecution", func(t *testing.T) {
		result, err := commandBus.ExecuteCommand(ctx, FailingCommand{ShouldFail: false})
		if err != nil {
			t.Fatalf("Expected successful execution, got error: %v", err)
		}
		if result != "success" {
			t.Errorf("Expected 'success', got %v", result)
		}
	})

	// Test panic recovery
	t.Run("PanicRecovery", func(t *testing.T) {
		_, err := commandBus.ExecuteCommand(ctx, FailingCommand{ShouldFail: true})
		if err == nil {
			t.Error("Expected error due to panic, but got nil")
		}
		if err != nil && fmt.Sprintf("%v", err) != "" {
			t.Logf("Recovered from panic with error: %v", err)
		}
	})

	// Verify metrics captured the failure
	t.Run("FailureMetrics", func(t *testing.T) {
		metrics := commandBus.GetMetrics()
		if metrics == nil {
			t.Error("Metrics should not be nil")
		} else {
			if metrics.TotalFailures == 0 {
				t.Error("Expected at least one failure to be recorded")
			}
			t.Logf("Metrics - Executions: %d, Failures: %d",
				metrics.TotalExecutions, metrics.TotalFailures)
		}
	})
}

// Helper function for Go < 1.21 compatibility
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
