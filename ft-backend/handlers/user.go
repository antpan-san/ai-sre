package handlers

import (
	"net/http"
	"strconv"

	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetUserProfile returns the profile of the authenticated user.
func GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	var user models.User
	if err := database.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": user, "msg": "success"})
}

// UpdateUserProfile updates the profile of the authenticated user.
func UpdateUserProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	var request struct {
		Phone    string `json:"phone"`
		FullName string `json:"full_name"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询用户失败"})
		return
	}

	if request.Phone != "" {
		user.Phone = request.Phone
	}
	if request.FullName != "" {
		user.FullName = request.FullName
	}
	if request.Avatar != "" {
		user.Avatar = request.Avatar
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": user, "msg": "success"})
}

// GetUserList returns a paginated list of users.
func GetUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	username := c.Query("username")
	role := c.Query("role")

	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.User{})

	if username != "" {
		db = db.Where("username ILIKE ?", "%"+username+"%")
	}
	if role != "" {
		db = db.Where("role = ?", role)
	}

	var total int64
	db.Count(&total)

	var users []models.User
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  users,
			"total": total,
		},
		"msg": "success",
	})
}

// GetUserDetail returns a single user by UUID.
func GetUserDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": user, "msg": "success"})
}

// AddUser creates a new user.
func AddUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("username = ?", user.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "用户名已存在"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "检查用户名失败"})
		return
	}

	if err := database.DB.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "邮箱已存在"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "检查邮箱失败"})
		return
	}

	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "密码加密失败"})
		return
	}
	user.Password = hashedPassword

	if user.Role == "" {
		user.Role = "user"
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "添加用户失败"})
		return
	}

	user.Password = ""

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": user, "msg": "success"})
}

// UpdateUser updates an existing user.
func UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	var request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Role     string `json:"role"`
		FullName string `json:"full_name"`
		Avatar   string `json:"avatar"`
		Password string `json:"password,omitempty"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询用户失败"})
		return
	}

	if request.Username != "" && request.Username != user.Username {
		var existingUser models.User
		if err := database.DB.Where("username = ?", request.Username).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "用户名已存在"})
			return
		} else if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "检查用户名失败"})
			return
		}
		user.Username = request.Username
	}

	if request.Email != "" && request.Email != user.Email {
		var existingUser models.User
		if err := database.DB.Where("email = ?", request.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "邮箱已存在"})
			return
		} else if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "检查邮箱失败"})
			return
		}
		user.Email = request.Email
	}

	if request.Phone != "" {
		user.Phone = request.Phone
	}
	if request.Role != "" {
		user.Role = request.Role
	}
	if request.FullName != "" {
		user.FullName = request.FullName
	}
	if request.Avatar != "" {
		user.Avatar = request.Avatar
	}
	if request.Password != "" {
		hashedPassword, err := utils.HashPassword(request.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "密码加密失败"})
			return
		}
		user.Password = hashedPassword
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新用户失败"})
		return
	}

	user.Password = ""

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": user, "msg": "success"})
}

// DeleteUser soft-deletes a user.
func DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询用户失败"})
		return
	}

	if user.Role == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "不能删除管理员用户"})
		return
	}

	if err := database.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

// BatchDeleteUser soft-deletes multiple users.
func BatchDeleteUser(c *gin.Context) {
	var request struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var adminCount int64
	database.DB.Model(&models.User{}).Where("id IN ? AND role = ?", request.IDs, "admin").Count(&adminCount)
	if adminCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "不能删除管理员用户"})
		return
	}

	if err := database.DB.Where("id IN ?", request.IDs).Delete(&models.User{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "批量删除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

// UpdateUserRole updates only the role of a user.
func UpdateUserRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	var request struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询用户失败"})
		return
	}

	user.Role = request.Role

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新用户角色失败"})
		return
	}

	user.Password = ""

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": user, "msg": "success"})
}
