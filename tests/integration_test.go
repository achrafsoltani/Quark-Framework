package tests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/AchrafSoltani/quark"
	"github.com/AchrafSoltani/quark/middleware"
)

// TestIntegration_FullRequestCycle tests a complete HTTP request/response cycle
// with multiple middleware and handlers.
func TestIntegration_FullRequestCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create app with middleware
	app := quark.New()
	app.Use(middleware.Recovery())
	app.Use(middleware.Logger())

	// Setup routes
	app.GET("/health", func(c *quark.Context) error {
		return c.JSON(200, quark.M{"status": "ok"})
	})

	app.POST("/users", func(c *quark.Context) error {
		var input struct {
			Name  string `json:"name" validate:"required,min:2"`
			Email string `json:"email" validate:"required,email"`
		}

		if err := c.Bind(&input); err != nil {
			return err
		}

		if errs := quark.Validate(input); errs.HasErrors() {
			return c.ErrorWithDetails(400, "Validation failed", errs.ToMap())
		}

		return c.Created(quark.M{
			"id":    1,
			"name":  input.Name,
			"email": input.Email,
		})
	})

	// Test GET request
	t.Run("GET /health", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		app.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["status"] != "ok" {
			t.Errorf("expected status=ok, got %v", response["status"])
		}
	})

	// Test POST request with validation
	t.Run("POST /users - valid data", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"John Doe","email":"john@example.com"}`)
		req := httptest.NewRequest("POST", "/users", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.ServeHTTP(w, req)

		if w.Code != 201 {
			t.Errorf("expected status 201, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["name"] != "John Doe" {
			t.Errorf("expected name=John Doe, got %v", response["name"])
		}
	})

	// Test POST request with invalid data
	t.Run("POST /users - invalid data", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"J","email":"invalid"}`)
		req := httptest.NewRequest("POST", "/users", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test 404
	t.Run("GET /notfound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/notfound", nil)
		w := httptest.NewRecorder()

		app.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

// TestIntegration_MiddlewareChain tests middleware execution order.
func TestIntegration_MiddlewareChain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	var order []string

	// Create middleware that records execution order
	mw1 := func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			order = append(order, "mw1-before")
			err := next(c)
			order = append(order, "mw1-after")
			return err
		}
	}

	mw2 := func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			order = append(order, "mw2-before")
			err := next(c)
			order = append(order, "mw2-after")
			return err
		}
	}

	app := quark.New()
	app.Use(mw1)
	app.Use(mw2)

	app.GET("/test", func(c *quark.Context) error {
		order = append(order, "handler")
		return c.JSON(200, quark.M{"ok": true})
	})

	// Execute request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	// Verify order
	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d]: expected %s, got %s", i, v, order[i])
		}
	}
}

// TestIntegration_RouteGroups tests route grouping with prefix and middleware.
func TestIntegration_RouteGroups(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	app := quark.New()

	// API v1 group
	v1 := app.Group("/api/v1")
	v1.GET("/users", func(c *quark.Context) error {
		return c.JSON(200, quark.M{"version": "v1"})
	})

	// API v2 group
	v2 := app.Group("/api/v2")
	v2.GET("/users", func(c *quark.Context) error {
		return c.JSON(200, quark.M{"version": "v2"})
	})

	// Nested group
	admin := v1.Group("/admin")
	admin.GET("/stats", func(c *quark.Context) error {
		return c.JSON(200, quark.M{"admin": true})
	})

	tests := []struct {
		path         string
		expectedKey  string
		expectedVal  interface{}
		expectedCode int
	}{
		{"/api/v1/users", "version", "v1", 200},
		{"/api/v2/users", "version", "v2", 200},
		{"/api/v1/admin/stats", "admin", true, 200},
		{"/api/v3/users", "", nil, 404},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectedCode == 200 {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if response[tt.expectedKey] != tt.expectedVal {
					t.Errorf("expected %s=%v, got %v", tt.expectedKey, tt.expectedVal, response[tt.expectedKey])
				}
			}
		})
	}
}

// TestIntegration_ErrorHandling tests error handling and recovery.
func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	app := quark.New()
	app.Use(middleware.Recovery())

	// Route that returns HTTPError
	app.GET("/error", func(c *quark.Context) error {
		return quark.ErrUnauthorized("access denied")
	})

	// Route that panics
	app.GET("/panic", func(c *quark.Context) error {
		panic("something went wrong")
	})

	t.Run("HTTPError", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("Panic Recovery", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code != 500 {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}
