// Package auth provides shared JWT authentication utilities for Tennex services
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Claims represents the JWT token claims
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret []byte
	TTL    time.Duration
}

// NewJWTConfig creates a new JWT configuration
func NewJWTConfig(secret string, ttl time.Duration) *JWTConfig {
	return &JWTConfig{
		Secret: []byte(secret),
		TTL:    ttl,
	}
}

// DefaultJWTConfig returns a default JWT configuration (24 hour TTL)
func DefaultJWTConfig(secret string) *JWTConfig {
	return NewJWTConfig(secret, 24*time.Hour)
}

// GenerateToken generates a new JWT token for the given user ID
func (c *JWTConfig) GenerateToken(userID uuid.UUID) (string, time.Time, error) {
	fmt.Printf("🔥 [TOKEN GEN DEBUG] Generating token for user: %s\n", userID.String())

	now := time.Now()
	expirationTime := now.Add(c.TTL)

	fmt.Printf("🔥 [TOKEN GEN DEBUG] Current time: %v\n", now)
	fmt.Printf("🔥 [TOKEN GEN DEBUG] Expiration time: %v\n", expirationTime)
	fmt.Printf("🔥 [TOKEN GEN DEBUG] TTL: %v\n", c.TTL)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	fmt.Printf("🔥 [TOKEN GEN DEBUG] Claims created: %+v\n", claims)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	fmt.Printf("🔥 [TOKEN GEN DEBUG] JWT secret length: %d\n", len(c.Secret))
	fmt.Printf("🔥 [TOKEN GEN DEBUG] JWT secret (first 10): %s\n", string(c.Secret)[:min(10, len(c.Secret))])

	tokenString, err := token.SignedString(c.Secret)
	if err != nil {
		fmt.Printf("🔥 [TOKEN GEN DEBUG] ❌ Failed to sign token: %v\n", err)
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	fmt.Printf("🔥 [TOKEN GEN DEBUG] ✅ Token generated successfully (length: %d)\n", len(tokenString))
	fmt.Printf("🔥 [TOKEN GEN DEBUG] Token (first 50): %s\n", tokenString[:min(50, len(tokenString))])

	return tokenString, expirationTime, nil
}

// ValidateToken validates and parses a JWT token, returning the claims
func (c *JWTConfig) ValidateToken(tokenString string) (*Claims, error) {
	fmt.Printf("🔍 [TOKEN DEBUG] Starting token validation\n")
	fmt.Printf("🔍 [TOKEN DEBUG] Token string length: %d\n", len(tokenString))

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		fmt.Printf("🔍 [TOKEN DEBUG] Token header: %+v\n", token.Header)
		fmt.Printf("🔍 [TOKEN DEBUG] Token method: %v\n", token.Method)

		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			fmt.Printf("🔍 [TOKEN DEBUG] ❌ Unexpected signing method: %v\n", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		fmt.Printf("🔍 [TOKEN DEBUG] ✅ Signing method is HMAC\n")
		fmt.Printf("🔍 [TOKEN DEBUG] Using secret length: %d\n", len(c.Secret))
		return c.Secret, nil
	})

	if err != nil {
		fmt.Printf("🔍 [TOKEN DEBUG] ❌ Token parsing failed: %v\n", err)
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		fmt.Printf("🔍 [TOKEN DEBUG] ❌ Token is not valid\n")
		return nil, fmt.Errorf("invalid token")
	}

	fmt.Printf("🔍 [TOKEN DEBUG] ✅ Token is valid\n")
	fmt.Printf("🔍 [TOKEN DEBUG] Parsed claims: %+v\n", claims)
	fmt.Printf("🔍 [TOKEN DEBUG] User ID: %s\n", claims.UserID.String())

	if claims.ExpiresAt != nil {
		fmt.Printf("🔍 [TOKEN DEBUG] Expires at: %v\n", claims.ExpiresAt.Time)
	}
	if claims.IssuedAt != nil {
		fmt.Printf("🔍 [TOKEN DEBUG] Issued at: %v\n", claims.IssuedAt.Time)
	}

	return claims, nil
}

// ExtractTokenFromHeader extracts Bearer token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

// UserContextKey is the context key for storing user information
type UserContextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey UserContextKey = "user_id"
	// ClaimsKey is the context key for JWT claims
	ClaimsKey UserContextKey = "jwt_claims"
)

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

// GetClaimsFromContext extracts JWT claims from request context
func GetClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(ClaimsKey).(*Claims)
	if !ok {
		return nil, fmt.Errorf("JWT claims not found in context")
	}
	return claims, nil
}

// AuthError represents authentication/authorization errors
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// Common authentication errors
var (
	ErrMissingToken = &AuthError{
		Code:    "missing_token",
		Message: "Authorization token is required",
		Status:  http.StatusUnauthorized,
	}
	ErrInvalidToken = &AuthError{
		Code:    "invalid_token",
		Message: "Invalid or expired token",
		Status:  http.StatusUnauthorized,
	}
	ErrTokenExpired = &AuthError{
		Code:    "token_expired",
		Message: "Token has expired",
		Status:  http.StatusUnauthorized,
	}
)
