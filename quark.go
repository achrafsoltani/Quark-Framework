// Package quark provides a lightweight, zero-dependency HTTP micro-framework for Go.
//
// Quark combines PHP Quark patterns (convention-based routing, DI container, service providers)
// with Go idioms and patterns from production APIs. It uses only the standard library.
//
// Example usage:
//
//	app := quark.New()
//	app.Use(middleware.Logger())
//	app.Use(middleware.Recovery())
//
//	app.GET("/health", func(c *quark.Context) error {
//	    return c.JSON(200, quark.M{"status": "ok"})
//	})
//
//	api := app.Group("/api/v1")
//	api.GET("/users/{id}", getUser)
//
//	app.Run(":8080")
package quark

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Version is the current version of Quark.
const Version = "0.1.0"

// App is the main application instance.
type App struct {
	router      *Router
	container   *Container
	config      *Config
	middleware  []MiddlewareFunc
	onStart     []func(*App) error
	onShutdown  []func(*App) error
	server      *http.Server
	contextPool sync.Pool
	debug       bool
	logger      Logger
}

// Logger interface for application logging.
type Logger interface {
	Printf(format string, v ...interface{})
}

// Option is a function that configures the App.
type Option func(*App)

// New creates a new Quark application.
func New(opts ...Option) *App {
	app := &App{
		router:     NewRouter(),
		container:  NewContainer(),
		config:     DefaultConfig(),
		middleware: make([]MiddlewareFunc, 0),
		onStart:    make([]func(*App) error, 0),
		onShutdown: make([]func(*App) error, 0),
		debug:      false,
		logger:     log.New(os.Stdout, "[quark] ", log.LstdFlags),
	}

	app.contextPool = sync.Pool{
		New: func() interface{} {
			return &Context{
				params: make(map[string]string),
				store:  make(map[string]interface{}),
				app:    app,
			}
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(app)
	}

	return app
}

// WithDebug enables debug mode.
func WithDebug(debug bool) Option {
	return func(a *App) {
		a.debug = debug
	}
}

// WithLogger sets a custom logger.
func WithLogger(l Logger) Option {
	return func(a *App) {
		a.logger = l
	}
}

// WithConfig sets the application configuration.
func WithConfig(cfg *Config) Option {
	return func(a *App) {
		a.config = cfg
	}
}

// Router returns the application router.
func (a *App) Router() *Router {
	return a.router
}

// Container returns the DI container.
func (a *App) Container() *Container {
	return a.container
}

// Config returns the application configuration.
func (a *App) Config() *Config {
	return a.config
}

// Debug returns whether debug mode is enabled.
func (a *App) Debug() bool {
	return a.debug
}

// Logger returns the application logger.
func (a *App) Logger() Logger {
	return a.logger
}

// Use adds middleware to the global middleware stack.
func (a *App) Use(mw ...MiddlewareFunc) {
	a.middleware = append(a.middleware, mw...)
}

// OnStart registers a callback to run when the app starts.
func (a *App) OnStart(fn func(*App) error) {
	a.onStart = append(a.onStart, fn)
}

// OnShutdown registers a callback to run when the app shuts down.
func (a *App) OnShutdown(fn func(*App) error) {
	a.onShutdown = append(a.onShutdown, fn)
}

// GET registers a GET route.
func (a *App) GET(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	a.router.GET(pattern, h, mw...)
}

// POST registers a POST route.
func (a *App) POST(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	a.router.POST(pattern, h, mw...)
}

// PUT registers a PUT route.
func (a *App) PUT(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	a.router.PUT(pattern, h, mw...)
}

// PATCH registers a PATCH route.
func (a *App) PATCH(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	a.router.PATCH(pattern, h, mw...)
}

// DELETE registers a DELETE route.
func (a *App) DELETE(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	a.router.DELETE(pattern, h, mw...)
}

// OPTIONS registers an OPTIONS route.
func (a *App) OPTIONS(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	a.router.OPTIONS(pattern, h, mw...)
}

// HEAD registers a HEAD route.
func (a *App) HEAD(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	a.router.HEAD(pattern, h, mw...)
}

// Any registers a route for all HTTP methods.
func (a *App) Any(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	a.router.Any(pattern, h, mw...)
}

// Static serves static files from the given filesystem path.
func (a *App) Static(prefix, root string) {
	a.router.Static(prefix, root)
}

// Group creates a new route group with the given prefix.
func (a *App) Group(prefix string, mw ...MiddlewareFunc) *RouteGroup {
	return NewRouteGroup(a.router, prefix, mw...)
}

// ServeHTTP implements the http.Handler interface.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get context from pool
	c := a.contextPool.Get().(*Context)
	c.reset(w, r)
	c.app = a

	// Build the handler chain with global middleware
	handler := a.router.handleRequest
	for i := len(a.middleware) - 1; i >= 0; i-- {
		mw := a.middleware[i]
		next := handler
		handler = func(ctx *Context) error {
			return mw(func(c *Context) error {
				return next(c)
			})(ctx)
		}
	}

	// Execute the handler
	if err := handler(c); err != nil {
		a.handleError(c, err)
	}

	// Return context to pool
	a.contextPool.Put(c)
}

// handleError handles errors returned from handlers.
func (a *App) handleError(c *Context, err error) {
	if c.IsWritten() {
		return
	}

	if httpErr, ok := err.(*HTTPError); ok {
		if a.debug && httpErr.Err != nil {
			c.JSON(httpErr.Code, M{
				"error": M{
					"code":    httpErr.Code,
					"message": httpErr.Message,
					"debug":   httpErr.Err.Error(),
				},
			})
		} else {
			c.Error(httpErr.Code, httpErr.Message)
		}
		return
	}

	// Generic error
	if a.debug {
		c.JSON(http.StatusInternalServerError, M{
			"error": M{
				"code":    http.StatusInternalServerError,
				"message": "Internal Server Error",
				"debug":   err.Error(),
			},
		})
	} else {
		c.InternalError("")
	}
}

// Run starts the HTTP server on the given address.
func (a *App) Run(addr string) error {
	if addr == "" {
		addr = fmt.Sprintf("%s:%s", a.config.Host, a.config.Port)
	}

	// Run onStart callbacks
	for _, fn := range a.onStart {
		if err := fn(a); err != nil {
			return fmt.Errorf("onStart callback failed: %w", err)
		}
	}

	a.server = &http.Server{
		Addr:         addr,
		Handler:      a,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
		IdleTimeout:  a.config.IdleTimeout,
	}

	a.logger.Printf("Starting server on %s", addr)

	return a.server.ListenAndServe()
}

// RunTLS starts the HTTPS server on the given address.
func (a *App) RunTLS(addr, certFile, keyFile string) error {
	if addr == "" {
		addr = fmt.Sprintf("%s:%s", a.config.Host, a.config.Port)
	}

	// Run onStart callbacks
	for _, fn := range a.onStart {
		if err := fn(a); err != nil {
			return fmt.Errorf("onStart callback failed: %w", err)
		}
	}

	a.server = &http.Server{
		Addr:         addr,
		Handler:      a,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
		IdleTimeout:  a.config.IdleTimeout,
	}

	a.logger.Printf("Starting TLS server on %s", addr)

	return a.server.ListenAndServeTLS(certFile, keyFile)
}

// RunWithGracefulShutdown starts the server with graceful shutdown on SIGINT/SIGTERM.
func (a *App) RunWithGracefulShutdown(addr string) error {
	if addr == "" {
		addr = fmt.Sprintf("%s:%s", a.config.Host, a.config.Port)
	}

	// Run onStart callbacks
	for _, fn := range a.onStart {
		if err := fn(a); err != nil {
			return fmt.Errorf("onStart callback failed: %w", err)
		}
	}

	a.server = &http.Server{
		Addr:         addr,
		Handler:      a,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
		IdleTimeout:  a.config.IdleTimeout,
	}

	// Channel to listen for errors from ListenAndServe
	serverErrors := make(chan error, 1)

	// Start the server
	go func() {
		a.logger.Printf("Starting server on %s", addr)
		serverErrors <- a.server.ListenAndServe()
	}()

	// Channel to listen for OS signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		a.logger.Printf("Received signal %v, starting graceful shutdown...", sig)

		// Create a context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()

		// Run onShutdown callbacks
		for _, fn := range a.onShutdown {
			if err := fn(a); err != nil {
				a.logger.Printf("onShutdown callback failed: %v", err)
			}
		}

		// Gracefully shutdown the server
		if err := a.server.Shutdown(ctx); err != nil {
			a.logger.Printf("Graceful shutdown failed: %v", err)
			return a.server.Close()
		}

		a.logger.Printf("Server stopped gracefully")
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (a *App) Shutdown(ctx context.Context) error {
	// Run onShutdown callbacks
	for _, fn := range a.onShutdown {
		if err := fn(a); err != nil {
			a.logger.Printf("onShutdown callback failed: %v", err)
		}
	}

	if a.server != nil {
		return a.server.Shutdown(ctx)
	}
	return nil
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Port:            "8080",
		Host:            "0.0.0.0",
		Environment:     "development",
		Debug:           false,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}
