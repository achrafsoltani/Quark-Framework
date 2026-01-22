package quark

import (
	"net/http"
	"testing"
)

func TestRouterPatternMatching(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		path       string
		shouldMatch bool
		params     map[string]string
	}{
		{
			name:        "exact match",
			pattern:     "/users",
			path:        "/users",
			shouldMatch: true,
			params:      map[string]string{},
		},
		{
			name:        "exact match with trailing slash",
			pattern:     "/users",
			path:        "/users/",
			shouldMatch: true,
			params:      map[string]string{},
		},
		{
			name:        "no match different path",
			pattern:     "/users",
			path:        "/posts",
			shouldMatch: false,
			params:      nil,
		},
		{
			name:        "simple param",
			pattern:     "/users/{id}",
			path:        "/users/123",
			shouldMatch: true,
			params:      map[string]string{"id": "123"},
		},
		{
			name:        "param with regex constraint",
			pattern:     "/users/{id:[0-9]+}",
			path:        "/users/456",
			shouldMatch: true,
			params:      map[string]string{"id": "456"},
		},
		{
			name:        "param with regex constraint no match",
			pattern:     "/users/{id:[0-9]+}",
			path:        "/users/abc",
			shouldMatch: false,
			params:      nil,
		},
		{
			name:        "multiple params",
			pattern:     "/users/{userId}/posts/{postId}",
			path:        "/users/1/posts/99",
			shouldMatch: true,
			params:      map[string]string{"userId": "1", "postId": "99"},
		},
		{
			name:        "catch-all param",
			pattern:     "/files/{path:.*}",
			path:        "/files/dir/subdir/file.txt",
			shouldMatch: true,
			params:      map[string]string{"path": "dir/subdir/file.txt"},
		},
		{
			name:        "root path",
			pattern:     "/",
			path:        "/",
			shouldMatch: true,
			params:      map[string]string{},
		},
		{
			name:        "alpha constraint",
			pattern:     "/users/{name:[a-zA-Z]+}",
			path:        "/users/john",
			shouldMatch: true,
			params:      map[string]string{"name": "john"},
		},
		{
			name:        "uuid pattern",
			pattern:     "/items/{id:[0-9a-f-]+}",
			path:        "/items/550e8400-e29b-41d4-a716-446655440000",
			shouldMatch: true,
			params:      map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()

			router.GET(tt.pattern, func(c *Context) error {
				return c.String(200, "ok")
			})

			route, params, _ := router.find(http.MethodGet, tt.path)

			if tt.shouldMatch {
				if route == nil {
					t.Errorf("expected match for pattern %q with path %q", tt.pattern, tt.path)
					return
				}
				for k, v := range tt.params {
					if params[k] != v {
						t.Errorf("param %q: expected %q, got %q", k, v, params[k])
					}
				}
			} else {
				if route != nil {
					t.Errorf("expected no match for pattern %q with path %q", tt.pattern, tt.path)
				}
			}
		})
	}
}

func TestRouterMethods(t *testing.T) {
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
		t.Run(method, func(t *testing.T) {
			router := NewRouter()

			handler := func(c *Context) error {
				return c.String(200, "ok")
			}

			switch method {
			case http.MethodGet:
				router.GET("/test", handler)
			case http.MethodPost:
				router.POST("/test", handler)
			case http.MethodPut:
				router.PUT("/test", handler)
			case http.MethodPatch:
				router.PATCH("/test", handler)
			case http.MethodDelete:
				router.DELETE("/test", handler)
			case http.MethodOptions:
				router.OPTIONS("/test", handler)
			case http.MethodHead:
				router.HEAD("/test", handler)
			}

			route, _, _ := router.find(method, "/test")
			if route == nil {
				t.Errorf("route not found for method %s", method)
			}
		})
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	router := NewRouter()
	router.GET("/users", func(c *Context) error {
		return c.String(200, "ok")
	})

	// POST should not match but path exists
	route, _, pathMatched := router.find(http.MethodPost, "/users")

	if route != nil {
		t.Error("expected no route for POST /users")
	}
	if !pathMatched {
		t.Error("expected pathMatched to be true")
	}
}

func TestRouterNotFound(t *testing.T) {
	router := NewRouter()
	router.GET("/users", func(c *Context) error {
		return c.String(200, "ok")
	})

	route, _, pathMatched := router.find(http.MethodGet, "/nonexistent")

	if route != nil {
		t.Error("expected no route for /nonexistent")
	}
	if pathMatched {
		t.Error("expected pathMatched to be false")
	}
}

func TestRouterStatic(t *testing.T) {
	router := NewRouter()
	router.Static("/static", "./testdata")

	routes := router.Routes()
	if len(routes) == 0 {
		t.Error("expected static route to be registered")
	}
}

func TestRouterAny(t *testing.T) {
	router := NewRouter()
	router.Any("/any", func(c *Context) error {
		return c.String(200, "ok")
	})

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
		route, _, _ := router.find(method, "/any")
		if route == nil {
			t.Errorf("expected route for %s /any", method)
		}
	}
}
