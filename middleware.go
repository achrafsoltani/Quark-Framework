// Package quark provides a lightweight, zero-dependency HTTP micro-framework for Go.
package quark

// MiddlewareFunc defines the signature for middleware functions.
// Middleware wraps a HandlerFunc and returns a new HandlerFunc.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Chain composes multiple middleware into a single middleware.
// Middleware are applied in the order they are passed (first to last).
func Chain(middleware ...MiddlewareFunc) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i](next)
		}
		return next
	}
}

// WrapMiddleware wraps a list of middleware around a handler.
func WrapMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
