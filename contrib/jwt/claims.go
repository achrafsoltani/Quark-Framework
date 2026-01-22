package jwt

import (
	"encoding/json"
	"time"
)

// Claims represents JWT claims with standard and custom fields.
type Claims struct {
	// Standard claims (RFC 7519)
	Issuer    string `json:"iss,omitempty"`
	Subject   string `json:"sub,omitempty"`
	Audience  string `json:"aud,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	NotBefore int64  `json:"nbf,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	ID        string `json:"jti,omitempty"`

	// Custom claims (arbitrary data)
	Custom map[string]interface{} `json:"-"`
}

// MarshalJSON implements custom JSON marshaling to flatten custom claims.
func (c Claims) MarshalJSON() ([]byte, error) {
	// Create a map with standard claims
	m := make(map[string]interface{})

	if c.Issuer != "" {
		m["iss"] = c.Issuer
	}
	if c.Subject != "" {
		m["sub"] = c.Subject
	}
	if c.Audience != "" {
		m["aud"] = c.Audience
	}
	if c.ExpiresAt != 0 {
		m["exp"] = c.ExpiresAt
	}
	if c.NotBefore != 0 {
		m["nbf"] = c.NotBefore
	}
	if c.IssuedAt != 0 {
		m["iat"] = c.IssuedAt
	}
	if c.ID != "" {
		m["jti"] = c.ID
	}

	// Add custom claims
	for k, v := range c.Custom {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling to extract custom claims.
func (c *Claims) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a map
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract standard claims
	if v, ok := m["iss"].(string); ok {
		c.Issuer = v
	}
	if v, ok := m["sub"].(string); ok {
		c.Subject = v
	}
	if v, ok := m["aud"].(string); ok {
		c.Audience = v
	}
	if v, ok := m["exp"].(float64); ok {
		c.ExpiresAt = int64(v)
	}
	if v, ok := m["nbf"].(float64); ok {
		c.NotBefore = int64(v)
	}
	if v, ok := m["iat"].(float64); ok {
		c.IssuedAt = int64(v)
	}
	if v, ok := m["jti"].(string); ok {
		c.ID = v
	}

	// Collect custom claims
	standardKeys := map[string]bool{
		"iss": true, "sub": true, "aud": true,
		"exp": true, "nbf": true, "iat": true, "jti": true,
	}

	c.Custom = make(map[string]interface{})
	for k, v := range m {
		if !standardKeys[k] {
			c.Custom[k] = v
		}
	}

	return nil
}

// NewClaims creates a new Claims with the given subject and expiration.
func NewClaims(subject string, expiresIn time.Duration) Claims {
	now := time.Now()
	return Claims{
		Subject:   subject,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(expiresIn).Unix(),
	}
}

// WithCustom adds custom claims.
func (c Claims) WithCustom(key string, value interface{}) Claims {
	if c.Custom == nil {
		c.Custom = make(map[string]interface{})
	}
	c.Custom[key] = value
	return c
}

// WithCustomMap adds multiple custom claims.
func (c Claims) WithCustomMap(m map[string]interface{}) Claims {
	if c.Custom == nil {
		c.Custom = make(map[string]interface{})
	}
	for k, v := range m {
		c.Custom[k] = v
	}
	return c
}

// Get retrieves a custom claim value.
func (c *Claims) Get(key string) interface{} {
	if c.Custom == nil {
		return nil
	}
	return c.Custom[key]
}

// GetString retrieves a custom claim as a string.
func (c *Claims) GetString(key string) string {
	v, _ := c.Get(key).(string)
	return v
}

// GetInt retrieves a custom claim as an int.
func (c *Claims) GetInt(key string) int {
	switch v := c.Get(key).(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}

// GetInt64 retrieves a custom claim as an int64.
func (c *Claims) GetInt64(key string) int64 {
	switch v := c.Get(key).(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case int:
		return int64(v)
	default:
		return 0
	}
}

// GetBool retrieves a custom claim as a bool.
func (c *Claims) GetBool(key string) bool {
	v, _ := c.Get(key).(bool)
	return v
}

// GetStringSlice retrieves a custom claim as a string slice.
func (c *Claims) GetStringSlice(key string) []string {
	v := c.Get(key)
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// IsExpired checks if the token has expired.
func (c *Claims) IsExpired() bool {
	if c.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > c.ExpiresAt
}

// ExpiresIn returns the duration until the token expires.
func (c *Claims) ExpiresIn() time.Duration {
	if c.ExpiresAt == 0 {
		return 0
	}
	exp := time.Unix(c.ExpiresAt, 0)
	return time.Until(exp)
}

// UserClaims is a convenience type for user-related claims.
type UserClaims struct {
	Claims
	UserID   int64    `json:"user_id,omitempty"`
	Username string   `json:"username,omitempty"`
	Email    string   `json:"email,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

// NewUserClaims creates user claims.
func NewUserClaims(userID int64, username string, expiresIn time.Duration) UserClaims {
	now := time.Now()
	return UserClaims{
		Claims: Claims{
			Subject:   username,
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(expiresIn).Unix(),
		},
		UserID:   userID,
		Username: username,
	}
}

// HasRole checks if the user has a specific role.
func (c *UserClaims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the given roles.
func (c *UserClaims) HasAnyRole(roles ...string) bool {
	roleSet := make(map[string]bool)
	for _, r := range c.Roles {
		roleSet[r] = true
	}
	for _, r := range roles {
		if roleSet[r] {
			return true
		}
	}
	return false
}
