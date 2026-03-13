package user

import (
	"backend/utils"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	UserIDKey   = "user_id"
	UsernameKey = "username"
)

// AuthMiddleware validates JWT token from Authorization header
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			slog.Warn("missing authorization header")
			utils.RespError(c, 401, "missing authorization header")
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			slog.Warn("invalid authorization header format")
			utils.RespError(c, 401, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		userID, username, err := ValidateToken(tokenString)
		if err != nil {
			slog.Warn("invalid token", slog.Any("error", err))
			utils.RespError(c, 401, "invalid or expired token")
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(UserIDKey, userID)
		c.Set(UsernameKey, username)

		c.Next()
	}
}

// GetUserID extracts user ID from context (requires AuthMiddleware)
func GetUserID(c *gin.Context) (uint64, bool) {
	val, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	userID, ok := val.(uint64)
	return userID, ok
}

// GetUsername extracts username from context (requires AuthMiddleware)
func GetUsername(c *gin.Context) (string, bool) {
	val, exists := c.Get(UsernameKey)
	if !exists {
		return "", false
	}
	username, ok := val.(string)
	return username, ok
}
