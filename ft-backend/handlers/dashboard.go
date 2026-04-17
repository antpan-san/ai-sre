package handlers

import (
	"net/http"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
)

// GetDashboardData 获取仪表盘数据
func GetDashboardData(c *gin.Context) {
	// 获取用户ID（虽然目前未使用，但保留获取逻辑以便未来扩展）
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	// 这里可以添加实际的仪表盘数据逻辑
	// 为了演示目的，我们返回一些模拟数据

	// 获取机器统计数据
	var totalMachines int64
	var onlineMachines int64
	var offlineMachines int64

	database.DB.Model(&models.Machine{}).Count(&totalMachines)
	database.DB.Model(&models.Machine{}).Where("status = ?", "online").Count(&onlineMachines)
	database.DB.Model(&models.Machine{}).Where("status = ?", "offline").Count(&offlineMachines)

	// 获取用户统计数据
	var totalUsers int64
	database.DB.Model(&models.User{}).Count(&totalUsers)

	// 获取操作日志统计数据
	var totalLogs int64
	database.DB.Model(&models.OperationLog{}).Count(&totalLogs)

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取仪表盘数据成功",
		"data": gin.H{
			"machines": gin.H{
				"total":  totalMachines,
				"online": onlineMachines,
				"offline": offlineMachines,
			},
			"users": gin.H{
				"total": totalUsers,
			},
			"logs": gin.H{
				"total": totalLogs,
			},
		},
	})
}