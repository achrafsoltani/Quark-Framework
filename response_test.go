package quark

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	data := M{"message": "hello", "count": 42}
	err := c.JSON(http.StatusOK, data)

	if err != nil {
		t.Errorf("JSON: unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("JSON: expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("JSON: expected content-type application/json, got %s", ct)
	}

	var result M
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Errorf("JSON: failed to decode response: %v", err)
	}
	if result["message"] != "hello" {
		t.Errorf("JSON: expected message=hello, got %v", result["message"])
	}
}

func TestContextJSONPaginated(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	items := []string{"a", "b", "c"}
	err := c.JSONPaginated(items, 2, 10, 25)

	if err != nil {
		t.Errorf("JSONPaginated: unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("JSONPaginated: expected status 200, got %d", rec.Code)
	}

	var result PaginatedResponse
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Errorf("JSONPaginated: failed to decode response: %v", err)
	}

	if result.Pagination.Page != 2 {
		t.Errorf("JSONPaginated: expected page=2, got %d", result.Pagination.Page)
	}
	if result.Pagination.PerPage != 10 {
		t.Errorf("JSONPaginated: expected per_page=10, got %d", result.Pagination.PerPage)
	}
	if result.Pagination.Total != 25 {
		t.Errorf("JSONPaginated: expected total=25, got %d", result.Pagination.Total)
	}
	if result.Pagination.TotalPages != 3 {
		t.Errorf("JSONPaginated: expected total_pages=3, got %d", result.Pagination.TotalPages)
	}
	if !result.Pagination.HasMore {
		t.Error("JSONPaginated: expected has_more=true")
	}
}

func TestContextString(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	err := c.String(http.StatusOK, "Hello, World!")

	if err != nil {
		t.Errorf("String: unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("String: expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("String: expected content-type text/plain, got %s", ct)
	}
	if rec.Body.String() != "Hello, World!" {
		t.Errorf("String: expected 'Hello, World!', got %s", rec.Body.String())
	}
}

func TestContextHTML(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	err := c.HTML(http.StatusOK, "<h1>Hello</h1>")

	if err != nil {
		t.Errorf("HTML: unexpected error: %v", err)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("HTML: expected content-type text/html, got %s", ct)
	}
	if rec.Body.String() != "<h1>Hello</h1>" {
		t.Errorf("HTML: expected '<h1>Hello</h1>', got %s", rec.Body.String())
	}
}

func TestContextNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	err := c.NoContent()

	if err != nil {
		t.Errorf("NoContent: unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("NoContent: expected status 204, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("NoContent: expected empty body, got %s", rec.Body.String())
	}
}

func TestContextCreated(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	data := M{"id": 1, "name": "test"}
	err := c.Created(data)

	if err != nil {
		t.Errorf("Created: unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("Created: expected status 201, got %d", rec.Code)
	}
}

func TestContextRedirect(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	err := c.Redirect(http.StatusFound, "/new-location")

	if err != nil {
		t.Errorf("Redirect: unexpected error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Errorf("Redirect: expected status 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/new-location" {
		t.Errorf("Redirect: expected location '/new-location', got %s", loc)
	}
}

func TestContextRedirectInvalidCode(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	err := c.Redirect(http.StatusOK, "/somewhere")

	if err == nil {
		t.Error("Redirect: expected error for invalid status code")
	}
}

func TestContextErrorResponses(t *testing.T) {
	tests := []struct {
		name     string
		method   func(*Context, string) error
		message  string
		expected int
	}{
		{"BadRequest", (*Context).BadRequest, "bad input", http.StatusBadRequest},
		{"Unauthorized", (*Context).Unauthorized, "please login", http.StatusUnauthorized},
		{"Forbidden", (*Context).Forbidden, "access denied", http.StatusForbidden},
		{"NotFound", (*Context).NotFound, "not found", http.StatusNotFound},
		{"Conflict", (*Context).Conflict, "conflict", http.StatusConflict},
		{"InternalError", (*Context).InternalError, "server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c := &Context{Writer: rec}

			err := tt.method(c, tt.message)

			if err != nil {
				t.Errorf("%s: unexpected error: %v", tt.name, err)
			}
			if rec.Code != tt.expected {
				t.Errorf("%s: expected status %d, got %d", tt.name, tt.expected, rec.Code)
			}

			var result M
			if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
				t.Errorf("%s: failed to decode response: %v", tt.name, err)
			}

			errObj, ok := result["error"].(map[string]interface{})
			if !ok {
				t.Errorf("%s: expected error object in response", tt.name)
				return
			}
			if errObj["message"] != tt.message {
				t.Errorf("%s: expected message=%s, got %v", tt.name, tt.message, errObj["message"])
			}
		})
	}
}

func TestContextErrorWithDefaultMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	c.BadRequest("")

	var result M
	json.NewDecoder(rec.Body).Decode(&result)

	errObj := result["error"].(map[string]interface{})
	if errObj["message"] != "Bad Request" {
		t.Errorf("expected default message 'Bad Request', got %v", errObj["message"])
	}
}

func TestContextBlob(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
	err := c.Blob(http.StatusOK, "image/png", data)

	if err != nil {
		t.Errorf("Blob: unexpected error: %v", err)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Blob: expected content-type image/png, got %s", ct)
	}
	if rec.Body.Len() != 4 {
		t.Errorf("Blob: expected 4 bytes, got %d", rec.Body.Len())
	}
}

func TestContextIsWritten(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	if c.IsWritten() {
		t.Error("IsWritten: expected false before writing")
	}

	c.String(200, "test")

	if !c.IsWritten() {
		t.Error("IsWritten: expected true after writing")
	}
}

func TestContextJSONPretty(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{Writer: rec}

	data := M{"name": "test"}
	err := c.JSONPretty(http.StatusOK, data, "  ")

	if err != nil {
		t.Errorf("JSONPretty: unexpected error: %v", err)
	}

	// Check that output is formatted (contains newlines)
	body := rec.Body.String()
	if body == `{"name":"test"}` {
		t.Error("JSONPretty: expected formatted output")
	}
}

func TestPaginationCalculation(t *testing.T) {
	tests := []struct {
		page       int
		perPage    int
		total      int
		totalPages int
		hasMore    bool
	}{
		{1, 10, 25, 3, true},
		{3, 10, 25, 3, false},
		{1, 10, 10, 1, false},
		{1, 10, 0, 0, false},
		{2, 20, 100, 5, true},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		c := &Context{Writer: rec}

		c.JSONPaginated([]string{}, tt.page, tt.perPage, tt.total)

		var result PaginatedResponse
		json.NewDecoder(rec.Body).Decode(&result)

		if result.Pagination.TotalPages != tt.totalPages {
			t.Errorf("page=%d perPage=%d total=%d: expected totalPages=%d, got %d",
				tt.page, tt.perPage, tt.total, tt.totalPages, result.Pagination.TotalPages)
		}
		if result.Pagination.HasMore != tt.hasMore {
			t.Errorf("page=%d perPage=%d total=%d: expected hasMore=%v, got %v",
				tt.page, tt.perPage, tt.total, tt.hasMore, result.Pagination.HasMore)
		}
	}
}
