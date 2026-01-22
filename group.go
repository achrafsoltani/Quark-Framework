package quark

import (
	"strings"
)

// RouteGroup represents a group of routes with a common prefix and middleware.
type RouteGroup struct {
	prefix     string
	router     *Router
	middleware []MiddlewareFunc
}

// NewRouteGroup creates a new route group.
func NewRouteGroup(router *Router, prefix string, middleware ...MiddlewareFunc) *RouteGroup {
	return &RouteGroup{
		prefix:     strings.TrimSuffix(prefix, "/"),
		router:     router,
		middleware: middleware,
	}
}

// Use adds middleware to the group.
func (g *RouteGroup) Use(mw ...MiddlewareFunc) {
	g.middleware = append(g.middleware, mw...)
}

// Group creates a nested group with additional prefix and middleware.
func (g *RouteGroup) Group(prefix string, mw ...MiddlewareFunc) *RouteGroup {
	combinedMiddleware := make([]MiddlewareFunc, len(g.middleware)+len(mw))
	copy(combinedMiddleware, g.middleware)
	copy(combinedMiddleware[len(g.middleware):], mw)

	return &RouteGroup{
		prefix:     g.prefix + strings.TrimSuffix(prefix, "/"),
		router:     g.router,
		middleware: combinedMiddleware,
	}
}

// handle registers a route with the combined prefix and middleware.
func (g *RouteGroup) handle(method, pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	// Combine group middleware with route middleware
	allMiddleware := make([]MiddlewareFunc, len(g.middleware)+len(mw))
	copy(allMiddleware, g.middleware)
	copy(allMiddleware[len(g.middleware):], mw)

	fullPattern := g.prefix + pattern
	g.router.Handle(method, fullPattern, h, allMiddleware...)
}

// GET registers a GET route.
func (g *RouteGroup) GET(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	g.handle("GET", pattern, h, mw...)
}

// POST registers a POST route.
func (g *RouteGroup) POST(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	g.handle("POST", pattern, h, mw...)
}

// PUT registers a PUT route.
func (g *RouteGroup) PUT(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	g.handle("PUT", pattern, h, mw...)
}

// PATCH registers a PATCH route.
func (g *RouteGroup) PATCH(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	g.handle("PATCH", pattern, h, mw...)
}

// DELETE registers a DELETE route.
func (g *RouteGroup) DELETE(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	g.handle("DELETE", pattern, h, mw...)
}

// OPTIONS registers an OPTIONS route.
func (g *RouteGroup) OPTIONS(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	g.handle("OPTIONS", pattern, h, mw...)
}

// HEAD registers a HEAD route.
func (g *RouteGroup) HEAD(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	g.handle("HEAD", pattern, h, mw...)
}

// Any registers a route for all HTTP methods.
func (g *RouteGroup) Any(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}
	for _, method := range methods {
		g.handle(method, pattern, h, mw...)
	}
}

// Static serves static files from the given filesystem path.
func (g *RouteGroup) Static(relativePath, root string) {
	g.router.Static(g.prefix+relativePath, root)
}

// Prefix returns the group's prefix.
func (g *RouteGroup) Prefix() string {
	return g.prefix
}
