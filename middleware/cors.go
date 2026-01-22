// Package middleware provides built-in middleware for the Quark framework.
package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/AchrafSoltani/quark"
)

// CORSConfig defines the configuration for CORS middleware.
type CORSConfig struct {
	// AllowOrigins is a list of origins that may access the resource.
	// Use "*" to allow any origin, or specify explicit origins.
	AllowOrigins []string

	// AllowMethods is a list of methods that are allowed.
	AllowMethods []string

	// AllowHeaders is a list of headers that are allowed in requests.
	AllowHeaders []string

	// ExposeHeaders is a list of headers that browsers are allowed to access.
	ExposeHeaders []string

	// AllowCredentials indicates whether the request can include user credentials.
	AllowCredentials bool

	// MaxAge indicates how long the results of a preflight request can be cached.
	MaxAge int
}

// DefaultCORSConfig is the default CORS configuration.
var DefaultCORSConfig = CORSConfig{
	AllowOrigins: []string{"*"},
	AllowMethods: []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodHead,
	},
	AllowHeaders: []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		"X-Requested-With",
	},
	ExposeHeaders:    []string{},
	AllowCredentials: false,
	MaxAge:           86400, // 24 hours
}

// CORS returns a CORS middleware with the given configuration.
func CORS(config CORSConfig) quark.MiddlewareFunc {
	// Precompute allowed origins map for faster lookup
	allowAllOrigins := false
	allowedOrigins := make(map[string]bool)
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
		allowedOrigins[origin] = true
	}

	// Precompute header values
	allowMethodsHeader := strings.Join(config.AllowMethods, ", ")
	allowHeadersHeader := strings.Join(config.AllowHeaders, ", ")
	exposeHeadersHeader := strings.Join(config.ExposeHeaders, ", ")
	maxAgeHeader := strconv.Itoa(config.MaxAge)

	return func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			origin := c.Header("Origin")

			// Check if origin is allowed
			var allowedOrigin string
			if origin != "" {
				if allowAllOrigins {
					if config.AllowCredentials {
						allowedOrigin = origin
					} else {
						allowedOrigin = "*"
					}
				} else if allowedOrigins[origin] {
					allowedOrigin = origin
				}
			}

			// Set CORS headers
			if allowedOrigin != "" {
				c.SetHeader("Access-Control-Allow-Origin", allowedOrigin)

				if config.AllowCredentials {
					c.SetHeader("Access-Control-Allow-Credentials", "true")
				}

				if exposeHeadersHeader != "" {
					c.SetHeader("Access-Control-Expose-Headers", exposeHeadersHeader)
				}
			}

			// Handle preflight request
			if c.Method() == http.MethodOptions {
				if allowedOrigin != "" {
					c.SetHeader("Access-Control-Allow-Methods", allowMethodsHeader)
					c.SetHeader("Access-Control-Allow-Headers", allowHeadersHeader)
					c.SetHeader("Access-Control-Max-Age", maxAgeHeader)
				}

				c.SetHeader("Content-Length", "0")
				c.Writer.WriteHeader(http.StatusNoContent)
				return nil
			}

			return next(c)
		}
	}
}

// CORSWithConfig returns a CORS middleware with default configuration.
func CORSDefault() quark.MiddlewareFunc {
	return CORS(DefaultCORSConfig)
}

// AllowOrigins creates a CORS config with specific allowed origins.
func AllowOrigins(origins ...string) CORSConfig {
	config := DefaultCORSConfig
	config.AllowOrigins = origins
	return config
}

// AllowAllOrigins creates a CORS config that allows all origins.
func AllowAllOrigins() CORSConfig {
	return DefaultCORSConfig
}

// AllowOriginsWithCredentials creates a CORS config with specific origins and credentials.
func AllowOriginsWithCredentials(origins ...string) CORSConfig {
	config := DefaultCORSConfig
	config.AllowOrigins = origins
	config.AllowCredentials = true
	return config
}
