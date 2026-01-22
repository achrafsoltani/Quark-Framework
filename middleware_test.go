package quark

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareChain(t *testing.T) {
	order := []string{}

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "mw1-before")
			err := next(c)
			order = append(order, "mw1-after")
			return err
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "mw2-before")
			err := next(c)
			order = append(order, "mw2-after")
			return err
		}
	}

	handler := func(c *Context) error {
		order = append(order, "handler")
		return c.String(200, "ok")
	}

	chained := Chain(mw1, mw2)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := &Context{Request: req, Writer: rec}

	chained(c)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Errorf("expected %d calls, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestWrapMiddleware(t *testing.T) {
	called := []string{}

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = append(called, "mw1")
			return next(c)
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = append(called, "mw2")
			return next(c)
		}
	}

	handler := func(c *Context) error {
		called = append(called, "handler")
		return nil
	}

	wrapped := WrapMiddleware(handler, mw1, mw2)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := &Context{Request: req, Writer: rec}

	wrapped(c)

	expected := []string{"mw1", "mw2", "handler"}
	if len(called) != len(expected) {
		t.Errorf("expected %d calls, got %d", len(expected), len(called))
	}
	for i, v := range expected {
		if called[i] != v {
			t.Errorf("position %d: expected %s, got %s", i, v, called[i])
		}
	}
}

func TestMiddlewareShortCircuit(t *testing.T) {
	handlerCalled := false

	authMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if c.Header("Authorization") == "" {
				return c.Unauthorized("missing token")
			}
			return next(c)
		}
	}

	handler := func(c *Context) error {
		handlerCalled = true
		return c.String(200, "ok")
	}

	wrapped := authMiddleware(handler)

	// Request without auth header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := &Context{Request: req, Writer: rec}

	wrapped(c)

	if handlerCalled {
		t.Error("handler should not be called when auth fails")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMiddlewareModifyContext(t *testing.T) {
	userMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Set("user_id", 123)
			return next(c)
		}
	}

	var userID int
	handler := func(c *Context) error {
		userID = c.GetInt("user_id")
		return c.String(200, "ok")
	}

	wrapped := userMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := &Context{
		Request: req,
		Writer:  rec,
		store:   make(map[string]interface{}),
	}

	wrapped(c)

	if userID != 123 {
		t.Errorf("expected user_id=123, got %d", userID)
	}
}

func TestMiddlewareModifyResponse(t *testing.T) {
	headerMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Request-ID", "12345")
			return next(c)
		}
	}

	handler := func(c *Context) error {
		return c.String(200, "ok")
	}

	wrapped := headerMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := &Context{Request: req, Writer: rec}

	wrapped(c)

	if rec.Header().Get("X-Request-ID") != "12345" {
		t.Error("expected X-Request-ID header")
	}
}

func TestMiddlewareErrorHandling(t *testing.T) {
	errorMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			err := next(c)
			if err != nil {
				// Log error, transform it, etc.
				if httpErr, ok := err.(*HTTPError); ok {
					return c.JSON(httpErr.Code, M{"error": httpErr.Message})
				}
				return c.InternalError("something went wrong")
			}
			return nil
		}
	}

	handler := func(c *Context) error {
		return ErrNotFound("resource not found")
	}

	wrapped := errorMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := &Context{Request: req, Writer: rec}

	wrapped(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestEmptyMiddlewareChain(t *testing.T) {
	handler := func(c *Context) error {
		return c.String(200, "ok")
	}

	chained := Chain()(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := &Context{Request: req, Writer: rec}

	chained(c)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRouteGroupMiddleware(t *testing.T) {
	router := NewRouter()
	called := []string{}

	groupMw := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = append(called, "group")
			return next(c)
		}
	}

	routeMw := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = append(called, "route")
			return next(c)
		}
	}

	group := NewRouteGroup(router, "/api", groupMw)
	group.GET("/test", func(c *Context) error {
		called = append(called, "handler")
		return c.String(200, "ok")
	}, routeMw)

	route, params, _ := router.find(http.MethodGet, "/api/test")
	if route == nil {
		t.Fatal("route not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := &Context{Request: req, Writer: rec, params: params, store: make(map[string]interface{})}

	// Apply route middleware
	handler := route.handler
	for i := len(route.middleware) - 1; i >= 0; i-- {
		handler = route.middleware[i](handler)
	}

	handler(c)

	expected := []string{"group", "route", "handler"}
	if len(called) != len(expected) {
		t.Errorf("expected %v, got %v", expected, called)
	}
}

func TestNestedRouteGroups(t *testing.T) {
	router := NewRouter()
	called := []string{}

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = append(called, "mw1")
			return next(c)
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = append(called, "mw2")
			return next(c)
		}
	}

	api := NewRouteGroup(router, "/api", mw1)
	v1 := api.Group("/v1", mw2)
	v1.GET("/users", func(c *Context) error {
		called = append(called, "handler")
		return c.String(200, "ok")
	})

	route, _, _ := router.find(http.MethodGet, "/api/v1/users")
	if route == nil {
		t.Fatal("route not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	c := &Context{Request: req, Writer: rec, store: make(map[string]interface{})}

	handler := route.handler
	for i := len(route.middleware) - 1; i >= 0; i-- {
		handler = route.middleware[i](handler)
	}

	handler(c)

	expected := []string{"mw1", "mw2", "handler"}
	if len(called) != len(expected) {
		t.Errorf("expected %v, got %v", expected, called)
	}
}
