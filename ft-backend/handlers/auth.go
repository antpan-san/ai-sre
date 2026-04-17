package handlers

import (
	"net/http"

	"ft-backend/common/config"
	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"omitempty,max=100"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register handles user registration.
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid input", "error": err.Error()})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "Username already exists"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Database error"})
		return
	}

	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "Email already exists"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Database error"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to hash password"})
		return
	}

	newUser := models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
		Role:     "user",
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "User registered successfully",
		"user": gin.H{
			"id":        newUser.ID,
			"username":  newUser.Username,
			"email":     newUser.Email,
			"full_name": newUser.FullName,
			"role":      newUser.Role,
		},
	})
}

// Login handles user login, returning a JWT token.
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid input", "error": err.Error()})
		return
	}

	cfg := c.MustGet("config").(*config.Config)

	var user models.User
	// 当前系统尚未在登录链路中引入 tenant 上下文，这里先固定到默认租户，
	// 避免多租户同名账号导致命中错误记录。
	if err := database.DB.Where("username = ? AND tenant_id = ?", req.Username, models.DefaultTenantID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误", "msg": "用户名或密码错误"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Database error", "msg": "数据库错误"})
		}
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误", "msg": "用户名或密码错误"})
		return
	}

	// user.ID is uuid.UUID; pass its string representation to the JWT generator
	accessToken, err := utils.GenerateAccessToken(
		user.ID.String(),
		user.Username,
		user.Email,
		user.Role,
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenExp,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to generate access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"token": accessToken,
			"user": gin.H{
				"id":         user.ID,
				"username":   user.Username,
				"email":      user.Email,
				"phone":      user.Phone,
				"role":       user.Role,
				"full_name":  user.FullName,
				"avatar":     user.Avatar,
				"createTime": user.CreatedAt,
				"updateTime": user.UpdatedAt,
			},
		},
		"msg": "success",
	})
}

// RefreshToken handles token refresh.
func RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid input", "error": err.Error()})
		return
	}

	cfg := c.MustGet("config").(*config.Config)

	claims, err := utils.ValidateToken(req.RefreshToken, cfg.JWT.SecretKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Invalid refresh token"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Database error"})
		}
		return
	}

	newAccessToken, err := utils.GenerateAccessToken(
		user.ID.String(),
		user.Username,
		user.Email,
		user.Role,
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenExp,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to generate access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Token refreshed successfully",
		"data": gin.H{
			"access_token": newAccessToken,
			"token_type":   "Bearer",
			"expires_in":   cfg.JWT.AccessTokenExp * 60,
		},
	})
}

// Logout handles user logout.
func Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": nil,
		"msg":  "success",
	})
}
