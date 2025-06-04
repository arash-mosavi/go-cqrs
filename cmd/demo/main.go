package main

import (
	"context"
	"fmt"
	"log"

	"github.com/arash-mosavi/go-cqrs/application"
)

// User represents a user in our system
type User struct {
	ID    string
	Name  string
	Email string
}

// CreateUserCommand implements the Command interface
type CreateUserCommand struct {
	Name  string
	Email string
}

func (c CreateUserCommand) CommandName() string {
	return "CreateUser"
}

// GetUserQuery implements the Query interface
type GetUserQuery struct {
	ID string
}

func (q GetUserQuery) QueryName() string {
	return "GetUser"
}

// UserCommandHandler handles user creation - demonstrates Single Responsibility Principle
type UserCommandHandler struct {
	users map[string]User
}

func NewUserCommandHandler() *UserCommandHandler {
	return &UserCommandHandler{
		users: make(map[string]User),
	}
}

func (h *UserCommandHandler) Handle(ctx context.Context, cmd application.Command) (interface{}, error) {
	createCmd := cmd.(CreateUserCommand)

	user := User{
		ID:    fmt.Sprintf("user_%d", len(h.users)+1),
		Name:  createCmd.Name,
		Email: createCmd.Email,
	}

	h.users[user.ID] = user
	fmt.Printf("User created: %s (ID: %s)\n", user.Name, user.ID)

	return user, nil
}

// UserQueryHandler handles user queries - demonstrates Separation of Concerns
type UserQueryHandler struct {
	users map[string]User
}

func NewUserQueryHandler(users map[string]User) *UserQueryHandler {
	return &UserQueryHandler{users: users}
}

func (h *UserQueryHandler) Handle(ctx context.Context, query application.Query) (interface{}, error) {
	getQuery := query.(GetUserQuery)

	if user, exists := h.users[getQuery.ID]; exists {
		fmt.Printf("User found: %s (%s)\n", user.Name, user.Email)
		return user, nil
	}

	return nil, fmt.Errorf("user not found: %s", getQuery.ID)
}

func main() {
	fmt.Println("CQRS Simple Demo - SOLID Principles Showcase")
	fmt.Println("==============================================")

	commandBus := application.NewCommandBus()
	queryBus := application.NewQueryBus()

	userCommandHandler := NewUserCommandHandler()
	userQueryHandler := NewUserQueryHandler(userCommandHandler.users)

	commandBus.RegisterHandler("CreateUser", userCommandHandler)
	queryBus.RegisterHandler("GetUser", userQueryHandler)

	ctx := context.Background()

	fmt.Println("\nCreating users...")
	users := []CreateUserCommand{
		{Name: "Alice Johnson", Email: "alice@example.com"},
		{Name: "Bob Smith", Email: "bob@example.com"},
		{Name: "Carol Davis", Email: "carol@example.com"},
	}

	var userIDs []string
	for _, userCmd := range users {
		result, err := commandBus.ExecuteCommand(ctx, userCmd)
		if err != nil {
			log.Printf("Error creating user: %v", err)
			continue
		}

		user := result.(User)
		userIDs = append(userIDs, user.ID)
	}

	fmt.Println("\nQuerying users...")
	for _, userID := range userIDs {
		result, err := queryBus.Execute(ctx, GetUserQuery{ID: userID})
		if err != nil {
			log.Printf("Error querying user %s: %v", userID, err)
			continue
		}

		user := result.(User)
		fmt.Printf("Retrieved: %s <%s>\n", user.Name, user.Email)
	}

	fmt.Println("\nQuerying non-existent user...")
	_, err := queryBus.Execute(ctx, GetUserQuery{ID: "nonexistent"})
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	fmt.Println("\ncompleted successfully!")
}
