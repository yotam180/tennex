package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ChiMiddleware creates a Chi middleware for JWT authentication
func (c *JWTConfig) ChiMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("🔐 [JWT DEBUG] Processing request: %s %s\n", r.Method, r.URL.Path)

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			fmt.Printf("🔐 [JWT DEBUG] Authorization header: '%s'\n", authHeader)
			fmt.Printf("🔐 [JWT DEBUG] JWT secret length: %d bytes\n", len(c.Secret))
			fmt.Printf("🔐 [JWT DEBUG] JWT secret (first 10 chars): '%s...'\n", string(c.Secret)[:min(10, len(c.Secret))])

			tokenString, err := ExtractTokenFromHeader(authHeader)
			if err != nil {
				fmt.Printf("🔐 [JWT DEBUG] ❌ Token extraction failed: %v\n", err)
				writeAuthError(w, ErrMissingToken)
				return
			}

			fmt.Printf("🔐 [JWT DEBUG] ✅ Token extracted successfully (length: %d)\n", len(tokenString))
			fmt.Printf("🔐 [JWT DEBUG] Token (first 50 chars): '%s...'\n", tokenString[:min(50, len(tokenString))])

			// Validate token
			claims, err := c.ValidateToken(tokenString)
			if err != nil {
				fmt.Printf("🔐 [JWT DEBUG] ❌ Token validation failed: %v\n", err)
				writeAuthError(w, ErrInvalidToken)
				return
			}

			fmt.Printf("🔐 [JWT DEBUG] ✅ Token validated successfully\n")
			fmt.Printf("🔐 [JWT DEBUG] User ID from claims: %s\n", claims.UserID.String())
			fmt.Printf("🔐 [JWT DEBUG] Token expires at: %v\n", claims.ExpiresAt.Time)

			// Add user information to request context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, ClaimsKey, claims)

			fmt.Printf("🔐 [JWT DEBUG] ✅ Authentication successful, continuing to next handler\n")
			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalChiMiddleware creates a Chi middleware for optional JWT authentication
// Unlike the required middleware, this doesn't return an error if no token is provided
func (c *JWTConfig) OptionalChiMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			tokenString, err := ExtractTokenFromHeader(authHeader)
			if err != nil {
				// No token provided, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Validate token if provided
			claims, err := c.ValidateToken(tokenString)
			if err != nil {
				// Invalid token, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Add user information to request context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, ClaimsKey, claims)

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth is a Chi middleware that ensures a request is authenticated
// This should be used in combination with OptionalChiMiddleware
func RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if user is authenticated
			_, err := GetUserIDFromContext(r.Context())
			if err != nil {
				writeAuthError(w, ErrMissingToken)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeAuthError writes an authentication error response
func writeAuthError(w http.ResponseWriter, authErr *AuthError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(authErr.Status)

	response := map[string]interface{}{
		"error":     authErr.Message,
		"code":      authErr.Code,
		"timestamp": "2025-09-24T01:00:00Z", // TODO: Use actual timestamp
	}

	json.NewEncoder(w).Encode(response)
}

// GetUserIDFromRequest extracts user ID from a Chi request (convenience function)
func GetUserIDFromRequest(r *http.Request) (string, error) {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		return "", err
	}
	return userID.String(), nil
}

// Example usage for route protection:
//
// r := chi.NewRouter()
// jwtConfig := auth.DefaultJWTConfig("your-secret-key")
//
// // Public routes (no auth required)
// r.Get("/health", healthHandler)
//
// // Protected routes (auth required)
// r.Route("/api", func(r chi.Router) {
//     r.Use(jwtConfig.ChiMiddleware())  // All routes in this group require auth
//     r.Get("/profile", profileHandler)
//     r.Post("/data", dataHandler)
// })
//
// // Mixed routes (optional auth)
// r.Route("/public", func(r chi.Router) {
//     r.Use(jwtConfig.OptionalChiMiddleware())  // Auth is optional
//     r.Get("/content", contentHandler)        // Anyone can access
//     r.With(auth.RequireAuth()).Post("/comment", commentHandler)  // Auth required for this specific route
// })
