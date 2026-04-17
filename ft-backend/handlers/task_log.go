package handlers

import (
	"net/http"
	"time"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// taskLogRequest is the payload that a client agent posts during task execution.
type taskLogRequest struct {
	TaskID    string `json:"task_id"    binding:"required"`
	SubTaskID string `json:"sub_task_id" binding:"required"`
	ClientID  string `json:"client_id"`
	Level     string `json:"level"`   // info | warn | error | debug
	Message   string `json:"message"  binding:"required"`
}

// PostTaskLog receives a single log line from a client while a task is executing.
// It persists the line to TaskLog and, on the first log received, transitions
// the associated SubTask from "dispatched"/"pending" → "running".
//
// Endpoint: POST /api/v1/task/log  (public, client-authenticated)
func PostTaskLog(c *gin.Context) {
	var req taskLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid request: " + err.Error()})
		return
	}

	taskID, err := uuid.Parse(req.TaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid task_id"})
		return
	}
	subTaskID, err := uuid.Parse(req.SubTaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid sub_task_id"})
		return
	}

	level := req.Level
	if level == "" {
		level = "info"
	}

	// Store the log line.
	entry := models.TaskLog{
		TaskID:    taskID,
		SubTaskID: &subTaskID,
		ClientID:  req.ClientID,
		Level:     level,
		Message:   req.Message,
	}
	if err := database.DB.Create(&entry).Error; err != nil {
		logger.Error("PostTaskLog: persist log failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "failed to save log"})
		return
	}

	// Transition SubTask to "running" if it is still pending/dispatched.
	// This gives the progress page an accurate status as soon as execution begins.
	var subTask models.SubTask
	if err := database.DB.Where("id = ?", subTaskID).First(&subTask).Error; err == nil {
		if subTask.Status == string(models.TaskStatusPending) ||
			subTask.Status == string(models.TaskStatusDispatched) {
			now := time.Now()
			database.DB.Model(&subTask).Updates(map[string]interface{}{
				"status":     string(models.TaskStatusRunning),
				"started_at": now,
			})
			// Also push Task to "running"
			database.DB.Model(&models.Task{}).
				Where("id = ? AND status IN ?", taskID,
					[]string{string(models.TaskStatusPending), string(models.TaskStatusDispatched)}).
				Updates(map[string]interface{}{
					"status":     string(models.TaskStatusRunning),
					"started_at": now,
				})
			// Broadcast K8s deploy progress so deploy page shows running state without polling.
			var task models.Task
			if database.DB.Where("id = ?", taskID).First(&task).Error == nil && task.Type == string(models.TaskTypeK8sDeploy) {
				go BroadcastK8sDeployProgress(taskID)
			}
		}
	} else if err != gorm.ErrRecordNotFound {
		// Non-fatal: log but don't fail the request.
		logger.Warn("PostTaskLog: could not load sub-task for status transition: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "ok"})
}
