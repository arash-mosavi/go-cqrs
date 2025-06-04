package main

import (
	"context"
	"fmt"
	"log"

	"github.com/arash-mosavi/go-cqrs/application"
)

type User struct {
	ID    string
	Name  string
	Email string
	Role  string
}

type Product struct {
	ID          string
	Name        string
	Description string
	Price       float64
	Stock       int
}

type Order struct {
	ID        string
	UserID    string
	ProductID string
	Quantity  int
	Total     float64
	Status    string
}

// Commands
type CreateUserCommand struct {
	Name  string
	Email string
	Role  string
}

func (c CreateUserCommand) CommandName() string { return "CreateUser" }

type CreateProductCommand struct {
	Name        string
	Description string
	Price       float64
	Stock       int
}

func (c CreateProductCommand) CommandName() string { return "CreateProduct" }

type CreateOrderCommand struct {
	UserID    string
	ProductID string
	Quantity  int
}

func (c CreateOrderCommand) CommandName() string { return "CreateOrder" }

// Queries
type GetUserQuery struct {
	ID string
}

func (q GetUserQuery) QueryName() string { return "GetUser" }

type GetProductQuery struct {
	ID string
}

func (q GetProductQuery) QueryName() string { return "GetProduct" }

type GetOrderQuery struct {
	ID string
}

func (q GetOrderQuery) QueryName() string { return "GetOrder" }

type ListOrdersQuery struct {
	UserID string
}

func (q ListOrdersQuery) QueryName() string { return "ListOrders" }

// Handlers
type UserHandler struct {
	users map[string]User
}

func NewUserHandler() *UserHandler {
	return &UserHandler{users: make(map[string]User)}
}

func (h *UserHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
	createCmd := cmd.(CreateUserCommand)
	user := User{
		ID:    fmt.Sprintf("user_%d", len(h.users)+1),
		Name:  createCmd.Name,
		Email: createCmd.Email,
		Role:  createCmd.Role,
	}
	h.users[user.ID] = user
	fmt.Printf("User created: %s (ID: %s, Role: %s)\n", user.Name, user.ID, user.Role)
	return user, nil
}

type ProductHandler struct {
	products map[string]Product
}

func NewProductHandler() *ProductHandler {
	return &ProductHandler{products: make(map[string]Product)}
}

func (h *ProductHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
	createCmd := cmd.(CreateProductCommand)
	product := Product{
		ID:          fmt.Sprintf("prod_%d", len(h.products)+1),
		Name:        createCmd.Name,
		Description: createCmd.Description,
		Price:       createCmd.Price,
		Stock:       createCmd.Stock,
	}
	h.products[product.ID] = product
	fmt.Printf("Product created: %s (ID: %s, Price: $%.2f)\n", product.Name, product.ID, product.Price)
	return product, nil
}

type OrderHandler struct {
	orders   map[string]Order
	products map[string]Product
}

func NewOrderHandler(products map[string]Product) *OrderHandler {
	return &OrderHandler{
		orders:   make(map[string]Order),
		products: products,
	}
}

func (h *OrderHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
	createCmd := cmd.(CreateOrderCommand)

	// Check product availability
	product, exists := h.products[createCmd.ProductID]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", createCmd.ProductID)
	}

	if product.Stock < createCmd.Quantity {
		return nil, fmt.Errorf("insufficient stock: need %d, have %d", createCmd.Quantity, product.Stock)
	}

	order := Order{
		ID:        fmt.Sprintf("order_%d", len(h.orders)+1),
		UserID:    createCmd.UserID,
		ProductID: createCmd.ProductID,
		Quantity:  createCmd.Quantity,
		Total:     product.Price * float64(createCmd.Quantity),
		Status:    "pending",
	}

	// Update stock
	product.Stock -= createCmd.Quantity
	h.products[createCmd.ProductID] = product

	h.orders[order.ID] = order
	fmt.Printf("Order created: %s (Total: $%.2f, Status: %s)\n", order.ID, order.Total, order.Status)
	return order, nil
}

// Query Handlers
type UserQueryHandler struct {
	users map[string]User
}

func NewUserQueryHandler(users map[string]User) *UserQueryHandler {
	return &UserQueryHandler{users: users}
}

func (h *UserQueryHandler) Handle(ctx context.Context, query application.Query) (interface{}, error) {
	getQuery := query.(GetUserQuery)
	if user, exists := h.users[getQuery.ID]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found: %s", getQuery.ID)
}

type ProductQueryHandler struct {
	products map[string]Product
}

func NewProductQueryHandler(products map[string]Product) *ProductQueryHandler {
	return &ProductQueryHandler{products: products}
}

func (h *ProductQueryHandler) Handle(ctx context.Context, query application.Query) (interface{}, error) {
	getQuery := query.(GetProductQuery)
	if product, exists := h.products[getQuery.ID]; exists {
		return product, nil
	}
	return nil, fmt.Errorf("product not found: %s", getQuery.ID)
}

type OrderQueryHandler struct {
	orders map[string]Order
}

func NewOrderQueryHandler(orders map[string]Order) *OrderQueryHandler {
	return &OrderQueryHandler{orders: orders}
}

func (h *OrderQueryHandler) Handle(ctx context.Context, query application.Query) (interface{}, error) {
	switch q := query.(type) {
	case GetOrderQuery:
		if order, exists := h.orders[q.ID]; exists {
			return order, nil
		}
		return nil, fmt.Errorf("order not found: %s", q.ID)
	case ListOrdersQuery:
		var userOrders []Order
		for _, order := range h.orders {
			if order.UserID == q.UserID {
				userOrders = append(userOrders, order)
			}
		}
		return userOrders, nil
	}
	return nil, fmt.Errorf("unsupported query type: %T", query)
}

func main() {
	fmt.Println("CQRS ")
	fmt.Println("=================================")

	commandBus := application.NewCommandBus(application.CommandBusOptions{
		InitialCapacity: 64,
		EnableMetrics:   true,
		Middleware: []application.CommandMiddleware{
			application.CommandLoggingMiddleware("COMMANDS"),
			application.ValidationMiddleware(),
		},
	})

	queryBus := application.NewQueryBus(application.QueryBusOptions{
		InitialCapacity: 64,
		EnableMetrics:   true,
		Middleware: []application.QueryMiddleware{
			application.QueryLoggingMiddleware("QUERIES"),
		},
	})

	// Create domain handlers
	userHandler := NewUserHandler()
	productHandler := NewProductHandler()
	orderHandler := NewOrderHandler(productHandler.products)

	userQueryHandler := NewUserQueryHandler(userHandler.users)
	productQueryHandler := NewProductQueryHandler(productHandler.products)
	orderQueryHandler := NewOrderQueryHandler(orderHandler.orders)

	// Register command handlers
	commandBus.RegisterHandler("CreateUser", userHandler)
	commandBus.RegisterHandler("CreateProduct", productHandler)
	commandBus.RegisterHandler("CreateOrder", orderHandler)

	// Register query handlers
	queryBus.RegisterHandler("GetUser", userQueryHandler)
	queryBus.RegisterHandler("GetProduct", productQueryHandler)
	queryBus.RegisterHandler("GetOrder", orderQueryHandler)
	queryBus.RegisterHandler("ListOrders", orderQueryHandler)

	ctx := context.Background()

	fmt.Println("\nScenario: E-commerce Workflow")
	fmt.Println("==========================================")

	fmt.Println("\nCreating users...")
	_, _ = commandBus.ExecuteCommand(ctx, CreateUserCommand{
		Name:  "Admin User",
		Email: "admin@store.com",
		Role:  "admin",
	})

	customer, _ := commandBus.ExecuteCommand(ctx, CreateUserCommand{
		Name:  "John Customer",
		Email: "john@customer.com",
		Role:  "customer",
	})

	fmt.Println("\nCreating products...")
	laptop, _ := commandBus.ExecuteCommand(ctx, CreateProductCommand{
		Name:        "Gaming Laptop",
		Description: "High-performance gaming laptop",
		Price:       1299.99,
		Stock:       5,
	})

	mouse, _ := commandBus.ExecuteCommand(ctx, CreateProductCommand{
		Name:        "Wireless Mouse",
		Description: "Ergonomic wireless mouse",
		Price:       59.99,
		Stock:       20,
	})

	fmt.Println("\nQuerying products...")
	customerUser := customer.(User)
	laptopProduct := laptop.(Product)

	result1, _ := queryBus.Execute(ctx, GetProductQuery{ID: laptopProduct.ID})
	fmt.Printf("First query - Product: %s\n", result1.(Product).Name)

	result2, _ := queryBus.Execute(ctx, GetProductQuery{ID: laptopProduct.ID})
	fmt.Printf("Second query (cached) - Product: %s\n", result2.(Product).Name)

	fmt.Println("\nCreating orders...")
	_, err := commandBus.ExecuteCommand(ctx, CreateOrderCommand{
		UserID:    customerUser.ID,
		ProductID: laptopProduct.ID,
		Quantity:  1,
	})
	if err != nil {
		log.Printf("Order failed: %v", err)
	}

	mouseProduct := mouse.(Product)
	_, err = commandBus.ExecuteCommand(ctx, CreateOrderCommand{
		UserID:    customerUser.ID,
		ProductID: mouseProduct.ID,
		Quantity:  2,
	})
	if err != nil {
		log.Printf("Order failed: %v", err)
	}

	fmt.Println("\nTesting stock validation...")
	_, err = commandBus.ExecuteCommand(ctx, CreateOrderCommand{
		UserID:    customerUser.ID,
		ProductID: laptopProduct.ID,
		Quantity:  10,
	})
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	fmt.Println("\nQuerying user orders...")
	orders, _ := queryBus.Execute(ctx, ListOrdersQuery{UserID: customerUser.ID})
	userOrders := orders.([]Order)
	fmt.Printf("User %s has %d orders:\n", customerUser.Name, len(userOrders))
	for _, order := range userOrders {
		fmt.Printf("   - Order %s: $%.2f (%s)\n", order.ID, order.Total, order.Status)
	}

	fmt.Println("\nPerformance Metrics")
	fmt.Println("======================")

	cmdMetrics := commandBus.GetMetrics()
	if cmdMetrics != nil {
		fmt.Printf("Commands - Total: %d, Failures: %d, Avg Time: %v\n",
			cmdMetrics.TotalExecutions, cmdMetrics.TotalFailures, cmdMetrics.AverageExecutionTime)
	}

	queryMetrics := queryBus.GetMetrics()
	if queryMetrics != nil {
		fmt.Printf("Queries  - Total: %d, Failures: %d, Avg Time: %v\n",
			queryMetrics.TotalExecutions, queryMetrics.TotalFailures, queryMetrics.AverageExecutionTime)
	}

	fmt.Println("\ncompleted successfully!")

}
