package quark

import (
	"encoding/json"
	"net/http"
)

// M is a shorthand for map[string]interface{}.
type M map[string]interface{}

// JSON sends a JSON response with the given status code.
func (c *Context) JSON(code int, data interface{}) error {
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(code)
	c.markWritten()

	if data == nil {
		return nil
	}

	return json.NewEncoder(c.Writer).Encode(data)
}

// JSONPretty sends a formatted JSON response.
func (c *Context) JSONPretty(code int, data interface{}, indent string) error {
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(code)
	c.markWritten()

	if data == nil {
		return nil
	}

	enc := json.NewEncoder(c.Writer)
	enc.SetIndent("", indent)
	return enc.Encode(data)
}

// PaginatedResponse represents a paginated API response.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination holds pagination metadata.
type Pagination struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasMore    bool `json:"has_more"`
}

// JSONPaginated sends a paginated JSON response.
func (c *Context) JSONPaginated(data interface{}, page, perPage, total int) error {
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	resp := PaginatedResponse{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
			HasMore:    page < totalPages,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// String sends a plain text response.
func (c *Context) String(code int, s string) error {
	c.SetHeader("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(code)
	c.markWritten()
	_, err := c.Writer.Write([]byte(s))
	return err
}

// HTML sends an HTML response.
func (c *Context) HTML(code int, html string) error {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(code)
	c.markWritten()
	_, err := c.Writer.Write([]byte(html))
	return err
}

// Blob sends a binary response.
func (c *Context) Blob(code int, contentType string, data []byte) error {
	c.SetHeader("Content-Type", contentType)
	c.Writer.WriteHeader(code)
	c.markWritten()
	_, err := c.Writer.Write(data)
	return err
}

// NoContent sends a 204 No Content response.
func (c *Context) NoContent() error {
	c.Writer.WriteHeader(http.StatusNoContent)
	c.markWritten()
	return nil
}

// Redirect redirects to the given URL.
func (c *Context) Redirect(code int, url string) error {
	if code < 300 || code > 308 {
		return ErrBadRequest("invalid redirect status code")
	}
	c.SetHeader("Location", url)
	c.Writer.WriteHeader(code)
	c.markWritten()
	return nil
}

// Created sends a 201 Created response with the given data.
func (c *Context) Created(data interface{}) error {
	return c.JSON(http.StatusCreated, data)
}

// Accepted sends a 202 Accepted response with the given data.
func (c *Context) Accepted(data interface{}) error {
	return c.JSON(http.StatusAccepted, data)
}

// Error sends an error JSON response.
func (c *Context) Error(code int, message string) error {
	return c.JSON(code, M{
		"error": M{
			"code":    code,
			"message": message,
		},
	})
}

// ErrorWithDetails sends an error response with additional details.
func (c *Context) ErrorWithDetails(code int, message string, details interface{}) error {
	return c.JSON(code, M{
		"error": M{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

// BadRequest sends a 400 Bad Request response.
func (c *Context) BadRequest(msg string) error {
	if msg == "" {
		msg = http.StatusText(http.StatusBadRequest)
	}
	return c.Error(http.StatusBadRequest, msg)
}

// Unauthorized sends a 401 Unauthorized response.
func (c *Context) Unauthorized(msg string) error {
	if msg == "" {
		msg = http.StatusText(http.StatusUnauthorized)
	}
	return c.Error(http.StatusUnauthorized, msg)
}

// Forbidden sends a 403 Forbidden response.
func (c *Context) Forbidden(msg string) error {
	if msg == "" {
		msg = http.StatusText(http.StatusForbidden)
	}
	return c.Error(http.StatusForbidden, msg)
}

// NotFound sends a 404 Not Found response.
func (c *Context) NotFound(msg string) error {
	if msg == "" {
		msg = http.StatusText(http.StatusNotFound)
	}
	return c.Error(http.StatusNotFound, msg)
}

// Conflict sends a 409 Conflict response.
func (c *Context) Conflict(msg string) error {
	if msg == "" {
		msg = http.StatusText(http.StatusConflict)
	}
	return c.Error(http.StatusConflict, msg)
}

// InternalError sends a 500 Internal Server Error response.
func (c *Context) InternalError(msg string) error {
	if msg == "" {
		msg = http.StatusText(http.StatusInternalServerError)
	}
	return c.Error(http.StatusInternalServerError, msg)
}
