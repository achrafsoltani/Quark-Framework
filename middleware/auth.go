package middleware

import (
	"strings"

	"github.com/AchrafSoltani/quark"
)

// AuthConfig defines the configuration for Auth middleware.
type AuthConfig struct {
	// Validator is the function that validates the token/credentials.
	// It receives the token string and should return user data and error.
	Validator func(token string) (interface{}, error)

	// TokenLookup is a string in the format of "<source>:<name>" that is used
	// to extract token from the request.
	// Possible values:
	//   - "header:Authorization" (default)
	//   - "header:X-API-Key"
	//   - "query:token"
	//   - "cookie:token"
	TokenLookup string

	// AuthScheme is the authentication scheme (e.g., "Bearer").
	// Only used when TokenLookup is a header.
	AuthScheme string

	// ContextKey is the key used to store the user data in the context.
	ContextKey string

	// Skipper defines a function to skip this middleware.
	Skipper func(*quark.Context) bool

	// ErrorHandler is called when authentication fails.
	ErrorHandler func(*quark.Context, error) error
}

// DefaultAuthConfig is the default auth configuration.
var DefaultAuthConfig = AuthConfig{
	TokenLookup: "header:Authorization",
	AuthScheme:  "Bearer",
	ContextKey:  "user",
	Skipper:     nil,
}

// Auth returns an Auth middleware with the given validator.
func Auth(validator func(token string) (interface{}, error)) quark.MiddlewareFunc {
	config := DefaultAuthConfig
	config.Validator = validator
	return AuthWithConfig(config)
}

// AuthWithConfig returns an Auth middleware with the given configuration.
func AuthWithConfig(config AuthConfig) quark.MiddlewareFunc {
	if config.Validator == nil {
		panic("auth middleware requires a validator function")
	}
	if config.TokenLookup == "" {
		config.TokenLookup = DefaultAuthConfig.TokenLookup
	}
	if config.ContextKey == "" {
		config.ContextKey = DefaultAuthConfig.ContextKey
	}

	// Parse token lookup
	parts := strings.Split(config.TokenLookup, ":")
	if len(parts) != 2 {
		panic("invalid TokenLookup format, expected <source>:<name>")
	}
	source := parts[0]
	name := parts[1]

	// Build extractor
	var extractor func(*quark.Context) string
	switch source {
	case "header":
		extractor = headerExtractor(name, config.AuthScheme)
	case "query":
		extractor = queryExtractor(name)
	case "cookie":
		extractor = cookieExtractor(name)
	default:
		panic("invalid token source: " + source)
	}

	return func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			// Check skipper
			if config.Skipper != nil && config.Skipper(c) {
				return next(c)
			}

			// Extract token
			token := extractor(c)
			if token == "" {
				err := quark.ErrUnauthorized("missing or invalid token")
				if config.ErrorHandler != nil {
					return config.ErrorHandler(c, err)
				}
				return err
			}

			// Validate token
			user, err := config.Validator(token)
			if err != nil {
				authErr := quark.ErrUnauthorized("invalid token")
				if config.ErrorHandler != nil {
					return config.ErrorHandler(c, authErr)
				}
				return authErr
			}

			// Store user in context
			c.Set(config.ContextKey, user)

			return next(c)
		}
	}
}

// headerExtractor creates a token extractor from a header.
func headerExtractor(header, scheme string) func(*quark.Context) string {
	return func(c *quark.Context) string {
		auth := c.Header(header)
		if auth == "" {
			return ""
		}

		if scheme != "" {
			prefix := scheme + " "
			if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
				return auth[len(prefix):]
			}
			return ""
		}

		return auth
	}
}

// queryExtractor creates a token extractor from a query parameter.
func queryExtractor(name string) func(*quark.Context) string {
	return func(c *quark.Context) string {
		return c.Query(name)
	}
}

// cookieExtractor creates a token extractor from a cookie.
func cookieExtractor(name string) func(*quark.Context) string {
	return func(c *quark.Context) string {
		cookie, err := c.Request.Cookie(name)
		if err != nil {
			return ""
		}
		return cookie.Value
	}
}

// APIKey returns an API key authentication middleware.
func APIKey(validator func(key string) (interface{}, error)) quark.MiddlewareFunc {
	return AuthWithConfig(AuthConfig{
		Validator:   validator,
		TokenLookup: "header:X-API-Key",
		AuthScheme:  "",
		ContextKey:  "api_key_user",
	})
}

// SkipPaths returns a skipper that skips the given paths.
func SkipPaths(paths ...string) func(*quark.Context) bool {
	pathMap := make(map[string]bool)
	for _, path := range paths {
		pathMap[path] = true
	}
	return func(c *quark.Context) bool {
		return pathMap[c.Path()]
	}
}

// SkipPathPrefixes returns a skipper that skips paths with the given prefixes.
func SkipPathPrefixes(prefixes ...string) func(*quark.Context) bool {
	return func(c *quark.Context) bool {
		path := c.Path()
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		return false
	}
}

// BasicAuth returns a Basic authentication middleware.
func BasicAuth(validator func(username, password string) (interface{}, error)) quark.MiddlewareFunc {
	return func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			username, password, ok := c.Request.BasicAuth()
			if !ok {
				c.SetHeader("WWW-Authenticate", `Basic realm="Restricted"`)
				return quark.ErrUnauthorized("authentication required")
			}

			user, err := validator(username, password)
			if err != nil {
				c.SetHeader("WWW-Authenticate", `Basic realm="Restricted"`)
				return quark.ErrUnauthorized("invalid credentials")
			}

			c.Set("user", user)
			return next(c)
		}
	}
}

// RequireAuth returns a middleware that requires the user to be authenticated.
// Use this after Auth middleware to ensure the user key is set.
func RequireAuth(contextKey string) quark.MiddlewareFunc {
	if contextKey == "" {
		contextKey = "user"
	}
	return func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			if c.Get(contextKey) == nil {
				return quark.ErrUnauthorized("authentication required")
			}
			return next(c)
		}
	}
}
