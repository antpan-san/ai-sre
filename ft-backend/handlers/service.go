package handlers

import (
	"net/http"

	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---- Service CRUD ----

// DeployService creates a new service deployment.
func DeployService(c *gin.Context) {
	var svc models.Service
	if err := c.ShouldBindJSON(&svc); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	if svc.Status == "" {
		svc.Status = "deploying"
	}
	if err := database.DB.Create(&svc).Error; err != nil {
		response.ServerError(c, "创建服务失败")
		return
	}
	response.OK(c, svc)
}

// GetServiceList returns services.
func GetServiceList(c *gin.Context) {
	p := response.GetPagination(c)
	db := database.DB.Model(&models.Service{})

	name := c.Query("name")
	status := c.Query("status")
	if name != "" {
		db = db.Where("name ILIKE ?", "%"+name+"%")
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}

	var total int64
	db.Count(&total)

	var services []models.Service
	response.Paginate(db, p, "").Find(&services)
	response.OKPage(c, services, total)
}

// GetServiceDetail returns a service by ID.
func GetServiceDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Query("id"))
	if err != nil {
		response.BadRequest(c, "无效的服务ID")
		return
	}
	var svc models.Service
	if response.HandleDBError(c, database.DB.Where("id = ?", id).First(&svc).Error, "服务不存在") {
		return
	}
	response.OK(c, svc)
}

// ServiceAction starts/stops/restarts a service by changing its status.
func ServiceAction(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ID string `json:"id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "无效的请求参数")
			return
		}
		id, err := uuid.Parse(req.ID)
		if err != nil {
			response.BadRequest(c, "无效的服务ID")
			return
		}

		var svc models.Service
		if response.HandleDBError(c, database.DB.Where("id = ?", id).First(&svc).Error, "服务不存在") {
			return
		}

		var newStatus string
		switch action {
		case "start":
			newStatus = "running"
		case "stop":
			newStatus = "stopped"
		case "restart":
			newStatus = "running"
		}

		database.DB.Model(&svc).Update("status", newStatus)
		svc.Status = newStatus
		response.OK(c, svc)
	}
}

// DeleteService soft-deletes a service.
func DeleteService(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		response.BadRequest(c, "无效的服务ID")
		return
	}
	if err := database.DB.Where("id = ?", id).Delete(&models.Service{}).Error; err != nil {
		response.ServerError(c, "删除服务失败")
		return
	}
	response.OKMsg(c, "删除成功")
}

// BatchDeleteService soft-deletes multiple services.
func BatchDeleteService(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	if err := database.DB.Where("id IN ?", req.IDs).Delete(&models.Service{}).Error; err != nil {
		response.ServerError(c, "批量删除失败")
		return
	}
	response.OKMsg(c, "删除成功")
}

// ---- Linux Service Management ----

// GetLinuxServiceList returns systemd services from a target machine (mock for now).
func GetLinuxServiceList(c *gin.Context) {
	// In production, this would query the machine via task system
	services := []map[string]interface{}{
		{"name": "sshd", "status": "running", "enabled": true},
		{"name": "nginx", "status": "stopped", "enabled": true},
		{"name": "docker", "status": "running", "enabled": true},
		{"name": "firewalld", "status": "running", "enabled": true},
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": services, "msg": "success"})
}

// OperateLinuxService starts/stops/restarts a Linux service.
func OperateLinuxService(c *gin.Context) {
	var req struct {
		MachineID string `json:"machine_id" binding:"required"`
		Service   string `json:"service" binding:"required"`
		Action    string `json:"action" binding:"required"` // start, stop, restart, enable, disable
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	// TODO: dispatch as a task to the agent on the target machine
	response.OKMsg(c, "操作指令已下发")
}
