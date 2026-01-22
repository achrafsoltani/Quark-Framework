package jwt

import (
	"strings"

	"github.com/AchrafSoltani/quark"
)

// MiddlewareConfig defines the configuration for JWT middleware.
type MiddlewareConfig struct {
	// JWT is the JWT handler to use for parsing tokens.
	JWT *JWT

	// TokenLookup is a string in the format of "<source>:<name>" that is used
	// to extract token from the request.
	// Possible values:
	//   - "header:Authorization" (default)
	//   - "query:token"
	//   - "cookie:token"
	TokenLookup string

	// AuthScheme is the authentication scheme (e.g., "Bearer").
	AuthScheme string

	// ContextKey is the key used to store the token in the context.
	ContextKey string

	// ClaimsContextKey is the key used to store the claims in the context.
	ClaimsContextKey string

	// Skipper defines a function to skip this middleware.
	Skipper func(*quark.Context) bool

	// ErrorHandler is called when authentication fails.
	ErrorHandler func(*quark.Context, error) error

	// SuccessHandler is called after successful authentication.
	SuccessHandler func(*quark.Context, *Token)
}

// DefaultMiddlewareConfig returns default middleware configuration.
func DefaultMiddlewareConfig(jwt *JWT) MiddlewareConfig {
	return MiddlewareConfig{
		JWT:              jwt,
		TokenLookup:      "header:Authorization",
		AuthScheme:       "Bearer",
		ContextKey:       "token",
		ClaimsContextKey: "claims",
		Skipper:          nil,
		ErrorHandler:     nil,
		SuccessHandler:   nil,
	}
}

// Middleware returns a JWT authentication middleware.
func Middleware(jwt *JWT) quark.MiddlewareFunc {
	return MiddlewareWithConfig(DefaultMiddlewareConfig(jwt))
}

// MiddlewareWithConfig returns a JWT middleware with the given configuration.
func MiddlewareWithConfig(config MiddlewareConfig) quark.MiddlewareFunc {
	if config.JWT == nil {
		panic("jwt middleware requires a JWT handler")
	}
	if config.TokenLookup == "" {
		config.TokenLookup = "header:Authorization"
	}
	if config.AuthScheme == "" {
		config.AuthScheme = "Bearer"
	}
	if config.ContextKey == "" {
		config.ContextKey = "token"
	}
	if config.ClaimsContextKey == "" {
		config.ClaimsContextKey = "claims"
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
			tokenString := extractor(c)
			if tokenString == "" {
				err := quark.ErrUnauthorized("missing token")
				if config.ErrorHandler != nil {
					return config.ErrorHandler(c, err)
				}
				return err
			}

			// Parse and validate token
			token, err := config.JWT.Parse(tokenString)
			if err != nil {
				authErr := quark.ErrUnauthorized(err.Error())
				if config.ErrorHandler != nil {
					return config.ErrorHandler(c, authErr)
				}
				return authErr
			}

			// Store token and claims in context
			c.Set(config.ContextKey, token)
			c.Set(config.ClaimsContextKey, &token.Claims)

			// Call success handler if set
			if config.SuccessHandler != nil {
				config.SuccessHandler(c, token)
			}

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

// GetToken retrieves the parsed token from the context.
func GetToken(c *quark.Context) *Token {
	if t, ok := c.Get("token").(*Token); ok {
		return t
	}
	return nil
}

// GetClaims retrieves the claims from the context.
func GetClaims(c *quark.Context) *Claims {
	if claims, ok := c.Get("claims").(*Claims); ok {
		return claims
	}
	return nil
}

// GetUserID is a convenience function to get the user ID from claims.
// Looks for "user_id" or "uid" custom claims.
func GetUserID(c *quark.Context) int64 {
	claims := GetClaims(c)
	if claims == nil {
		return 0
	}

	if id := claims.GetInt64("user_id"); id != 0 {
		return id
	}
	if id := claims.GetInt64("uid"); id != 0 {
		return id
	}
	return 0
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

// RequireRoles returns a middleware that requires the user to have specific roles.
func RequireRoles(roles ...string) quark.MiddlewareFunc {
	return func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return quark.ErrUnauthorized("authentication required")
			}

			userRoles := claims.GetStringSlice("roles")
			roleSet := make(map[string]bool)
			for _, r := range userRoles {
				roleSet[r] = true
			}

			for _, required := range roles {
				if !roleSet[required] {
					return quark.ErrForbidden("insufficient permissions")
				}
			}

			return next(c)
		}
	}
}

// RequireAnyRole returns a middleware that requires the user to have any of the given roles.
func RequireAnyRole(roles ...string) quark.MiddlewareFunc {
	return func(next quark.HandlerFunc) quark.HandlerFunc {
		return func(c *quark.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return quark.ErrUnauthorized("authentication required")
			}

			userRoles := claims.GetStringSlice("roles")
			roleSet := make(map[string]bool)
			for _, r := range userRoles {
				roleSet[r] = true
			}

			for _, r := range roles {
				if roleSet[r] {
					return next(c)
				}
			}

			return quark.ErrForbidden("insufficient permissions")
		}
	}
}
