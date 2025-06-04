package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arash-mosavi/go-cqrs/application"
)

// Domain Models
type Post struct {
	ID       string
	Title    string
	Content  string
	AuthorID string
	Created  time.Time
}

// Commands
type CreatePostCommand struct {
	Title    string `validate:"required"`
	Content  string `validate:"required"`
	AuthorID string `validate:"required"`
}

func (c CreatePostCommand) CommandName() string { return "CreatePost" }

// Command Handlers
type PostCommandHandler struct {
	posts map[string]Post
}

func NewPostCommandHandler() *PostCommandHandler {
	return &PostCommandHandler{
		posts: make(map[string]Post),
	}
}

func (h *PostCommandHandler) Handle(ctx context.Context, command application.Command) (interface{}, error) {
	switch cmd := command.(type) {
	case CreatePostCommand:
		post := Post{
			ID:       fmt.Sprintf("post_%d", len(h.posts)+1),
			Title:    cmd.Title,
			Content:  cmd.Content,
			AuthorID: cmd.AuthorID,
			Created:  time.Now(),
		}
		h.posts[post.ID] = post
		fmt.Printf("Created post: '%s' by author %s\n", post.Title, post.AuthorID)
		return post, nil
	default:
		return nil, fmt.Errorf("unknown command: %T", command)
	}
}

// Queries
type GetPostQuery struct {
	ID string `validate:"required"`
}

func (q GetPostQuery) QueryName() string { return "GetPost" }

type ListPostsQuery struct {
	AuthorID string
}

func (q ListPostsQuery) QueryName() string { return "ListPosts" }

// Query Handlers
type PostQueryHandler struct {
	posts map[string]Post
}

func NewPostQueryHandler(posts map[string]Post) *PostQueryHandler {
	return &PostQueryHandler{posts: posts}
}

func (h *PostQueryHandler) Handle(ctx context.Context, query application.Query) (interface{}, error) {
	switch q := query.(type) {
	case GetPostQuery:
		if post, exists := h.posts[q.ID]; exists {
			return post, nil
		}
		return nil, fmt.Errorf("post not found: %s", q.ID)
	case ListPostsQuery:
		var posts []Post
		for _, post := range h.posts {
			if q.AuthorID == "" || post.AuthorID == q.AuthorID {
				posts = append(posts, post)
			}
		}
		return posts, nil
	default:
		return nil, fmt.Errorf("unknown query: %T", query)
	}
}

func main() {
	fmt.Println("Blog System CQRS Demo")
	fmt.Println("===================")

	// Initialize CQRS components
	commandBus := application.NewCommandBus()
	queryBus := application.NewQueryBus()

	// Create handlers with shared state
	commandHandler := NewPostCommandHandler()
	queryHandler := NewPostQueryHandler(commandHandler.posts)

	// Register handlers
	commandBus.RegisterHandler("CreatePost", commandHandler)
	queryBus.RegisterHandler("GetPost", queryHandler)
	queryBus.RegisterHandler("ListPosts", queryHandler)

	ctx := context.Background()

	// Create posts
	fmt.Println("\nCreating posts...")

	// Create first post
	post1, err := commandBus.ExecuteCommand(ctx, CreatePostCommand{
		Title:    "CQRS",
		Content:  "...",
		AuthorID: "author_1",
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = commandBus.ExecuteCommand(ctx, CreatePostCommand{
		Title:    "Advanced",
		Content:  "...",
		AuthorID: "author_1",
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = commandBus.ExecuteCommand(ctx, CreatePostCommand{
		Title:    "testing system",
		Content:  "...",
		AuthorID: "author_2",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nQuerying posts...")

	p1 := post1.(Post)
	result, err := queryBus.Execute(ctx, GetPostQuery{ID: p1.ID})
	if err != nil {
		log.Fatal(err)
	}
	retrievedPost := result.(Post)
	fmt.Printf("Retrieved post: %s\n", retrievedPost.Title)

	result, err = queryBus.Execute(ctx, ListPostsQuery{})
	if err != nil {
		log.Fatal(err)
	}
	allPosts := result.([]Post)
	fmt.Printf("Total posts: %d\n", len(allPosts))

	result, err = queryBus.Execute(ctx, ListPostsQuery{AuthorID: "author_1"})
	if err != nil {
		log.Fatal(err)
	}
	authorPosts := result.([]Post)
	fmt.Printf("Posts by author_1: %d\n", len(authorPosts))

	for _, post := range authorPosts {
		fmt.Printf("  - %s\n", post.Title)
	}

	fmt.Println("\nBlog system demo completed!")
}
