package handlers

import (
	"encoding/json"
	"net/http"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetJobMachines returns the machine list for the job center (online machines only).
func GetJobMachines(c *gin.Context) {
	var machines []models.Machine
	db := database.DB.Model(&models.Machine{}).Where("status = ?", "online")

	name := c.Query("name")
	if name != "" {
		db = db.Where("name ILIKE ?", "%"+name+"%")
	}

	db.Order("name ASC").Find(&machines)

	// Transform to the format the frontend expects
	type MachineItem struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		IP     string `json:"ip"`
		Status string `json:"status"`
	}
	var list []MachineItem
	for _, m := range machines {
		list = append(list, MachineItem{
			ID:     m.ID.String(),
			Name:   m.Name,
			IP:     m.IP,
			Status: m.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": list,
		"msg":  "success",
	})
}

// ExecuteJob creates a shell task and dispatches it to selected machines.
func ExecuteJob(c *gin.Context) {
	var req struct {
		MachineIDs []string `json:"machine_ids" binding:"required"`
		Command    string   `json:"command" binding:"required"`
		Timeout    int      `json:"timeout"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数", "error": err.Error()})
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = 60
	}

	username, _ := c.Get("username")
	payload, _ := json.Marshal(map[string]interface{}{
		"script": req.Command,
	})
	targetIDsJSON, _ := json.Marshal(req.MachineIDs)

	task := models.Task{
		Name:       "Job: " + truncateString(req.Command, 50),
		Type:       string(models.TaskTypeShell),
		Status:     string(models.TaskStatusPending),
		CreatedBy:  username.(string),
		Payload:    models.JSONB(payload),
		TargetIDs:  models.JSONB(targetIDsJSON),
		TotalCount: len(req.MachineIDs),
		TimeoutSec: req.Timeout,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&task).Error; err != nil {
		tx.Rollback()
		logger.Error("创建Job任务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败"})
		return
	}

	for _, machineID := range req.MachineIDs {
		machineUUID, err := uuid.Parse(machineID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的机器ID: " + machineID})
			return
		}

		var machine models.Machine
		if err := tx.Where("id = ?", machineUUID).First(&machine).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "机器不存在: " + machineID})
				return
			}
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询机器失败"})
			return
		}

		subTask := models.SubTask{
			TaskID:    task.ID,
			MachineID: machineUUID,
			ClientID:  machine.IP,
			Command:   "run_shell",
			Status:    string(models.TaskStatusPending),
			Payload:   models.JSONB(payload),
			MaxRetry:  1,
		}
		if err := tx.Create(&subTask).Error; err != nil {
			tx.Rollback()
			logger.Error("创建子任务失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建子任务失败"})
			return
		}
	}
	createTaskExecutionRecord(tx, task, "job", "shell", req.Command, "", map[string]interface{}{
		"capability": "manual",
		"advice":     "Shell 作业无法可靠推导自动回滚，请根据输出和业务变更执行人工恢复。",
	})

	if err := tx.Commit().Error; err != nil {
		logger.Error("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"jobId":  task.ID.String(),
			"status": task.Status,
		},
		"msg": "任务已创建",
	})
}

// GetJobResult returns execution results for a specific job.
func GetJobResult(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的任务ID"})
		return
	}

	var task models.Task
	if err := database.DB.Where("id = ?", jobID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "任务不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询任务失败"})
		return
	}

	var subTasks []models.SubTask
	database.DB.Where("task_id = ?", jobID).Order("created_at ASC").Find(&subTasks)

	// Build result list with machine info
	type ResultItem struct {
		MachineID   string `json:"machine_id"`
		MachineName string `json:"machine_name"`
		MachineIP   string `json:"machine_ip"`
		Status      string `json:"status"`
		Output      string `json:"output"`
		ExitCode    *int   `json:"exit_code"`
		Error       string `json:"error"`
	}

	var results []ResultItem
	for _, st := range subTasks {
		var machine models.Machine
		database.DB.Where("id = ?", st.MachineID).First(&machine)

		results = append(results, ResultItem{
			MachineID:   st.MachineID.String(),
			MachineName: machine.Name,
			MachineIP:   machine.IP,
			Status:      st.Status,
			Output:      st.Output,
			ExitCode:    st.ExitCode,
			Error:       st.Error,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"jobId":   task.ID.String(),
			"name":    task.Name,
			"status":  task.Status,
			"results": results,
		},
		"msg": "success",
	})
}

// truncateString truncates a string to the given length.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
