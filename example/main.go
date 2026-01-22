// Example application demonstrating Quark framework usage.
package main

import (
	"log"
	"time"

	"github.com/AchrafSoltani/quark"
	"github.com/AchrafSoltani/quark/contrib/jwt"
	"github.com/AchrafSoltani/quark/middleware"
)

// User represents a user entity.
type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name" validate:"required,min:2,max:50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"min:0,max:150"`
	Username string `json:"username" validate:"required,alphanum,min:3,max:20"`
}

// In-memory user storage for demo.
var users = []User{
	{ID: 1, Name: "John Doe", Email: "john@example.com", Age: 30, Username: "johndoe"},
	{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Age: 25, Username: "janesmith"},
}
var nextID int64 = 3

// JWT secret (in production, use environment variable)
var jwtSecret = []byte("your-secret-key-change-in-production")

func main() {
	// Create a new Quark application
	app := quark.New(
		quark.WithDebug(true),
	)

	// Register global middleware
	app.Use(middleware.Recovery())
	app.Use(middleware.Logger())
	app.Use(middleware.CORS(middleware.DefaultCORSConfig))

	// Health check endpoint
	app.GET("/health", healthHandler)

	// Public routes
	app.POST("/auth/login", loginHandler)

	// API routes (protected)
	api := app.Group("/api/v1")

	// JWT middleware for protected routes
	jwtHandler := jwt.NewWithSecret(jwtSecret)
	api.Use(jwt.Middleware(jwtHandler))

	// User routes
	api.GET("/users", listUsers)
	api.POST("/users", createUser)
	api.GET("/users/{id:[0-9]+}", getUser)
	api.PUT("/users/{id:[0-9]+}", updateUser)
	api.DELETE("/users/{id:[0-9]+}", deleteUser)

	// Start server with graceful shutdown
	log.Println("Starting server on :8080")
	if err := app.RunWithGracefulShutdown(":8080"); err != nil {
		log.Fatal(err)
	}
}

// healthHandler returns the server health status.
func healthHandler(c *quark.Context) error {
	return c.JSON(200, quark.M{
		"status":  "ok",
		"version": quark.Version,
	})
}

// loginHandler handles user login and returns a JWT token.
func loginHandler(c *quark.Context) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.Bind(&input); err != nil {
		return err
	}

	// Simple demo validation (in production, verify against database)
	if input.Username != "demo" || input.Password != "password" {
		return c.Unauthorized("invalid credentials")
	}

	// Create JWT token
	jwtHandler := jwt.New(jwt.Config{
		Secret:    jwtSecret,
		ExpiresIn: 24 * time.Hour,
	})

	claims := jwt.NewClaims(input.Username, 24*time.Hour).
		WithCustom("user_id", 1).
		WithCustom("roles", []string{"user", "admin"})

	token, err := jwtHandler.Generate(claims)
	if err != nil {
		return c.InternalError("failed to generate token")
	}

	return c.JSON(200, quark.M{
		"token":      token,
		"expires_in": 24 * 60 * 60, // seconds
	})
}

// listUsers returns a paginated list of users.
func listUsers(c *quark.Context) error {
	p := c.Pagination(20, 100)

	// Calculate slice bounds
	start := p.Offset
	if start > len(users) {
		start = len(users)
	}
	end := start + p.PerPage
	if end > len(users) {
		end = len(users)
	}

	return c.JSONPaginated(users[start:end], p.Page, p.PerPage, len(users))
}

// createUser creates a new user.
func createUser(c *quark.Context) error {
	var input User
	if err := c.Bind(&input); err != nil {
		return err
	}

	// Validate input
	if errs := quark.Validate(input); errs.HasErrors() {
		return c.JSON(400, quark.M{
			"error":  "validation failed",
			"errors": errs.ToMap(),
		})
	}

	// Create user
	input.ID = nextID
	nextID++
	users = append(users, input)

	return c.Created(input)
}

// getUser returns a user by ID.
func getUser(c *quark.Context) error {
	id, err := c.ParamInt("id")
	if err != nil {
		return c.BadRequest("invalid user ID")
	}

	for _, user := range users {
		if user.ID == id {
			return c.JSON(200, user)
		}
	}

	return c.NotFound("user not found")
}

// updateUser updates an existing user.
func updateUser(c *quark.Context) error {
	id, err := c.ParamInt("id")
	if err != nil {
		return c.BadRequest("invalid user ID")
	}

	var input User
	if err := c.Bind(&input); err != nil {
		return err
	}

	// Validate input
	if errs := quark.Validate(input); errs.HasErrors() {
		return c.JSON(400, quark.M{
			"error":  "validation failed",
			"errors": errs.ToMap(),
		})
	}

	// Find and update user
	for i, user := range users {
		if user.ID == id {
			input.ID = id
			users[i] = input
			return c.JSON(200, input)
		}
	}

	return c.NotFound("user not found")
}

// deleteUser deletes a user by ID.
func deleteUser(c *quark.Context) error {
	id, err := c.ParamInt("id")
	if err != nil {
		return c.BadRequest("invalid user ID")
	}

	for i, user := range users {
		if user.ID == id {
			users = append(users[:i], users[i+1:]...)
			return c.NoContent()
		}
	}

	return c.NotFound("user not found")
}
