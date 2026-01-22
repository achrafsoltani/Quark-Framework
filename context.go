package quark

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Context wraps the HTTP request and response with helper methods.
type Context struct {
	Request  *http.Request
	Writer   http.ResponseWriter
	params   map[string]string
	store    map[string]interface{}
	app      *App
	response bool // tracks if response has been written
}

// newContext creates a new Context for the given request/response.
func newContext(w http.ResponseWriter, r *http.Request, app *App) *Context {
	return &Context{
		Request: r,
		Writer:  w,
		params:  make(map[string]string),
		store:   make(map[string]interface{}),
		app:     app,
	}
}

// reset resets the context for reuse (object pooling).
func (c *Context) reset(w http.ResponseWriter, r *http.Request) {
	c.Request = r
	c.Writer = w
	c.params = make(map[string]string)
	c.store = make(map[string]interface{})
	c.response = false
}

// App returns the application instance.
func (c *Context) App() *App {
	return c.app
}

// Context returns the request's context.Context.
func (c *Context) Context() context.Context {
	return c.Request.Context()
}

// WithContext returns a shallow copy with the given context.
func (c *Context) WithContext(ctx context.Context) *Context {
	c.Request = c.Request.WithContext(ctx)
	return c
}

// SetParams sets the path parameters (called by router).
func (c *Context) SetParams(params map[string]string) {
	c.params = params
}

// Param returns a path parameter by name.
func (c *Context) Param(name string) string {
	return c.params[name]
}

// ParamInt returns a path parameter as int64.
func (c *Context) ParamInt(name string) (int64, error) {
	val := c.params[name]
	if val == "" {
		return 0, ErrBadRequest("missing parameter: " + name)
	}
	return strconv.ParseInt(val, 10, 64)
}

// ParamIntDefault returns a path parameter as int64 with a default value.
func (c *Context) ParamIntDefault(name string, def int64) int64 {
	val, err := c.ParamInt(name)
	if err != nil {
		return def
	}
	return val
}

// Query returns a query parameter by name.
func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

// QueryDefault returns a query parameter with a default value.
func (c *Context) QueryDefault(name, def string) string {
	val := c.Query(name)
	if val == "" {
		return def
	}
	return val
}

// QueryInt returns a query parameter as int.
func (c *Context) QueryInt(name string, def int) int {
	val := c.Query(name)
	if val == "" {
		return def
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return i
}

// QueryInt64 returns a query parameter as int64.
func (c *Context) QueryInt64(name string, def int64) int64 {
	val := c.Query(name)
	if val == "" {
		return def
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return def
	}
	return i
}

// QueryBool returns a query parameter as bool.
func (c *Context) QueryBool(name string) bool {
	val := strings.ToLower(c.Query(name))
	return val == "true" || val == "1" || val == "yes"
}

// QuerySlice returns a query parameter as a slice of strings.
func (c *Context) QuerySlice(name string) []string {
	return c.Request.URL.Query()[name]
}

// Header returns a request header value.
func (c *Context) Header(name string) string {
	return c.Request.Header.Get(name)
}

// SetHeader sets a response header.
func (c *Context) SetHeader(name, value string) {
	c.Writer.Header().Set(name, value)
}

// ContentType returns the request Content-Type header.
func (c *Context) ContentType() string {
	ct := c.Header("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = ct[:idx]
	}
	return strings.TrimSpace(ct)
}

// Bind decodes the request body into v based on Content-Type.
// Currently supports JSON only.
func (c *Context) Bind(v interface{}) error {
	if c.Request.Body == nil {
		return ErrBadRequest("empty request body")
	}

	ct := c.ContentType()
	switch ct {
	case "application/json", "":
		return c.BindJSON(v)
	default:
		return ErrBadRequest("unsupported content type: " + ct)
	}
}

// BindJSON decodes JSON from the request body.
func (c *Context) BindJSON(v interface{}) error {
	if c.Request.Body == nil {
		return ErrBadRequest("empty request body")
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return WrapError(http.StatusBadRequest, "failed to read request body", err)
	}

	if len(body) == 0 {
		return ErrBadRequest("empty request body")
	}

	if err := json.Unmarshal(body, v); err != nil {
		return WrapError(http.StatusBadRequest, "invalid JSON", err)
	}

	return nil
}

// Get retrieves a value from the context store.
func (c *Context) Get(key string) interface{} {
	return c.store[key]
}

// Set stores a value in the context store.
func (c *Context) Set(key string, value interface{}) {
	c.store[key] = value
}

// GetString retrieves a string value from the context store.
func (c *Context) GetString(key string) string {
	if val, ok := c.store[key].(string); ok {
		return val
	}
	return ""
}

// GetInt retrieves an int value from the context store.
func (c *Context) GetInt(key string) int {
	if val, ok := c.store[key].(int); ok {
		return val
	}
	return 0
}

// GetInt64 retrieves an int64 value from the context store.
func (c *Context) GetInt64(key string) int64 {
	if val, ok := c.store[key].(int64); ok {
		return val
	}
	return 0
}

// PaginationParams holds pagination parameters.
type PaginationParams struct {
	Page    int
	PerPage int
	Offset  int
}

// Pagination extracts pagination parameters from query string.
// Uses "page" and "per_page" (or "limit") query params.
func (c *Context) Pagination(defaultPerPage, maxPerPage int) PaginationParams {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}

	perPage := c.QueryInt("per_page", 0)
	if perPage == 0 {
		perPage = c.QueryInt("limit", defaultPerPage)
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	offset := (page - 1) * perPage

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
		Offset:  offset,
	}
}

// RealIP returns the client's real IP address.
// Checks X-Real-IP, X-Forwarded-For, and falls back to RemoteAddr.
func (c *Context) RealIP() string {
	// X-Real-IP
	if ip := c.Header("X-Real-IP"); ip != "" {
		return ip
	}

	// X-Forwarded-For
	if xff := c.Header("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}

	// RemoteAddr
	addr := c.Request.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// Method returns the request HTTP method.
func (c *Context) Method() string {
	return c.Request.Method
}

// Path returns the request URL path.
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// IsWritten returns true if a response has been written.
func (c *Context) IsWritten() bool {
	return c.response
}

// markWritten marks the response as written.
func (c *Context) markWritten() {
	c.response = true
}
