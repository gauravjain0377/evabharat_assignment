// middleware/auth - JWT authentication middleware.
// Protects routes by validating the Authorization header and injecting user info into the request context.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gauravjain0377/ticket-system/utils"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UsernameKey contextKey = "username"
)

// AuthMiddleware returns a middleware that validates JWT tokens on protected routes.
// It expects the header: Authorization: Bearer <token>
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.Error(w, http.StatusUnauthorized, "authorization header is required")
				return
			}

			// Expect format: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.Error(w, http.StatusUnauthorized, "authorization header must be: Bearer <token>")
				return
			}

			tokenString := parts[1]

			// Validate the token and extract claims
			claims, err := utils.ValidateToken(tokenString, jwtSecret)
			if err != nil {
				utils.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// Inject user info into the request context so handlers can access it
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)

			// Pass the request to the next handler with the enriched context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the request context (set by AuthMiddleware).
func GetUserID(r *http.Request) int64 {
	userID, ok := r.Context().Value(UserIDKey).(int64)
	if !ok {
		return 0
	}
	return userID
}
