package handlers

import (
	"net/http"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DebugGetToken returns a test token. REMOVE IN PRODUCTION.
func DebugGetToken(c *gin.Context) {
	logger.Debug("DebugGetToken called")

	cfg := c.MustGet("config").(*config.Config)

	// Use a deterministic UUID for the debug admin user
	debugUserID := uuid.MustParse("00000000-0000-0000-0000-000000000099")

	token, err := utils.GenerateAccessToken(
		debugUserID.String(),
		"admin",
		"admin@example.com",
		"admin",
		cfg.JWT.SecretKey,
		60,
	)

	if err != nil {
		logger.Error("Error generating token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "生成token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取调试token成功",
		"data": gin.H{
			"token": token,
			"usage": "在Authorization头中使用: Bearer " + token,
		},
	})
}

// DebugTestAuth tests JWT authentication. REMOVE IN PRODUCTION.
func DebugTestAuth(c *gin.Context) {
	logger.Debug("DebugTestAuth called")

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户未认证"})
		return
	}

	username, _ := c.Get("username")
	email, _ := c.Get("email")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "认证成功",
		"data": gin.H{
			"userID":   userID,
			"username": username,
			"email":    email,
			"role":     role,
		},
	})
}
