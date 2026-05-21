package handlers

import (
	"fmt"
	"strings"

	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---- Service CRUD ----

// DeployService creates a new service deployment.
func DeployService(c *gin.Context) {
	_ = reconcileStaleDeployingServices(database.DB)
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
	_ = reconcileStaleDeployingServices(database.DB)
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
	_ = reconcileStaleDeployingServices(database.DB)
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

// GetLinuxServiceList dispatches a read-only systemd query task to a target machine.
func GetLinuxServiceList(c *gin.Context) {
	machineID := strings.TrimSpace(c.Query("machine_id"))
	if machineID == "" {
		machineID = strings.TrimSpace(c.Query("machineId"))
	}
	if machineID == "" {
		response.OK(c, gin.H{"list": []interface{}{}, "total": 0, "message": "请选择机器后查询 systemd 服务"})
		return
	}
	script := `set -euo pipefail
if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl not found" >&2
  exit 1
fi
systemctl list-units --type=service --all --no-pager --plain \
  | awk 'NR>1 && $1 ~ /\.service$/ {printf "%s\t%s\t%s\n",$1,$4,$5}' \
  | head -500
`
	task, err := createManagedTask(c, managedTaskRequest{
		Name:        "Linux 服务列表查询",
		Type:        string(models.TaskTypeShell),
		Command:     "run_shell",
		Description: "只读查询 systemd 服务列表",
		MachineIDs:  []string{machineID},
		Payload:     map[string]interface{}{"script": script, "readonly": true, "purpose": "linux_service_list"},
		TimeoutSec:  60,
		MaxRetry:    1,
	})
	if err != nil {
		response.ServerError(c, "创建服务查询任务失败")
		return
	}
	response.OK(c, gin.H{"list": []interface{}{}, "total": 0, "task_id": task.ID.String(), "status": task.Status})
}

// OperateLinuxService dispatches a systemctl operation task to the target machine.
func OperateLinuxService(c *gin.Context) {
	var req struct {
		MachineID string `json:"machine_id"`
		Service   string `json:"service"`
		Action    string `json:"action"` // start, stop, restart, enable, disable
		ServiceID string `json:"serviceId"`
		Operation string `json:"operation"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	if req.Action == "" {
		req.Action = req.Operation
	}
	if req.Service == "" {
		req.Service = req.ServiceID
	}
	if strings.TrimSpace(req.MachineID) == "" {
		response.BadRequest(c, "缺少机器ID")
		return
	}
	if !isAllowedSystemctlAction(req.Action) {
		response.BadRequest(c, "不支持的 systemctl 操作")
		return
	}
	if !isSafeSystemdUnit(req.Service) {
		response.BadRequest(c, "非法的服务名称")
		return
	}

	script := fmt.Sprintf("set -euo pipefail\nsystemctl %s %s\nsystemctl --no-pager --plain status %s || true\n",
		req.Action, shellQuote(req.Service), shellQuote(req.Service))
	task, err := createManagedTask(c, managedTaskRequest{
		Name:        "Linux 服务操作: " + req.Service,
		Type:        string(models.TaskTypeShell),
		Command:     "run_shell",
		Description: "systemctl " + req.Action + " " + req.Service,
		MachineIDs:  []string{req.MachineID},
		Payload:     map[string]interface{}{"script": script, "service": req.Service, "action": req.Action},
		TimeoutSec:  120,
		MaxRetry:    1,
	})
	if err != nil {
		response.ServerError(c, "创建服务操作任务失败")
		return
	}
	response.OK(c, gin.H{"task_id": task.ID.String(), "status": task.Status})
}

func isAllowedSystemctlAction(action string) bool {
	switch action {
	case "start", "stop", "restart", "enable", "disable":
		return true
	default:
		return false
	}
}

func isSafeSystemdUnit(name string) bool {
	if strings.TrimSpace(name) == "" || len(name) > 160 {
		return false
	}
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') ||
			ch == '.' || ch == '_' || ch == '-' || ch == '@' {
			continue
		}
		return false
	}
	return true
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
