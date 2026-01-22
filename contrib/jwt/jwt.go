// Package jwt provides JWT (JSON Web Token) utilities using only the standard library.
// It implements HS256 (HMAC-SHA256) signing without external dependencies.
//
// Basic usage:
//
//	// Create JWT handler
//	secret := []byte("your-secret-key")
//	jwtHandler := jwt.NewWithSecret(secret)
//
//	// Generate a token
//	claims := jwt.NewClaims()
//	claims.Subject = "user123"
//	claims.Set("role", "admin")
//	token, err := jwtHandler.Generate(claims)
//
//	// Parse and validate a token
//	parsedToken, err := jwtHandler.Parse(token)
//	if err != nil {
//	    // Invalid or expired token
//	}
//	userID := parsedToken.Claims.Subject
//	role := parsedToken.Claims.GetString("role")
//
// Integration with Quark middleware:
//
//	jwtHandler := jwt.NewWithSecret([]byte("secret"))
//	app.Use(jwt.Middleware(jwtHandler))
//
//	app.GET("/protected", func(c *quark.Context) error {
//	    claims := c.Get("claims").(jwt.Claims)
//	    userID := claims.Subject
//	    return c.JSON(200, quark.M{"user_id": userID})
//	})
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Algorithm constants
const (
	AlgorithmHS256 = "HS256"
)

// Common errors
var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrTokenNotYetValid = errors.New("token is not yet valid")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrMissingClaims    = errors.New("missing required claims")
)

// Header represents the JWT header.
type Header struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

// Token represents a parsed JWT.
type Token struct {
	Header    Header
	Claims    Claims
	Signature string
	Raw       string
	Valid     bool
}

// Config holds JWT configuration.
type Config struct {
	// Secret is the HMAC secret key.
	Secret []byte

	// Issuer is the token issuer (iss claim).
	Issuer string

	// Audience is the intended audience (aud claim).
	Audience string

	// ExpiresIn is the token expiration duration.
	ExpiresIn time.Duration

	// NotBeforeLeeway is the leeway for not before validation.
	NotBeforeLeeway time.Duration

	// ExpirationLeeway is the leeway for expiration validation.
	ExpirationLeeway time.Duration
}

// DefaultConfig returns a default JWT configuration.
func DefaultConfig(secret []byte) Config {
	return Config{
		Secret:           secret,
		ExpiresIn:        24 * time.Hour,
		NotBeforeLeeway:  0,
		ExpirationLeeway: 0,
	}
}

// JWT is a JWT handler with configuration.
type JWT struct {
	config Config
}

// New creates a new JWT handler with the given configuration.
func New(config Config) *JWT {
	return &JWT{config: config}
}

// NewWithSecret creates a new JWT handler with just a secret.
func NewWithSecret(secret []byte) *JWT {
	return New(DefaultConfig(secret))
}

// Generate creates a new JWT with the given claims.
func (j *JWT) Generate(claims Claims) (string, error) {
	now := time.Now()

	// Set default claims if not provided
	if claims.IssuedAt == 0 {
		claims.IssuedAt = now.Unix()
	}
	if claims.ExpiresAt == 0 && j.config.ExpiresIn > 0 {
		claims.ExpiresAt = now.Add(j.config.ExpiresIn).Unix()
	}
	if claims.Issuer == "" && j.config.Issuer != "" {
		claims.Issuer = j.config.Issuer
	}
	if claims.Audience == "" && j.config.Audience != "" {
		claims.Audience = j.config.Audience
	}

	return j.Sign(claims)
}

// Sign creates a JWT from claims.
func (j *JWT) Sign(claims Claims) (string, error) {
	header := Header{
		Algorithm: AlgorithmHS256,
		Type:      "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	headerEncoded := base64URLEncode(headerJSON)
	claimsEncoded := base64URLEncode(claimsJSON)

	signingInput := headerEncoded + "." + claimsEncoded
	signature := j.sign(signingInput)

	return signingInput + "." + signature, nil
}

// Parse parses and validates a JWT string.
func (j *JWT) Parse(tokenString string) (*Token, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Decode header
	headerJSON, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	var header Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal header: %w", err)
	}

	if header.Algorithm != AlgorithmHS256 {
		return nil, fmt.Errorf("unsupported algorithm: %s", header.Algorithm)
	}

	// Decode claims
	claimsJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	expectedSignature := j.sign(signingInput)

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return nil, ErrInvalidSignature
	}

	// Validate claims
	if err := j.validateClaims(&claims); err != nil {
		return nil, err
	}

	return &Token{
		Header:    header,
		Claims:    claims,
		Signature: parts[2],
		Raw:       tokenString,
		Valid:     true,
	}, nil
}

// validateClaims validates the standard claims.
func (j *JWT) validateClaims(claims *Claims) error {
	now := time.Now().Unix()

	// Check expiration
	if claims.ExpiresAt > 0 {
		exp := claims.ExpiresAt + int64(j.config.ExpirationLeeway.Seconds())
		if now > exp {
			return ErrExpiredToken
		}
	}

	// Check not before
	if claims.NotBefore > 0 {
		nbf := claims.NotBefore - int64(j.config.NotBeforeLeeway.Seconds())
		if now < nbf {
			return ErrTokenNotYetValid
		}
	}

	// Validate issuer if configured
	if j.config.Issuer != "" && claims.Issuer != j.config.Issuer {
		return fmt.Errorf("invalid issuer: expected %s, got %s", j.config.Issuer, claims.Issuer)
	}

	// Validate audience if configured
	if j.config.Audience != "" && claims.Audience != j.config.Audience {
		return fmt.Errorf("invalid audience: expected %s, got %s", j.config.Audience, claims.Audience)
	}

	return nil
}

// sign creates an HMAC-SHA256 signature.
func (j *JWT) sign(input string) string {
	h := hmac.New(sha256.New, j.config.Secret)
	h.Write([]byte(input))
	return base64URLEncode(h.Sum(nil))
}

// Refresh generates a new token with the same claims but extended expiration.
func (j *JWT) Refresh(tokenString string) (string, error) {
	token, err := j.Parse(tokenString)
	if err != nil {
		// Allow refreshing expired tokens
		if !errors.Is(err, ErrExpiredToken) {
			return "", err
		}
		// Re-parse without validation for refresh
		token, err = j.parseWithoutValidation(tokenString)
		if err != nil {
			return "", err
		}
	}

	// Update timestamps
	now := time.Now()
	token.Claims.IssuedAt = now.Unix()
	token.Claims.ExpiresAt = now.Add(j.config.ExpiresIn).Unix()

	return j.Sign(token.Claims)
}

// parseWithoutValidation parses a token without validating claims.
func (j *JWT) parseWithoutValidation(tokenString string) (*Token, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	headerJSON, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	var header Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal header: %w", err)
	}

	claimsJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	expectedSignature := j.sign(signingInput)

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return nil, ErrInvalidSignature
	}

	return &Token{
		Header:    header,
		Claims:    claims,
		Signature: parts[2],
		Raw:       tokenString,
		Valid:     false, // Not validated
	}, nil
}

// base64URLEncode encodes data using base64url encoding.
func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// base64URLDecode decodes base64url encoded data.
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
