package handlers

import (
	"net/http"

	"ft-backend/common/config"
	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"omitempty,max=100"`
}

type LoginRequest struct {
	Username       string `json:"username" binding:"required"`
	Password       string `json:"password" binding:"required"`
	CaptchaID      string `json:"captcha_id"`
	CaptchaAnswer  string `json:"captcha_answer"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// PublicAuthOptions exposes which auth features are enabled (for login / register UI).
func PublicAuthOptions(c *gin.Context) {
	cfg := c.MustGet("config").(*config.Config)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"public_registration_allowed": cfg.Security.PublicRegistrationAllowed(),
			"login_captcha_required":      cfg.Security.LoginCaptchaRequired(),
		},
		"msg": "success",
	})
}

// GetLoginCaptcha issues a one-time numeric captcha for login.
func GetLoginCaptcha(c *gin.Context) {
	cfg := c.MustGet("config").(*config.Config)
	if !cfg.Security.LoginCaptchaRequired() {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"captcha_id":      "",
				"challenge":       "",
				"captcha_skipped": true,
			},
			"msg": "success",
		})
		return
	}
	id, challenge := issueLoginCaptcha()
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"captcha_id": id,
			"challenge":  challenge,
		},
		"msg": "success",
	})
}

// Register handles user registration.
func Register(c *gin.Context) {
	cfg := c.MustGet("config").(*config.Config)
	if !cfg.Security.PublicRegistrationAllowed() {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "管理员已关闭公开注册，请联系管理员开通账号"})
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请求参数无效", "message": err.Error()})
		return
	}

	tid := uuid.MustParse(models.DefaultTenantID)

	var existingUser models.User
	if err := database.DB.Where("username = ? AND tenant_id = ?", req.Username, tid).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": "用户名已被占用"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "数据库错误"})
		return
	}

	if err := database.DB.Where("email = ? AND tenant_id = ?", req.Email, tid).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": "邮箱已被占用"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "数据库错误"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "密码处理失败"})
		return
	}

	newUser := models.User{
		SoftDeleteModel: models.SoftDeleteModel{
			BaseModel: models.BaseModel{TenantID: tid},
		},
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
		Role:     "user",
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建用户失败"})
		return
	}

	newUser.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"user": newUser,
		},
		"msg": "注册成功，请使用账号登录",
	})
}

// Login handles user login, returning a JWT token.
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请求参数无效", "message": err.Error()})
		return
	}

	cfg := c.MustGet("config").(*config.Config)

	if cfg.Security.LoginCaptchaRequired() {
		if !verifyAndConsumeLoginCaptcha(req.CaptchaID, req.CaptchaAnswer) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "验证码错误或已过期，请刷新验证码后重试"})
			return
		}
	}

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
