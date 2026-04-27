package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/iotservice"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- Task CRUD ----

// CreateTask creates a new task and splits it into sub-tasks for each target machine.
func CreateTask(c *gin.Context) {
	var req struct {
		Name        string          `json:"name" binding:"required"`
		Type        string          `json:"type" binding:"required"`
		Description string          `json:"description"`
		Payload     json.RawMessage `json:"payload"`
		TargetIDs   []string        `json:"target_ids" binding:"required"`
		Priority    int             `json:"priority"`
		TimeoutSec  int             `json:"timeout_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数", "error": err.Error()})
		return
	}

	username, _ := c.Get("username")

	if req.TimeoutSec <= 0 {
		req.TimeoutSec = 300
	}

	targetIDsJSON, _ := json.Marshal(req.TargetIDs)

	task := models.Task{
		Name:        req.Name,
		Type:        req.Type,
		Status:      string(models.TaskStatusPending),
		Priority:    req.Priority,
		CreatedBy:   username.(string),
		Description: req.Description,
		Payload:     models.JSONB(req.Payload),
		TargetIDs:   models.JSONB(targetIDsJSON),
		TotalCount:  len(req.TargetIDs),
		TimeoutSec:  req.TimeoutSec,
	}

	// Start transaction to create task + sub-tasks atomically
	tx := database.DB.Begin()
	if err := tx.Create(&task).Error; err != nil {
		tx.Rollback()
		logger.Error("创建任务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败"})
		return
	}

	// Create sub-tasks for each target machine
	for _, targetID := range req.TargetIDs {
		machineUUID, err := uuid.Parse(targetID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的机器ID: " + targetID})
			return
		}

		// Look up the machine to get its client_id (from heartbeat records)
		var machine models.Machine
		if err := tx.Where("id = ?", machineUUID).First(&machine).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "机器不存在: " + targetID})
			return
		}

		// Determine the command type based on task type
		command := mapTaskTypeToCommand(req.Type)

		// Prefer client_id (agent identity) over IP for task dispatch
		clientID := machine.ClientID
		if clientID == "" {
			clientID = machine.IP // Fallback for machines not yet heartbeated
		}

		subTask := models.SubTask{
			TaskID:    task.ID,
			MachineID: machineUUID,
			ClientID:  clientID,
			Command:   command,
			Status:    string(models.TaskStatusPending),
			Payload:   models.JSONB(req.Payload),
			MaxRetry:  3,
		}
		if err := tx.Create(&subTask).Error; err != nil {
			tx.Rollback()
			logger.Error("创建子任务失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建子任务失败"})
			return
		}
	}

	// Record task log
	taskLog := models.TaskLog{
		TaskID:   task.ID,
		ClientID: "",
		Level:    "info",
		Message:  "任务已创建，共" + strconv.Itoa(len(req.TargetIDs)) + "个子任务",
	}
	tx.Create(&taskLog)

	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": task, "msg": "任务创建成功"})
}

// GetTaskList returns a paginated list of tasks.
func GetTaskList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	taskType := c.Query("type")
	status := c.Query("status")
	name := c.Query("name")

	offset := (page - 1) * pageSize
	db := database.DB.Model(&models.Task{})

	if taskType != "" {
		db = db.Where("type = ?", taskType)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if name != "" {
		db = db.Where("name ILIKE ?", "%"+name+"%")
	}

	var total int64
	db.Count(&total)

	var tasks []models.Task
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&tasks)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  tasks,
			"total": total,
		},
		"msg": "success",
	})
}

// GetTaskDetail returns task with its sub-tasks.
func GetTaskDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的任务ID"})
		return
	}

	var task models.Task
	if err := database.DB.Where("id = ?", id).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "任务不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询任务失败"})
		return
	}

	var subTasks []models.SubTask
	database.DB.Where("task_id = ?", id).Order("created_at ASC").Find(&subTasks)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"task":      task,
			"sub_tasks": subTasks,
		},
		"msg": "success",
	})
}

// CancelTask cancels a pending/dispatched task and its sub-tasks.
func CancelTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的任务ID"})
		return
	}

	var task models.Task
	if err := database.DB.Where("id = ?", id).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "任务不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询任务失败"})
		return
	}

	// Only pending/dispatched tasks can be cancelled
	if task.Status != string(models.TaskStatusPending) && task.Status != string(models.TaskStatusDispatched) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "只有待处理或已下发的任务可以取消"})
		return
	}

	now := time.Now()
	tx := database.DB.Begin()
	tx.Model(&task).Updates(map[string]interface{}{
		"status":      string(models.TaskStatusCancelled),
		"finished_at": now,
	})
	tx.Model(&models.SubTask{}).Where("task_id = ? AND status IN ?", id,
		[]string{string(models.TaskStatusPending), string(models.TaskStatusDispatched)}).
		Updates(map[string]interface{}{
			"status":      string(models.TaskStatusCancelled),
			"finished_at": now,
		})
	tx.Create(&models.TaskLog{
		TaskID:  id,
		Level:   "warn",
		Message: "任务已被取消",
	})
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "任务已取消"})
}

// GetTaskLogs returns execution logs for a task.
func GetTaskLogs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的任务ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	offset := (page - 1) * pageSize

	var total int64
	database.DB.Model(&models.TaskLog{}).Where("task_id = ?", id).Count(&total)

	var logs []models.TaskLog
	database.DB.Where("task_id = ?", id).
		Limit(pageSize).Offset(offset).
		Order("created_at DESC").
		Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  logs,
			"total": total,
		},
		"msg": "success",
	})
}

// ---- Sub-Task Result Callback (called by client via heartbeat) ----

// ReportTaskResult receives execution results from client agents.
func ReportTaskResult(c *gin.Context) {
	var result models.CommandResult
	if err := c.ShouldBindJSON(&result); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	subTaskID, err := uuid.Parse(result.SubTaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的子任务ID"})
		return
	}

	taskID, err := uuid.Parse(result.TaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的任务ID"})
		return
	}

	// Load the SubTask to determine its command type and target machine.
	// This is needed to handle special post-processing (e.g. sync_nodes).
	var subTask models.SubTask
	if err := database.DB.Where("id = ?", subTaskID).First(&subTask).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "子任务不存在"})
		return
	}

	tx := database.DB.Begin()

	// Update sub-task
	now := time.Now()
	updates := map[string]interface{}{
		"status":      result.Status,
		"output":      result.Output,
		"exit_code":   result.ExitCode,
		"error":       result.Error,
		"finished_at": now,
	}
	if err := tx.Model(&models.SubTask{}).Where("id = ?", subTaskID).Updates(updates).Error; err != nil {
		tx.Rollback()
		logger.Error("更新子任务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新子任务失败"})
		return
	}

	// Record log
	tx.Create(&models.TaskLog{
		TaskID:    taskID,
		SubTaskID: &subTaskID,
		ClientID:  result.ClientID,
		Level:     logLevelFromStatus(result.Status),
		Message:   "子任务执行完成: " + result.Status,
		Details:   models.NewJSONBFromMap(map[string]interface{}{"output": result.Output, "exit_code": result.ExitCode}),
	})

	// Update parent task counters
	updateParentTaskStatus(tx, taskID)
	syncTaskExecutionRecord(tx, taskID)

	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新任务状态失败"})
		return
	}

	// -----------------------------------------------------------------------
	// Broadcast K8s deploy progress via WebSocket so frontend updates without polling.
	// -----------------------------------------------------------------------
	var parentTask models.Task
	if database.DB.Where("id = ?", taskID).First(&parentTask).Error == nil && parentTask.Type == string(models.TaskTypeK8sDeploy) {
		go BroadcastK8sDeployProgress(taskID)
	}

	// -----------------------------------------------------------------------
	// Special post-processing for sync_nodes tasks.
	// -----------------------------------------------------------------------
	if subTask.Command == "sync_nodes" && result.Status == string(models.TaskStatusSuccess) && result.Output != "" {
		var parsed struct {
			Workers json.RawMessage `json:"workers"`
		}
		if err := json.Unmarshal([]byte(result.Output), &parsed); err == nil && len(parsed.Workers) > 0 && string(parsed.Workers) != "null" {
			go iotservice.ApplySyncNodesResult(subTask.MachineID, result.ClientID, string(parsed.Workers))
		}
	}

	// -----------------------------------------------------------------------
	// Special post-processing for install_k8s tasks.
	// Update the K8sCluster record's status to match the final task result so
	// the cluster list page shows the correct state immediately.
	// -----------------------------------------------------------------------
	if subTask.Command == "install_k8s" {
		go applyK8sDeployResult(taskID, result.Status)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "结果已接收"})
}

// ---- Helper Functions ----

// updateParentTaskStatus recalculates the parent task's status from its sub-tasks.
func updateParentTaskStatus(tx *gorm.DB, taskID uuid.UUID) {
	var total, successCount, failedCount, runningCount int64

	tx.Model(&models.SubTask{}).Where("task_id = ?", taskID).Count(&total)
	tx.Model(&models.SubTask{}).Where("task_id = ? AND status = ?", taskID, "success").Count(&successCount)
	tx.Model(&models.SubTask{}).Where("task_id = ? AND status = ?", taskID, "failed").Count(&failedCount)
	tx.Model(&models.SubTask{}).Where("task_id = ? AND status = ?", taskID, "running").Count(&runningCount)

	updates := map[string]interface{}{
		"success_count": successCount,
		"failed_count":  failedCount,
	}

	// Determine overall task status
	if successCount+failedCount == total && total > 0 {
		// All sub-tasks completed
		now := time.Now()
		updates["finished_at"] = now
		if failedCount > 0 {
			updates["status"] = string(models.TaskStatusFailed)
		} else {
			updates["status"] = string(models.TaskStatusSuccess)
		}
	} else if runningCount > 0 || successCount > 0 || failedCount > 0 {
		updates["status"] = string(models.TaskStatusRunning)
	}

	tx.Model(&models.Task{}).Where("id = ?", taskID).Updates(updates)
}

// mapTaskTypeToCommand maps a task type to a command string for the client.
func mapTaskTypeToCommand(taskType string) string {
	switch models.TaskType(taskType) {
	case models.TaskTypeShell:
		return "run_shell"
	case models.TaskTypeK8sDeploy:
		return "install_k8s"
	case models.TaskTypeSysInit:
		return "sys_init"
	case models.TaskTypeTimeSync:
		return "time_sync"
	case models.TaskTypeInstallMonitor:
		return "install_monitor"
	case models.TaskTypeFileDistrib:
		return "distribute_file"
	case models.TaskTypeSecurityHarden:
		return "security_harden"
	case models.TaskTypeDiskOptimize:
		return "disk_optimize"
	case models.TaskTypeRegisterNodes:
		return "sync_nodes"
	default:
		return "run_shell"
	}
}

// logLevelFromStatus returns a log level string based on task status.
func logLevelFromStatus(status string) string {
	switch status {
	case "success":
		return "info"
	case "failed":
		return "error"
	default:
		return "warn"
	}
}

// applyK8sDeployResult updates the K8sCluster record that corresponds to a
// completed install_k8s task.  Called as a goroutine after ReportTaskResult commits.
func applyK8sDeployResult(taskID uuid.UUID, resultStatus string) {
	// Load the task to find the cluster_id stored in its Payload.
	var task models.Task
	if err := database.DB.Where("id = ?", taskID).First(&task).Error; err != nil {
		logger.Error("applyK8sDeployResult: task %s not found: %v", taskID, err)
		return
	}

	// Extract cluster_id from payload {"script":..., "cluster_id":"...", "cluster_name":"..."}
	var payload struct {
		ClusterID string `json:"cluster_id"`
	}
	if err := json.Unmarshal(task.Payload, &payload); err != nil || payload.ClusterID == "" {
		logger.Warn("applyK8sDeployResult: no cluster_id in task payload for task %s", taskID)
		return
	}

	clusterID, err := uuid.Parse(payload.ClusterID)
	if err != nil {
		logger.Error("applyK8sDeployResult: invalid cluster_id %s: %v", payload.ClusterID, err)
		return
	}

	var clusterStatus string
	switch resultStatus {
	case string(models.TaskStatusSuccess):
		clusterStatus = "running"
	case string(models.TaskStatusFailed):
		clusterStatus = "failed"
	default:
		clusterStatus = "failed"
	}

	if err := database.DB.Model(&models.K8sCluster{}).
		Where("id = ?", clusterID).
		Update("status", clusterStatus).Error; err != nil {
		logger.Error("applyK8sDeployResult: update cluster %s to %s failed: %v",
			clusterID, clusterStatus, err)
		return
	}
	logger.Info("applyK8sDeployResult: cluster %s → %s", clusterID, clusterStatus)
}

// GetRunningTasks returns currently running/dispatched tasks (used by frontend polling).
// Endpoint: GET /api/v1/tasks/running
func GetRunningTasks(c *gin.Context) {
	var tasks []models.Task
	database.DB.
		Where("status IN ?", []string{
			string(models.TaskStatusPending),
			string(models.TaskStatusDispatched),
			string(models.TaskStatusRunning),
		}).
		Order("created_at DESC").
		Limit(50).
		Find(&tasks)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": tasks,
		"msg":  "success",
	})
}
