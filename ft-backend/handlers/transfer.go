package handlers

import (
	"net/http"
	"strconv"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
)

// GetTransferHistory returns file transfer records for the authenticated user.
func GetTransferHistory(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	offset := (page - 1) * pageSize

	var transfers []models.Transfer
	var total int64

	db := database.DB.Model(&models.Transfer{}).Where("user_id = ?", uid)

	if transferType := c.Query("type"); transferType != "" {
		db = db.Where("type = ?", transferType)
	}
	if status := c.Query("status"); status != "" {
		db = db.Where("status = ?", status)
	}

	db.Count(&total)

	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&transfers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取传输记录失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取传输记录成功",
		"data": gin.H{"list": transfers, "total": total},
	})
}
