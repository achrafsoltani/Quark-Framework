# Quark Framework

A lightweight, zero-dependency HTTP micro-framework for Go.

Quark combines PHP Quark patterns (convention-based routing, DI container, service providers) with Go idioms and patterns from production APIs. It uses only the standard library for core functionality.

## Features

- **Zero Dependencies**: Core uses only Go standard library
- **Type-Safe DI Container**: Generics-based dependency injection
- **Regex-Based Router**: Path parameters with regex constraints
- **Middleware System**: Composable middleware with route-level support
- **Struct Validation**: Tag-based validation for request data
- **Built-in Middleware**: CORS, Logger, Recovery, Auth
- **Optional Modules**: Database helpers, JWT, HTML templates

## Installation

```bash
go get github.com/AchrafSoltani/quark
```

## Quick Start

```go
package main

import (
    "github.com/AchrafSoltani/quark"
    "github.com/AchrafSoltani/quark/middleware"
)

func main() {
    app := quark.New()

    // Global middleware
    app.Use(middleware.Recovery())
    app.Use(middleware.Logger())
    app.Use(middleware.CORS(middleware.DefaultCORSConfig))

    // Routes
    app.GET("/health", func(c *quark.Context) error {
        return c.JSON(200, quark.M{"status": "ok"})
    })

    // Route groups
    api := app.Group("/api/v1")
    api.GET("/users", listUsers)
    api.POST("/users", createUser)
    api.GET("/users/{id:[0-9]+}", getUser)

    app.Run(":8080")
}

func listUsers(c *quark.Context) error {
    p := c.Pagination(20, 100)
    // Fetch users with pagination...
    return c.JSONPaginated(users, p.Page, p.PerPage, total)
}

func createUser(c *quark.Context) error {
    var input User
    if err := c.Bind(&input); err != nil {
        return err
    }
    if errs := quark.Validate(input); len(errs) > 0 {
        return c.JSON(400, quark.M{"errors": errs})
    }
    // Save user...
    return c.Created(user)
}

func getUser(c *quark.Context) error {
    id, _ := c.ParamInt("id")
    // Fetch user...
    return c.JSON(200, user)
}
```

## Core Concepts

### Application

```go
app := quark.New(
    quark.WithDebug(true),
    quark.WithLogger(customLogger),
)

// Lifecycle hooks
app.OnStart(func(a *quark.App) error {
    // Initialize resources
    return nil
})

app.OnShutdown(func(a *quark.App) error {
    // Cleanup resources
    return nil
})

// Start with graceful shutdown
app.RunWithGracefulShutdown(":8080")
```

### Routing

```go
// Path parameters
app.GET("/users/{id}", getUser)
app.GET("/users/{id:[0-9]+}", getUserById)  // With regex constraint
app.GET("/files/{path:.*}", serveFile)      // Catch-all

// Route groups
api := app.Group("/api/v1", authMiddleware)
api.GET("/users", listUsers)
api.POST("/users", createUser)

// Nested groups
admin := api.Group("/admin", adminMiddleware)
admin.GET("/stats", getStats)
```

### Context

```go
func handler(c *quark.Context) error {
    // Path parameters
    id := c.Param("id")
    idInt, _ := c.ParamInt("id")

    // Query parameters
    search := c.Query("search")
    page := c.QueryInt("page", 1)
    active := c.QueryBool("active")

    // Request body
    var input struct {
        Name string `json:"name"`
    }
    c.Bind(&input)

    // Headers
    auth := c.Header("Authorization")
    c.SetHeader("X-Custom", "value")

    // Context store
    c.Set("user", user)
    user := c.Get("user")

    // Client info
    ip := c.RealIP()
    method := c.Method()
    path := c.Path()

    return c.JSON(200, data)
}
```

### Responses

```go
// JSON
c.JSON(200, data)
c.JSONPretty(200, data, "  ")
c.JSONPaginated(items, page, perPage, total)

// Other formats
c.String(200, "Hello")
c.HTML(200, "<h1>Hello</h1>")
c.Blob(200, "image/png", imageData)

// Status helpers
c.NoContent()           // 204
c.Created(data)         // 201
c.Redirect(302, url)

// Error responses
c.Error(500, "Something went wrong")
c.BadRequest("Invalid input")
c.Unauthorized("Please login")
c.Forbidden("Access denied")
c.NotFound("Resource not found")
```

### Middleware

```go
// Custom middleware
func Timer() quark.MiddlewareFunc {
    return func(next quark.HandlerFunc) quark.HandlerFunc {
        return func(c *quark.Context) error {
            start := time.Now()
            err := next(c)
            duration := time.Since(start)
            c.SetHeader("X-Response-Time", duration.String())
            return err
        }
    }
}

// Built-in middleware
app.Use(middleware.Recovery())
app.Use(middleware.Logger())
app.Use(middleware.CORS(middleware.DefaultCORSConfig))
app.Use(middleware.Auth(tokenValidator))

// Route-level middleware
app.GET("/admin", adminHandler, adminMiddleware)
```

### DI Container

```go
// Register services
quark.Provide(app.Container(), "db", func(c *quark.Container) (*sql.DB, error) {
    return sql.Open("postgres", dsn)
})

quark.ProvideValue(app.Container(), "config", config)

// Resolve services
db, err := quark.Resolve[*sql.DB](app.Container(), "db")
db := quark.MustResolve[*sql.DB](app.Container(), "db")

// Service providers
type DatabaseProvider struct {
    quark.BaseProvider
}

func (p *DatabaseProvider) Register(c *quark.Container) error {
    quark.Provide(c, "db", createDB)
    return nil
}

app.Container().RegisterProviders(&DatabaseProvider{})
```

### Validation

```go
type User struct {
    Name     string `json:"name" validate:"required,min:2,max:50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min:0,max:150"`
    Role     string `json:"role" validate:"oneof:admin user guest"`
    Username string `json:"username" validate:"required,alphanum"`
}

func createUser(c *quark.Context) error {
    var input User
    if err := c.Bind(&input); err != nil {
        return err
    }

    if errs := quark.Validate(input); errs.HasErrors() {
        return c.JSON(400, quark.M{
            "error":  "validation failed",
            "errors": errs.ToMap(),
        })
    }

    // Create user...
}
```

Supported validation tags:
- `required` - Field must not be empty
- `min:n` - Minimum length/value
- `max:n` - Maximum length/value
- `len:n` - Exact length
- `email` - Valid email format
- `url` - Valid URL format
- `alpha` - Letters only
- `alphanum` - Letters and numbers only
- `numeric` - Numbers only
- `uuid` - Valid UUID
- `oneof:a b c` - Must be one of values
- `pattern:regex` - Must match regex
- `gt:n`, `gte:n`, `lt:n`, `lte:n` - Numeric comparisons

### Configuration

```go
type AppConfig struct {
    Port        string        `env:"PORT" default:"8080"`
    DatabaseURL string        `env:"DATABASE_URL"`
    Debug       bool          `env:"DEBUG" default:"false"`
    Timeout     time.Duration `env:"TIMEOUT" default:"30s"`
}

cfg := &AppConfig{}
quark.LoadFromEnv(cfg)

// Or use helpers
port := quark.Env("PORT", "8080")
debug := quark.EnvBool("DEBUG", false)
timeout := quark.EnvDuration("TIMEOUT", 30*time.Second)
```

## Optional Modules

### JWT Authentication

```go
import "github.com/AchrafSoltani/quark/contrib/jwt"

// Create JWT handler
jwtHandler := jwt.New(jwt.Config{
    Secret:    []byte("your-secret"),
    ExpiresIn: 24 * time.Hour,
})

// Generate token
claims := jwt.NewClaims("user123", 24*time.Hour).
    WithCustom("user_id", 123).
    WithCustom("roles", []string{"admin"})
token, _ := jwtHandler.Generate(claims)

// Middleware
app.Use(jwt.Middleware(jwtHandler))

// Access claims in handler
claims := jwt.GetClaims(c)
userID := claims.GetInt64("user_id")
```

### Database Helpers

```go
import "github.com/AchrafSoltani/quark/contrib/database"

// Open connection
db, _ := database.Open(database.Config{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     5432,
    Database: "myapp",
    Username: "user",
    Password: "pass",
})

// Transactions
db.WithTx(ctx, func(tx *database.Tx) error {
    // Queries within transaction
    return nil
})

// Pagination
page, _ := database.PaginateQuery(ctx, db,
    "SELECT * FROM users",
    scanUser,
    database.NewPaginationParams(1, 20, 20, 100),
    "active = $1", true,
)
```

### HTML Templates

```go
import "github.com/AchrafSoltani/quark/contrib/template"

engine, _ := template.New(template.Config{
    Dir:       "templates",
    Extension: ".html",
    Reload:    true, // For development
})

func handler(c *quark.Context) error {
    return engine.HTML(c, 200, "users/index", quark.M{
        "users": users,
    })
}
```

## Project Structure

```
quark-framework/
├── quark.go              # Application, lifecycle, route shortcuts
├── router.go             # HTTP router with path parameters
├── context.go            # Request context with helpers
├── response.go           # JSON, HTML, error responses
├── middleware.go         # Middleware types and composition
├── container.go          # DI container with generics
├── config.go             # Environment-based configuration
├── errors.go             # HTTP error types
├── group.go              # Route grouping
├── validator.go          # Struct validation
│
├── middleware/           # Built-in middleware
│   ├── cors.go
│   ├── logger.go
│   ├── recovery.go
│   └── auth.go
│
└── contrib/              # Optional modules
    ├── database/         # database/sql helpers
    ├── jwt/              # JWT without external deps
    └── template/         # html/template helpers
```

## License

MIT License
