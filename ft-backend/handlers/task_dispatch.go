package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"ft-backend/common/logger"
	"ft-backend/common/redis"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
)

type managedTaskRequest struct {
	Name        string
	Type        string
	Command     string
	Description string
	MachineIDs  []string
	Payload     map[string]interface{}
	TimeoutSec  int
	MaxRetry    int
}

func createManagedTask(c *gin.Context, req managedTaskRequest) (*models.Task, error) {
	if len(req.MachineIDs) == 0 {
		return nil, fmt.Errorf("missing target machines")
	}
	if req.TimeoutSec <= 0 {
		req.TimeoutSec = 300
	}
	if req.MaxRetry <= 0 {
		req.MaxRetry = 1
	}
	if strings.TrimSpace(req.Command) == "" {
		req.Command = mapTaskTypeToCommand(req.Type)
	}

	payloadJSON, _ := json.Marshal(req.Payload)
	targetIDsJSON, _ := json.Marshal(req.MachineIDs)
	task := models.Task{
		Name:        req.Name,
		Type:        req.Type,
		Status:      string(models.TaskStatusPending),
		CreatedBy:   currentUsername(c),
		Description: req.Description,
		Payload:     models.JSONB(payloadJSON),
		TargetIDs:   models.JSONB(targetIDsJSON),
		TotalCount:  len(req.MachineIDs),
		TimeoutSec:  req.TimeoutSec,
	}
	applyBillingSnapshot(c, &task)

	tx := database.DB.Begin()
	if err := tx.Create(&task).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	for _, machineID := range req.MachineIDs {
		var machine models.Machine
		if err := tx.Where("id = ?", machineID).First(&machine).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		dispatchID := machine.ClientID
		if dispatchID == "" {
			dispatchID = machine.IP
		}
		if dispatchID == "" {
			tx.Rollback()
			return nil, fmt.Errorf("machine %s has no client_id or ip", machineID)
		}
		subTask := models.SubTask{
			TaskID:    task.ID,
			MachineID: machine.ID,
			ClientID:  dispatchID,
			Command:   req.Command,
			Status:    string(models.TaskStatusPending),
			Payload:   models.JSONB(payloadJSON),
			MaxRetry:  req.MaxRetry,
		}
		if err := tx.Create(&subTask).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		enqueueManagedCommand(machine, task, subTask, json.RawMessage(payloadJSON))
	}
	tx.Create(&models.TaskLog{TaskID: task.ID, Level: "info", Message: req.Name + " 任务已创建"})
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func enqueueManagedCommand(machine models.Machine, task models.Task, subTask models.SubTask, payload json.RawMessage) {
	if !redis.IsConnected() {
		return
	}
	cmd := models.Command{
		TaskID:    task.ID.String(),
		SubTaskID: subTask.ID.String(),
		Command:   subTask.Command,
		Payload:   payload,
		Timeout:   task.TimeoutSec,
	}
	for _, key := range []string{machine.ClientID, machine.IP} {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if err := redis.EnqueueTask(key, cmd); err != nil {
			logger.Warn("enqueue task failed key=%s task=%s sub_task=%s: %v", key, task.ID, subTask.ID, err)
		}
	}
}

func currentUsername(c *gin.Context) string {
	if c != nil {
		if v, ok := c.Get("username"); ok {
			if s, _ := v.(string); strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return "system"
}
