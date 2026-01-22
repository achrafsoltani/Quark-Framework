package quark

import (
	"fmt"
	"net/http"
)

// HTTPError represents an HTTP error with a status code and message.
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code=%d, message=%s, error=%v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// Unwrap returns the wrapped error for errors.Is/As support.
func (e *HTTPError) Unwrap() error {
	return e.Err
}

// NewHTTPError creates a new HTTPError with the given code and message.
func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
	}
}

// WrapError creates a new HTTPError wrapping an existing error.
func WrapError(code int, message string, err error) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common HTTP errors

// ErrBadRequest returns a 400 Bad Request error.
func ErrBadRequest(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusBadRequest)
	}
	return NewHTTPError(http.StatusBadRequest, msg)
}

// ErrUnauthorized returns a 401 Unauthorized error.
func ErrUnauthorized(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusUnauthorized)
	}
	return NewHTTPError(http.StatusUnauthorized, msg)
}

// ErrForbidden returns a 403 Forbidden error.
func ErrForbidden(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusForbidden)
	}
	return NewHTTPError(http.StatusForbidden, msg)
}

// ErrNotFound returns a 404 Not Found error.
func ErrNotFound(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusNotFound)
	}
	return NewHTTPError(http.StatusNotFound, msg)
}

// ErrMethodNotAllowed returns a 405 Method Not Allowed error.
func ErrMethodNotAllowed(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusMethodNotAllowed)
	}
	return NewHTTPError(http.StatusMethodNotAllowed, msg)
}

// ErrConflict returns a 409 Conflict error.
func ErrConflict(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusConflict)
	}
	return NewHTTPError(http.StatusConflict, msg)
}

// ErrUnprocessableEntity returns a 422 Unprocessable Entity error.
func ErrUnprocessableEntity(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusUnprocessableEntity)
	}
	return NewHTTPError(http.StatusUnprocessableEntity, msg)
}

// ErrTooManyRequests returns a 429 Too Many Requests error.
func ErrTooManyRequests(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusTooManyRequests)
	}
	return NewHTTPError(http.StatusTooManyRequests, msg)
}

// ErrInternal returns a 500 Internal Server Error.
func ErrInternal(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusInternalServerError)
	}
	return NewHTTPError(http.StatusInternalServerError, msg)
}

// ErrServiceUnavailable returns a 503 Service Unavailable error.
func ErrServiceUnavailable(msg string) *HTTPError {
	if msg == "" {
		msg = http.StatusText(http.StatusServiceUnavailable)
	}
	return NewHTTPError(http.StatusServiceUnavailable, msg)
}
