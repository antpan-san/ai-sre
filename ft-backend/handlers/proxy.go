package handlers

import (
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetProxyConfigList returns all proxy configs.
func GetProxyConfigList(c *gin.Context) {
	p := response.GetPagination(c)
	db := database.DB.Model(&models.ProxyConfig{})

	name := c.Query("name")
	if name != "" {
		db = db.Where("name ILIKE ?", "%"+name+"%")
	}

	var total int64
	db.Count(&total)

	var configs []models.ProxyConfig
	response.Paginate(db, p, "").Find(&configs)
	response.OKPage(c, configs, total)
}

// GetProxyConfigDetail returns a single proxy config.
func GetProxyConfigDetail(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		idStr = c.Param("id")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "无效的配置ID")
		return
	}
	var cfg models.ProxyConfig
	if response.HandleDBError(c, database.DB.Where("id = ?", id).First(&cfg).Error, "配置不存在") {
		return
	}
	response.OK(c, cfg)
}

// SaveProxyConfig creates or updates a proxy config.
func SaveProxyConfig(c *gin.Context) {
	var cfg models.ProxyConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if cfg.ID != uuid.Nil {
		// Update
		var existing models.ProxyConfig
		if response.HandleDBError(c, database.DB.Where("id = ?", cfg.ID).First(&existing).Error, "配置不存在") {
			return
		}
		cfg.TenantID = existing.TenantID
		if err := database.DB.Save(&cfg).Error; err != nil {
			response.ServerError(c, "保存配置失败")
			return
		}
	} else {
		// Create
		if err := database.DB.Create(&cfg).Error; err != nil {
			response.ServerError(c, "创建配置失败")
			return
		}
	}
	response.OK(c, cfg)
}

// DeleteProxyConfig soft-deletes a proxy config.
func DeleteProxyConfig(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		response.BadRequest(c, "无效的配置ID")
		return
	}
	if err := database.DB.Where("id = ?", id).Delete(&models.ProxyConfig{}).Error; err != nil {
		response.ServerError(c, "删除配置失败")
		return
	}
	response.OKMsg(c, "删除成功")
}

// ApplyProxyConfig applies a proxy configuration to its target machine.
func ApplyProxyConfig(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		response.BadRequest(c, "无效的配置ID")
		return
	}

	var cfg models.ProxyConfig
	if response.HandleDBError(c, database.DB.Where("id = ?", id).First(&cfg).Error, "配置不存在") {
		return
	}

	// Mark as active
	database.DB.Model(&cfg).Update("status", "active")
	cfg.Status = "active"

	// TODO: dispatch apply task to agent on the target machine

	response.OKMsg(c, "配置已应用")
}
