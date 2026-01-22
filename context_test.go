package quark

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContextParams(t *testing.T) {
	c := &Context{
		params: map[string]string{
			"id":   "123",
			"name": "john",
		},
	}

	// Test Param
	if got := c.Param("id"); got != "123" {
		t.Errorf("Param(id): expected 123, got %s", got)
	}
	if got := c.Param("name"); got != "john" {
		t.Errorf("Param(name): expected john, got %s", got)
	}
	if got := c.Param("nonexistent"); got != "" {
		t.Errorf("Param(nonexistent): expected empty, got %s", got)
	}

	// Test ParamInt
	id, err := c.ParamInt("id")
	if err != nil {
		t.Errorf("ParamInt(id): unexpected error: %v", err)
	}
	if id != 123 {
		t.Errorf("ParamInt(id): expected 123, got %d", id)
	}

	// Test ParamInt with non-numeric
	_, err = c.ParamInt("name")
	if err == nil {
		t.Error("ParamInt(name): expected error for non-numeric value")
	}

	// Test ParamIntDefault
	if got := c.ParamIntDefault("id", 0); got != 123 {
		t.Errorf("ParamIntDefault(id): expected 123, got %d", got)
	}
	if got := c.ParamIntDefault("nonexistent", 999); got != 999 {
		t.Errorf("ParamIntDefault(nonexistent): expected 999, got %d", got)
	}
}

func TestContextQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?name=john&age=30&active=true&tags=a&tags=b", nil)
	c := &Context{
		Request: req,
		params:  make(map[string]string),
	}

	// Test Query
	if got := c.Query("name"); got != "john" {
		t.Errorf("Query(name): expected john, got %s", got)
	}
	if got := c.Query("nonexistent"); got != "" {
		t.Errorf("Query(nonexistent): expected empty, got %s", got)
	}

	// Test QueryDefault
	if got := c.QueryDefault("name", "default"); got != "john" {
		t.Errorf("QueryDefault(name): expected john, got %s", got)
	}
	if got := c.QueryDefault("nonexistent", "default"); got != "default" {
		t.Errorf("QueryDefault(nonexistent): expected default, got %s", got)
	}

	// Test QueryInt
	if got := c.QueryInt("age", 0); got != 30 {
		t.Errorf("QueryInt(age): expected 30, got %d", got)
	}
	if got := c.QueryInt("nonexistent", 99); got != 99 {
		t.Errorf("QueryInt(nonexistent): expected 99, got %d", got)
	}
	if got := c.QueryInt("name", 0); got != 0 {
		t.Errorf("QueryInt(name): expected 0 for non-numeric, got %d", got)
	}

	// Test QueryBool
	if got := c.QueryBool("active"); !got {
		t.Error("QueryBool(active): expected true")
	}
	if got := c.QueryBool("nonexistent"); got {
		t.Error("QueryBool(nonexistent): expected false")
	}

	// Test QuerySlice
	tags := c.QuerySlice("tags")
	if len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Errorf("QuerySlice(tags): expected [a, b], got %v", tags)
	}
}

func TestContextBind(t *testing.T) {
	type Input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	tests := []struct {
		name        string
		contentType string
		body        string
		expectErr   bool
		expected    Input
	}{
		{
			name:        "valid JSON",
			contentType: "application/json",
			body:        `{"name":"John","email":"john@example.com","age":30}`,
			expectErr:   false,
			expected:    Input{Name: "John", Email: "john@example.com", Age: 30},
		},
		{
			name:        "valid JSON without content-type",
			contentType: "",
			body:        `{"name":"Jane"}`,
			expectErr:   false,
			expected:    Input{Name: "Jane"},
		},
		{
			name:        "invalid JSON",
			contentType: "application/json",
			body:        `{"name":}`,
			expectErr:   true,
		},
		{
			name:        "empty body",
			contentType: "application/json",
			body:        "",
			expectErr:   true,
		},
		{
			name:        "unsupported content type",
			contentType: "application/xml",
			body:        "<user><name>John</name></user>",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			c := &Context{
				Request: req,
				params:  make(map[string]string),
			}

			var input Input
			err := c.Bind(&input)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if input != tt.expected {
					t.Errorf("expected %+v, got %+v", tt.expected, input)
				}
			}
		})
	}
}

func TestContextStore(t *testing.T) {
	c := &Context{
		store: make(map[string]interface{}),
	}

	// Test Set and Get
	c.Set("user", "john")
	c.Set("count", 42)
	c.Set("active", true)

	if got := c.Get("user"); got != "john" {
		t.Errorf("Get(user): expected john, got %v", got)
	}
	if got := c.Get("nonexistent"); got != nil {
		t.Errorf("Get(nonexistent): expected nil, got %v", got)
	}

	// Test GetString
	if got := c.GetString("user"); got != "john" {
		t.Errorf("GetString(user): expected john, got %s", got)
	}
	if got := c.GetString("count"); got != "" {
		t.Errorf("GetString(count): expected empty for non-string, got %s", got)
	}

	// Test GetInt
	if got := c.GetInt("count"); got != 42 {
		t.Errorf("GetInt(count): expected 42, got %d", got)
	}
	if got := c.GetInt("user"); got != 0 {
		t.Errorf("GetInt(user): expected 0 for non-int, got %d", got)
	}
}

func TestContextPagination(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		defaultPerPage int
		maxPerPage     int
		expectedPage   int
		expectedPerPage int
		expectedOffset int
	}{
		{
			name:           "defaults",
			query:          "",
			defaultPerPage: 20,
			maxPerPage:     100,
			expectedPage:   1,
			expectedPerPage: 20,
			expectedOffset: 0,
		},
		{
			name:           "custom page and per_page",
			query:          "page=3&per_page=50",
			defaultPerPage: 20,
			maxPerPage:     100,
			expectedPage:   3,
			expectedPerPage: 50,
			expectedOffset: 100,
		},
		{
			name:           "per_page exceeds max",
			query:          "per_page=200",
			defaultPerPage: 20,
			maxPerPage:     100,
			expectedPage:   1,
			expectedPerPage: 100,
			expectedOffset: 0,
		},
		{
			name:           "negative page defaults to 1",
			query:          "page=-5",
			defaultPerPage: 20,
			maxPerPage:     100,
			expectedPage:   1,
			expectedPerPage: 20,
			expectedOffset: 0,
		},
		{
			name:           "limit alias for per_page",
			query:          "limit=30",
			defaultPerPage: 20,
			maxPerPage:     100,
			expectedPage:   1,
			expectedPerPage: 30,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			c := &Context{
				Request: req,
				params:  make(map[string]string),
			}

			p := c.Pagination(tt.defaultPerPage, tt.maxPerPage)

			if p.Page != tt.expectedPage {
				t.Errorf("Page: expected %d, got %d", tt.expectedPage, p.Page)
			}
			if p.PerPage != tt.expectedPerPage {
				t.Errorf("PerPage: expected %d, got %d", tt.expectedPerPage, p.PerPage)
			}
			if p.Offset != tt.expectedOffset {
				t.Errorf("Offset: expected %d, got %d", tt.expectedOffset, p.Offset)
			}
		})
	}
}

func TestContextRealIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "1.2.3.4"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "1.2.3.4",
		},
		{
			name:       "X-Forwarded-For single",
			headers:    map[string]string{"X-Forwarded-For": "5.6.7.8"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "5.6.7.8",
		},
		{
			name:       "X-Forwarded-For multiple",
			headers:    map[string]string{"X-Forwarded-For": "1.1.1.1, 2.2.2.2, 3.3.3.3"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "1.1.1.1",
		},
		{
			name:       "fallback to RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr

			c := &Context{Request: req}
			if got := c.RealIP(); got != tt.expected {
				t.Errorf("RealIP(): expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestContextHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	rec := httptest.NewRecorder()
	c := &Context{
		Request: req,
		Writer:  rec,
	}

	// Test Header
	if got := c.Header("Authorization"); got != "Bearer token123" {
		t.Errorf("Header(Authorization): expected 'Bearer token123', got %s", got)
	}

	// Test ContentType
	if got := c.ContentType(); got != "application/json" {
		t.Errorf("ContentType(): expected 'application/json', got %s", got)
	}

	// Test SetHeader
	c.SetHeader("X-Custom", "value")
	if got := rec.Header().Get("X-Custom"); got != "value" {
		t.Errorf("SetHeader: expected 'value', got %s", got)
	}
}

func TestContextMethodAndPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	c := &Context{Request: req}

	if got := c.Method(); got != http.MethodPost {
		t.Errorf("Method(): expected POST, got %s", got)
	}
	if got := c.Path(); got != "/api/users" {
		t.Errorf("Path(): expected /api/users, got %s", got)
	}
}

func TestContextReset(t *testing.T) {
	req1 := httptest.NewRequest(http.MethodGet, "/first", nil)
	rec1 := httptest.NewRecorder()

	c := &Context{
		Request:  req1,
		Writer:   rec1,
		params:   map[string]string{"id": "1"},
		store:    map[string]interface{}{"user": "john"},
		response: true,
	}

	req2 := httptest.NewRequest(http.MethodPost, "/second", nil)
	rec2 := httptest.NewRecorder()

	c.reset(rec2, req2)

	if c.Request != req2 {
		t.Error("reset: Request not updated")
	}
	if c.Writer != rec2 {
		t.Error("reset: Writer not updated")
	}
	if len(c.params) != 0 {
		t.Error("reset: params not cleared")
	}
	if len(c.store) != 0 {
		t.Error("reset: store not cleared")
	}
	if c.response {
		t.Error("reset: response not cleared")
	}
}

func TestContextBindJSON(t *testing.T) {
	type Data struct {
		Value string `json:"value"`
	}

	body := bytes.NewBufferString(`{"value":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")

	c := &Context{Request: req}

	var data Data
	if err := c.BindJSON(&data); err != nil {
		t.Errorf("BindJSON: unexpected error: %v", err)
	}
	if data.Value != "test" {
		t.Errorf("BindJSON: expected 'test', got %s", data.Value)
	}
}
