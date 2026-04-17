package handlers

import (
	"net/http"
	"strconv"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- Monitoring Configs ----

// GetMonitoringConfigList returns all monitoring configurations.
func GetMonitoringConfigList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	configType := c.Query("type")

	offset := (page - 1) * pageSize
	db := database.DB.Model(&models.MonitoringConfig{})

	if configType != "" {
		db = db.Where("type = ?", configType)
	}

	var total int64
	db.Count(&total)

	var configs []models.MonitoringConfig
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&configs)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  configs,
			"total": total,
		},
		"msg": "success",
	})
}

// GetMonitoringConfig returns a single monitoring configuration.
func GetMonitoringConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的配置ID"})
		return
	}

	var config models.MonitoringConfig
	if err := database.DB.Where("id = ?", id).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "配置不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": config, "msg": "success"})
}

// CreateMonitoringConfig creates a new monitoring configuration.
func CreateMonitoringConfig(c *gin.Context) {
	var config models.MonitoringConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	if err := database.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": config, "msg": "创建成功"})
}

// UpdateMonitoringConfig updates an existing monitoring configuration.
func UpdateMonitoringConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的配置ID"})
		return
	}

	var existing models.MonitoringConfig
	if err := database.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "配置不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询配置失败"})
		return
	}

	var config models.MonitoringConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	config.ID = id
	config.TenantID = existing.TenantID
	if err := database.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": config, "msg": "更新成功"})
}

// DeleteMonitoringConfig deletes a monitoring configuration.
func DeleteMonitoringConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的配置ID"})
		return
	}

	if err := database.DB.Where("id = ?", id).Delete(&models.MonitoringConfig{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}

// ---- Alert Rules ----

// GetAlertRules returns all alert rules.
func GetAlertRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	offset := (page - 1) * pageSize
	db := database.DB.Model(&models.AlertRule{})

	var total int64
	db.Count(&total)

	var rules []models.AlertRule
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&rules)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  rules,
			"total": total,
		},
		"msg": "success",
	})
}

// CreateAlertRule creates a new alert rule.
func CreateAlertRule(c *gin.Context) {
	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建告警规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": rule, "msg": "创建成功"})
}

// UpdateAlertRule updates an existing alert rule.
func UpdateAlertRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的规则ID"})
		return
	}

	var existing models.AlertRule
	if err := database.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "规则不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询规则失败"})
		return
	}

	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	rule.ID = id
	rule.TenantID = existing.TenantID
	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": rule, "msg": "更新成功"})
}

// DeleteAlertRule deletes an alert rule.
func DeleteAlertRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的规则ID"})
		return
	}

	if err := database.DB.Where("id = ?", id).Delete(&models.AlertRule{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}
