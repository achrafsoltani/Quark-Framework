package quark

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// HandlerFunc defines the signature for request handlers.
type HandlerFunc func(*Context) error

// Route represents a single route with its pattern, handler, and middleware.
type Route struct {
	method     string
	pattern    string
	handler    HandlerFunc
	middleware []MiddlewareFunc
	regex      *regexp.Regexp
	paramNames []string
}

// Router is a regex-based HTTP router with path parameters.
type Router struct {
	routes      []*Route
	notFound    HandlerFunc
	methodNotAllowed HandlerFunc
	mu          sync.RWMutex
}

// NewRouter creates a new Router.
func NewRouter() *Router {
	return &Router{
		routes: make([]*Route, 0),
		notFound: func(c *Context) error {
			return c.NotFound("route not found")
		},
		methodNotAllowed: func(c *Context) error {
			return c.Error(http.StatusMethodNotAllowed, "method not allowed")
		},
	}
}

// SetNotFound sets the handler for 404 responses.
func (r *Router) SetNotFound(h HandlerFunc) {
	r.notFound = h
}

// SetMethodNotAllowed sets the handler for 405 responses.
func (r *Router) SetMethodNotAllowed(h HandlerFunc) {
	r.methodNotAllowed = h
}

// Handle registers a new route with the given method and pattern.
// Pattern syntax:
//   - /users           - Exact match
//   - /users/{id}      - Named parameter (matches anything except /)
//   - /users/{id:[0-9]+} - Named parameter with regex constraint
func (r *Router) Handle(method, pattern string, h HandlerFunc, middleware ...MiddlewareFunc) {
	route := &Route{
		method:     method,
		pattern:    pattern,
		handler:    h,
		middleware: middleware,
	}

	// Parse pattern and build regex
	route.regex, route.paramNames = parsePattern(pattern)

	r.mu.Lock()
	r.routes = append(r.routes, route)
	r.mu.Unlock()
}

// parsePattern converts a route pattern to a regex and extracts param names.
func parsePattern(pattern string) (*regexp.Regexp, []string) {
	var paramNames []string
	regexPattern := "^"

	// Handle trailing slash
	pattern = strings.TrimSuffix(pattern, "/")
	if pattern == "" {
		pattern = "/"
	}

	i := 0
	for i < len(pattern) {
		if pattern[i] == '{' {
			// Find closing brace
			end := strings.Index(pattern[i:], "}")
			if end == -1 {
				// Invalid pattern, treat as literal
				regexPattern += regexp.QuoteMeta(string(pattern[i]))
				i++
				continue
			}
			end += i

			// Extract param spec
			paramSpec := pattern[i+1 : end]

			// Check for regex constraint
			var paramName, paramRegex string
			if colonIdx := strings.Index(paramSpec, ":"); colonIdx != -1 {
				paramName = paramSpec[:colonIdx]
				paramRegex = paramSpec[colonIdx+1:]
			} else {
				paramName = paramSpec
				paramRegex = "[^/]+"
			}

			paramNames = append(paramNames, paramName)
			regexPattern += "(" + paramRegex + ")"
			i = end + 1
		} else {
			regexPattern += regexp.QuoteMeta(string(pattern[i]))
			i++
		}
	}

	regexPattern += "/?$"
	return regexp.MustCompile(regexPattern), paramNames
}

// match attempts to match a path against a route.
// Returns the extracted parameters if matched, or nil if not.
func (route *Route) match(path string) map[string]string {
	matches := route.regex.FindStringSubmatch(path)
	if matches == nil {
		return nil
	}

	params := make(map[string]string)
	for i, name := range route.paramNames {
		if i+1 < len(matches) {
			params[name] = matches[i+1]
		}
	}
	return params
}

// find looks up a route for the given method and path.
func (r *Router) find(method, path string) (*Route, map[string]string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var pathMatched bool

	for _, route := range r.routes {
		params := route.match(path)
		if params != nil {
			pathMatched = true
			if route.method == method {
				return route, params, false
			}
		}
	}

	return nil, nil, pathMatched
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// This is a fallback; normally App handles this
	c := newContext(w, req, nil)
	r.handleRequest(c)
}

// handleRequest processes a request through the router.
func (r *Router) handleRequest(c *Context) error {
	route, params, pathMatched := r.find(c.Method(), c.Path())

	if route == nil {
		if pathMatched {
			return r.methodNotAllowed(c)
		}
		return r.notFound(c)
	}

	c.SetParams(params)

	// Apply route-specific middleware
	handler := route.handler
	for i := len(route.middleware) - 1; i >= 0; i-- {
		handler = route.middleware[i](handler)
	}

	return handler(c)
}

// GET registers a GET route.
func (r *Router) GET(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	r.Handle(http.MethodGet, pattern, h, mw...)
}

// POST registers a POST route.
func (r *Router) POST(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	r.Handle(http.MethodPost, pattern, h, mw...)
}

// PUT registers a PUT route.
func (r *Router) PUT(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	r.Handle(http.MethodPut, pattern, h, mw...)
}

// PATCH registers a PATCH route.
func (r *Router) PATCH(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	r.Handle(http.MethodPatch, pattern, h, mw...)
}

// DELETE registers a DELETE route.
func (r *Router) DELETE(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	r.Handle(http.MethodDelete, pattern, h, mw...)
}

// OPTIONS registers an OPTIONS route.
func (r *Router) OPTIONS(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	r.Handle(http.MethodOptions, pattern, h, mw...)
}

// HEAD registers a HEAD route.
func (r *Router) HEAD(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	r.Handle(http.MethodHead, pattern, h, mw...)
}

// Any registers a route for all HTTP methods.
func (r *Router) Any(pattern string, h HandlerFunc, mw ...MiddlewareFunc) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodHead,
	}
	for _, method := range methods {
		r.Handle(method, pattern, h, mw...)
	}
}

// Static serves static files from the given filesystem path.
func (r *Router) Static(prefix, root string) {
	fs := http.FileServer(http.Dir(root))
	handler := http.StripPrefix(prefix, fs)

	r.GET(prefix+"/{filepath:.*}", func(c *Context) error {
		handler.ServeHTTP(c.Writer, c.Request)
		return nil
	})
}

// Routes returns all registered routes (for debugging).
func (r *Router) Routes() []*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]*Route, len(r.routes))
	copy(routes, r.routes)
	return routes
}

// RouteInfo returns route information for debugging.
func (route *Route) RouteInfo() (method, pattern string) {
	return route.method, route.pattern
}
