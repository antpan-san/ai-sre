package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// Response DTOs (frontend-aligned)
// ─────────────────────────────────────────────────────────────────────────────

// K8sVersionDTO is the frontend-compatible K8s version payload.
type K8sVersionDTO struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Recommended bool   `json:"recommended"`
	IsActive    bool   `json:"is_active"`
}

// DeployProgressDTO matches the frontend DeployProgress interface.
type DeployProgressDTO struct {
	DeployID     string  `json:"deployId"`
	Status       string  `json:"status"`
	Progress     int     `json:"progress"`
	CurrentStep  string  `json:"currentStep"`
	StepProgress int     `json:"stepProgress"`
	StartTime    *string `json:"startTime,omitempty"`
	EndTime      *string `json:"endTime,omitempty"`
	Error        string  `json:"error,omitempty"`
	TotalCount   int     `json:"totalCount"`
	SuccessCount int     `json:"successCount"`
	FailedCount  int     `json:"failedCount"`
}

// DeployLogDTO matches the frontend DeployLog interface.
type DeployLogDTO struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Step      string `json:"step,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// GetK8sVersions – returns active versions with a recommended flag.
// ─────────────────────────────────────────────────────────────────────────────

func GetK8sVersions(c *gin.Context) {
	versions, err := database.GetK8sVersions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Failed to fetch K8s versions",
			"data": nil,
		})
		return
	}

	// Sort versions descending so the newest is first (and marked recommended).
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	dtos := make([]K8sVersionDTO, 0, len(versions))
	for i, v := range versions {
		dtos = append(dtos, K8sVersionDTO{
			ID:          v.ID.String(),
			Version:     v.Version,
			Description: describeVersion(v.Version, i == 0),
			Recommended: i == 0, // newest active version is recommended
			IsActive:    v.IsActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": dtos,
	})
}

func describeVersion(ver string, recommended bool) string {
	if recommended {
		return ver + " (推荐版本)"
	}
	return ver
}

// ─────────────────────────────────────────────────────────────────────────────
// GetK8sDeployMachines – returns machines available for K8s deployment.
// ─────────────────────────────────────────────────────────────────────────────

func GetK8sDeployMachines(c *gin.Context) {
	var machines []models.Machine
	status := c.Query("status")

	query := database.DB
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&machines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Failed to fetch machines",
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": machines,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// CheckClusterName – checks whether a cluster name is available.
// ─────────────────────────────────────────────────────────────────────────────

func CheckClusterName(c *gin.Context) {
	clusterName := c.Query("clusterName")
	if clusterName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Cluster name is required",
			"data": gin.H{"isAvailable": false},
		})
		return
	}

	var count int64
	if err := database.DB.Model(&models.K8sCluster{}).Where("cluster_name = ?", clusterName).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Failed to check cluster name",
			"data": gin.H{"isAvailable": false},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{"isAvailable": count == 0},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// BuildDeployProgressDTO loads task + sub-tasks and returns frontend DTO.
// Used by GetK8sDeployProgress and by WebSocket broadcast.
// ─────────────────────────────────────────────────────────────────────────────

func BuildDeployProgressDTO(taskID uuid.UUID) (DeployProgressDTO, bool) {
	var task models.Task
	if err := database.DB.Where("id = ?", taskID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return DeployProgressDTO{}, false
		}
		return DeployProgressDTO{}, false
	}

	var subTasks []models.SubTask
	database.DB.Where("task_id = ?", taskID).Order("created_at ASC").Find(&subTasks)

	completedCount := 0
	currentStep := task.Description
	for _, st := range subTasks {
		if st.Status == string(models.TaskStatusSuccess) || st.Status == string(models.TaskStatusFailed) {
			completedCount++
		} else if st.Status == string(models.TaskStatusRunning) {
			currentStep = "执行中: " + st.Command
		}
	}

	progress := 0
	if task.TotalCount > 0 {
		progress = completedCount * 100 / task.TotalCount
	}

	stepProgress := 0
	if task.TotalCount > 0 && completedCount < task.TotalCount {
		stepProgress = (completedCount * 100) / task.TotalCount
	} else if completedCount == task.TotalCount && task.TotalCount > 0 {
		stepProgress = 100
	}

	var startTime, endTime *string
	if task.StartedAt != nil {
		s := task.StartedAt.Format(time.RFC3339)
		startTime = &s
	}
	if task.FinishedAt != nil {
		e := task.FinishedAt.Format(time.RFC3339)
		endTime = &e
	}

	errMsg := ""
	for _, st := range subTasks {
		if st.Status == string(models.TaskStatusFailed) && st.Error != "" {
			errMsg = st.Error
			break
		}
	}

	dto := DeployProgressDTO{
		DeployID:     task.ID.String(),
		Status:       mapTaskStatus(task.Status),
		Progress:     progress,
		CurrentStep:  currentStep,
		StepProgress: stepProgress,
		StartTime:    startTime,
		EndTime:      endTime,
		Error:        errMsg,
		TotalCount:   task.TotalCount,
		SuccessCount: task.SuccessCount,
		FailedCount:  task.FailedCount,
	}
	return dto, true
}

// BroadcastK8sDeployProgress pushes the current progress for the given deploy to all WS clients.
// Call this after any task/subtask status update for k8s_deploy tasks.
func BroadcastK8sDeployProgress(taskID uuid.UUID) {
	if utils.GlobalWebSocketManager == nil {
		return
	}
	dto, ok := BuildDeployProgressDTO(taskID)
	if !ok {
		return
	}
	utils.GlobalWebSocketManager.Broadcast(utils.WebSocketMessage{
		Type: "k8s_deploy_progress",
		Data: dto,
	})
	logger.Debug("BroadcastK8sDeployProgress: deployId=%s status=%s progress=%d", dto.DeployID, dto.Status, dto.Progress)
}

// GetK8sDeployProgress – returns richer progress aligned with frontend.
// ─────────────────────────────────────────────────────────────────────────────

func GetK8sDeployProgress(c *gin.Context) {
	deployID := c.Query("deployId")
	if deployID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "deployId 参数缺失"})
		return
	}

	taskID, err := uuid.Parse(deployID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 deployId"})
		return
	}

	dto, ok := BuildDeployProgressDTO(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "部署任务不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": dto,
	})
}

// mapTaskStatus converts backend TaskStatus to frontend-expected status values.
func mapTaskStatus(status string) string {
	switch status {
	case string(models.TaskStatusPending), string(models.TaskStatusDispatched):
		return "pending"
	case string(models.TaskStatusRunning):
		return "running"
	case string(models.TaskStatusSuccess):
		return "success"
	case string(models.TaskStatusFailed), string(models.TaskStatusTimeout):
		return "failed"
	case string(models.TaskStatusCancelled):
		return "cancelled"
	default:
		return "pending"
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GetK8sDeployLogs – returns logs in frontend-aligned {logs,total,hasMore} format.
// ─────────────────────────────────────────────────────────────────────────────

func GetK8sDeployLogs(c *gin.Context) {
	deployID := c.Query("deployId")
	if deployID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "deployId 参数缺失"})
		return
	}

	taskID, err := uuid.Parse(deployID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 deployId"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var total int64
	database.DB.Model(&models.TaskLog{}).Where("task_id = ?", taskID).Count(&total)

	var rawLogs []models.TaskLog
	database.DB.Where("task_id = ?", taskID).
		Order("created_at ASC").
		Limit(limit).Offset(offset).
		Find(&rawLogs)

	logs := make([]DeployLogDTO, 0, len(rawLogs))
	for _, l := range rawLogs {
		logs = append(logs, DeployLogDTO{
			Timestamp: l.CreatedAt.Format(time.RFC3339),
			Level:     normalizeLogLevel(l.Level),
			Message:   l.Message,
			Step:      l.ClientID, // reuse ClientID as step hint when set
		})
	}

	hasMore := int64(offset+limit) < total

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"logs":    logs,
			"total":   total,
			"hasMore": hasMore,
		},
	})
}

func normalizeLogLevel(level string) string {
	switch level {
	case "warn", "warning":
		return "warning"
	case "error":
		return "error"
	default:
		return "info"
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TerminateK8sDeploy – 终止部署并下发清理任务，使 client 端严格恢复到部署前状态。
// 部署脚本每步会写入状态文件，清理任务按逆序执行对应清理。
// ─────────────────────────────────────────────────────────────────────────────

func TerminateK8sDeploy(c *gin.Context) {
	var req struct {
		DeployID string `json:"deployId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "deployId 必填"})
		return
	}

	taskID, err := uuid.Parse(req.DeployID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 deployId"})
		return
	}

	var task models.Task
	if err := database.DB.Where("id = ?", taskID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "部署任务不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询任务失败"})
		return
	}

	if task.Type != string(models.TaskTypeK8sDeploy) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "该任务不是 K8s 部署任务"})
		return
	}

	if task.Status != string(models.TaskStatusPending) && task.Status != string(models.TaskStatusRunning) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "只有进行中或待执行的部署可以终止"})
		return
	}

	username, _ := c.Get("username")

	now := time.Now()
	tx := database.DB.Begin()
	tx.Model(&task).Updates(map[string]interface{}{
		"status":      string(models.TaskStatusCancelled),
		"finished_at": now,
	})
	tx.Model(&models.SubTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":      string(models.TaskStatusCancelled),
		"finished_at": now,
	})
	tx.Create(&models.TaskLog{
		TaskID:  taskID,
		Level:   "warn",
		Message: "用户终止部署，已下发清理任务以恢复到部署前状态",
	})
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新任务状态失败"})
		return
	}

	var targetIDs []string
	if err := json.Unmarshal(task.TargetIDs, &targetIDs); err != nil || len(targetIDs) == 0 {
		logger.Warn("TerminateK8sDeploy: no target_ids for task %s", taskID)
		if utils.GlobalWebSocketManager != nil {
			go BroadcastK8sDeployProgress(taskID)
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "部署已终止"})
		return
	}

	masterMachineID := targetIDs[0]
	masterUUID, err := uuid.Parse(masterMachineID)
	if err != nil {
		if utils.GlobalWebSocketManager != nil {
			go BroadcastK8sDeployProgress(taskID)
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "部署已终止"})
		return
	}

	var machine models.Machine
	if err := database.DB.Where("id = ?", masterUUID).First(&machine).Error; err != nil {
		if utils.GlobalWebSocketManager != nil {
			go BroadcastK8sDeployProgress(taskID)
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "部署已终止"})
		return
	}

	clusterName := strings.TrimPrefix(task.Name, "K8s部署: ")
	clusterName = strings.TrimPrefix(clusterName, "K8s 部署: ")
	if clusterName == "" {
		clusterName = taskID.String()[:8]
	}

	cleanupScript := generateK8sCleanupScript()
	payload, _ := json.Marshal(map[string]interface{}{"script": cleanupScript})

	cleanupTask := models.Task{
		Name:        "K8s清理: " + clusterName,
		Type:        string(models.TaskTypeShell),
		Status:      string(models.TaskStatusPending),
		CreatedBy:   username.(string),
		Description: "终止部署后清理，恢复到部署前状态",
		Payload:     models.JSONB(payload),
		TargetIDs:   task.TargetIDs,
		TotalCount:  1,
		TimeoutSec:  600,
	}
	if err := database.DB.Create(&cleanupTask).Error; err != nil {
		logger.Error("TerminateK8sDeploy: create cleanup task failed: %v", err)
		if utils.GlobalWebSocketManager != nil {
			go BroadcastK8sDeployProgress(taskID)
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "部署已终止，清理任务创建失败请手动清理"})
		return
	}

	subTask := models.SubTask{
		TaskID:    cleanupTask.ID,
		MachineID: masterUUID,
		ClientID:  machine.ClientID,
		Command:   "run_shell",
		Status:    string(models.TaskStatusPending),
		Payload:   models.JSONB(payload),
		MaxRetry:  1,
	}
	if machine.ClientID == "" {
		subTask.ClientID = machine.IP
	}
	if err := database.DB.Create(&subTask).Error; err != nil {
		logger.Error("TerminateK8sDeploy: create cleanup sub-task failed: %v", err)
	}

	if utils.GlobalWebSocketManager != nil {
		go BroadcastK8sDeployProgress(taskID)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "部署已终止，已下发清理任务"})
}

// ─────────────────────────────────────────────────────────────────────────────
// GetK8sDeployRecords – returns list of K8s deploy tasks (deployment history).
// ─────────────────────────────────────────────────────────────────────────────

func GetK8sDeployRecords(c *gin.Context) {
	var tasks []models.Task
	if err := database.DB.Where("type = ?", models.TaskTypeK8sDeploy).
		Order("created_at DESC").
		Limit(100).
		Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取部署记录失败"})
		return
	}

	type recordRow struct {
		DeployID     string  `json:"deployId"`
		ClusterName  string  `json:"clusterName"`
		Status       string  `json:"status"`
		Progress     int     `json:"progress"`
		CurrentStep  string  `json:"currentStep"`
		StepProgress int     `json:"stepProgress"`
		StartTime    *string `json:"startTime,omitempty"`
		EndTime      *string `json:"endTime,omitempty"`
		Error        string  `json:"error,omitempty"`
		CreatedAt    string  `json:"createdAt"`
	}

	rows := make([]recordRow, 0, len(tasks))
	for _, t := range tasks {
		dto, ok := BuildDeployProgressDTO(t.ID)
		if !ok {
			continue
		}
		clusterName := strings.TrimPrefix(t.Name, "K8s部署: ")
		clusterName = strings.TrimPrefix(clusterName, "K8s 部署: ")
		if clusterName == "" {
			clusterName = t.Name
		}
		var startTime, endTime *string
		if t.StartedAt != nil {
			s := t.StartedAt.Format(time.RFC3339)
			startTime = &s
		}
		if t.FinishedAt != nil {
			e := t.FinishedAt.Format(time.RFC3339)
			endTime = &e
		}
		rows = append(rows, recordRow{
			DeployID:     dto.DeployID,
			ClusterName:  clusterName,
			Status:       dto.Status,
			Progress:     dto.Progress,
			CurrentStep:  dto.CurrentStep,
			StepProgress: dto.StepProgress,
			StartTime:    startTime,
			EndTime:      endTime,
			Error:        dto.Error,
			CreatedAt:    t.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": rows,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// GetK8sClusters – returns the list of deployed clusters.
// ─────────────────────────────────────────────────────────────────────────────

func GetK8sClusters(c *gin.Context) {
	var clusters []models.K8sCluster
	var total int64

	database.DB.Model(&models.K8sCluster{}).Count(&total)
	if err := database.DB.Order("created_at DESC").Find(&clusters).Error; err != nil {
		response.ServerError(c, "获取集群列表失败")
		return
	}

	response.OKPage(c, clusters, total)
}
