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

// GetOperationLogs returns a paginated list of operation logs.
func GetOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	username := c.Query("username")
	operation := c.Query("operation")
	resource := c.Query("resource")
	status := c.Query("status")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.OperationLog{})

	if username != "" {
		db = db.Where("username ILIKE ?", "%"+username+"%")
	}
	if operation != "" {
		db = db.Where("operation ILIKE ?", "%"+operation+"%")
	}
	if resource != "" {
		db = db.Where("resource ILIKE ?", "%"+resource+"%")
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if startDate != "" {
		db = db.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("created_at <= ?", endDate)
	}

	var total int64
	db.Count(&total)

	var logs []models.OperationLog
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"list": logs, "total": total},
		"msg":  "success",
	})
}

// GetOperationLogDetail returns a single operation log by UUID.
func GetOperationLogDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的日志ID"})
		return
	}

	var log models.OperationLog
	if err := database.DB.Where("id = ?", id).First(&log).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "日志不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": log, "msg": "success"})
}

// GetPermissions returns a paginated list of permissions.
func GetPermissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.Query("name")
	code := c.Query("code")

	offset := (page - 1) * pageSize

	db := database.DB.Model(&models.Permission{})

	if name != "" {
		db = db.Where("name ILIKE ?", "%"+name+"%")
	}
	if code != "" {
		db = db.Where("code ILIKE ?", "%"+code+"%")
	}

	var total int64
	db.Count(&total)

	var permissions []models.Permission
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&permissions)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"list": permissions, "total": total},
		"msg":  "success",
	})
}

// GetPermissionDetail returns a single permission by UUID.
func GetPermissionDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的权限ID"})
		return
	}

	var permission models.Permission
	if err := database.DB.Where("id = ?", id).First(&permission).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "权限不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询权限失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": permission, "msg": "success"})
}

// AddPermission creates a new permission.
func AddPermission(c *gin.Context) {
	var permission models.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var existingPermission models.Permission
	if err := database.DB.Where("code = ?", permission.Code).First(&existingPermission).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "权限代码已存在"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "检查权限失败"})
		return
	}

	if err := database.DB.Create(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "添加权限失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": permission, "msg": "success"})
}

// UpdatePermission updates an existing permission.
func UpdatePermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的权限ID"})
		return
	}

	var permission models.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	var existingPermission models.Permission
	if err := database.DB.Where("id = ?", id).First(&existingPermission).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "权限不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询权限失败"})
		return
	}

	if permission.Code != existingPermission.Code {
		if err := database.DB.Where("code = ? AND id != ?", permission.Code, id).First(&models.Permission{}).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "权限代码已存在"})
			return
		} else if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "检查权限失败"})
			return
		}
	}

	permission.ID = id
	permission.TenantID = existingPermission.TenantID
	if err := database.DB.Save(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新权限失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": permission, "msg": "success"})
}

// DeletePermission soft-deletes a permission and its role associations.
func DeletePermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的权限ID"})
		return
	}

	var permission models.Permission
	if err := database.DB.Where("id = ?", id).First(&permission).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "权限不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询权限失败"})
		return
	}

	if err := database.DB.Delete(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除权限失败"})
		return
	}

	database.DB.Where("permission_id = ?", id).Delete(&models.RolePermission{})

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

// BatchDeletePermissions soft-deletes multiple permissions.
func BatchDeletePermissions(c *gin.Context) {
	var request struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	if err := database.DB.Where("id IN ?", request.IDs).Delete(&models.Permission{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "批量删除权限失败"})
		return
	}

	database.DB.Where("permission_id IN ?", request.IDs).Delete(&models.RolePermission{})

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}

// GetRolePermissions returns all permissions assigned to a role.
func GetRolePermissions(c *gin.Context) {
	role := c.Param("role")

	var rolePermissions []models.RolePermission
	if err := database.DB.Where("role_id = ?", role).Find(&rolePermissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询角色权限失败"})
		return
	}

	permissionIDs := make([]uuid.UUID, 0, len(rolePermissions))
	for _, rp := range rolePermissions {
		permissionIDs = append(permissionIDs, rp.PermissionID)
	}

	var permissions []models.Permission
	if len(permissionIDs) > 0 {
		database.DB.Where("id IN ?", permissionIDs).Find(&permissions)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"list": permissions, "total": len(permissions)},
		"msg":  "success",
	})
}

// AssignRolePermissions replaces all permissions for a role.
func AssignRolePermissions(c *gin.Context) {
	role := c.Param("role")

	var request struct {
		PermissionIds []uuid.UUID `json:"permissionIds"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("role_id = ?", role).Delete(&models.RolePermission{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除现有角色权限失败"})
		return
	}

	for _, permissionID := range request.PermissionIds {
		rp := models.RolePermission{
			RoleID:       role,
			PermissionID: permissionID,
		}
		if err := tx.Create(&rp).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "分配角色权限失败"})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success"})
}
