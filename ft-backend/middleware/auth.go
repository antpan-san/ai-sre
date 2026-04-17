package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"ft-backend/common/logger"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// JWTAuth is a JWT authentication middleware.
// It extracts the Bearer token, validates it, and stores user info in the context.
// The userID is stored as uuid.UUID in the context.
func JWTAuth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Debug("Processing request: %s %s", c.Request.Method, c.Request.URL.Path)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Authorization header is required",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Authorization header format must be Bearer {token}",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := utils.ValidateToken(tokenString, secretKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  fmt.Sprintf("Invalid or expired token: %v", err),
			})
			c.Abort()
			return
		}

		// Parse the UUID string into uuid.UUID for type safety
		userUUID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Invalid user ID in token",
			})
			c.Abort()
			return
		}

		logger.Debug("Token valid. UserID: %s, Username: %s, Role: %s", claims.UserID, claims.Username, claims.Role)

		// Store user info in context (userID as uuid.UUID)
		c.Set("userID", userUUID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}
