package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const executionSummaryLimit = 8000

type executionRecordPayload struct {
	CorrelationID      string          `json:"correlation_id"`
	Source             string          `json:"source"`
	Category           string          `json:"category"`
	Name               string          `json:"name"`
	Command            string          `json:"command"`
	Status             string          `json:"status"`
	ExitCode           *int            `json:"exit_code"`
	CreatedBy          string          `json:"created_by"`
	TriggerUser        string          `json:"trigger_user"`
	TargetHost         string          `json:"target_host"`
	TargetIPs          []string        `json:"target_ips"`
	ResourceType       string          `json:"resource_type"`
	ResourceID         string          `json:"resource_id"`
	ResourceName       string          `json:"resource_name"`
	TaskID             string          `json:"task_id"`
	ParentExecutionID  string          `json:"parent_execution_id"`
	StdoutSummary      string          `json:"stdout_summary"`
	StderrSummary      string          `json:"stderr_summary"`
	Effects            json.RawMessage `json:"effects"`
	Metadata           json.RawMessage `json:"metadata"`
	RollbackCapability string          `json:"rollback_capability"`
	RollbackPlan       json.RawMessage `json:"rollback_plan"`
	RollbackAdvice     string          `json:"rollback_advice"`
	InviteID           string          `json:"invite_id"`
	Token              string          `json:"token"`
}

type executionEventPayload struct {
	RecordID      string          `json:"record_id"`
	CorrelationID string          `json:"correlation_id"`
	InviteID      string          `json:"invite_id"`
	Token         string          `json:"token"`
	Level         string          `json:"level"`
	Phase         string          `json:"phase"`
	Message       string          `json:"message"`
	Output        string          `json:"output"`
	Details       json.RawMessage `json:"details"`
}

type executionFinishPayload struct {
	executionRecordPayload
	RecordID string `json:"record_id"`
}

// PrepareExecutionRecord creates a pending record and one-time report token for
// copied scripts generated from authenticated console pages.
func PrepareExecutionRecord(c *gin.Context) {
	var req executionRecordPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}
	username, _ := c.Get("username")
	if req.CreatedBy == "" {
		req.CreatedBy, _ = username.(string)
	}
	token, tokenHash, err := newExecutionReportToken()
	if err != nil {
		response.ServerError(c, "生成执行上报 token 失败")
		return
	}
	rec := buildExecutionRecord(req)
	rec.Status = models.ExecutionStatusPending
	rec.ReportTokenHash = tokenHash
	if rec.CorrelationID == "" {
		rec.CorrelationID = uuid.NewString()
	}
	if err := database.DB.Create(&rec).Error; err != nil {
		logger.Error("PrepareExecutionRecord: %v", err)
		response.ServerError(c, "保存执行记录失败")
		return
	}
	response.OK(c, gin.H{
		"id":            rec.ID.String(),
		"correlationId": rec.CorrelationID,
		"reportToken":   token,
	})
}

// StartExecutionRecord is a public, non-blocking telemetry endpoint used by
// copied scripts and ai-sre. It authenticates with an invite token or a
// per-record report token.
func StartExecutionRecord(c *gin.Context) {
	var req executionRecordPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}
	if !authorizeExecutionReport(req.InviteID, req.Token, req.CorrelationID, "") {
		response.Unauthorized(c, "执行上报 token 无效")
		return
	}

	now := time.Now()
	rec := buildExecutionRecord(req)
	if rec.CorrelationID == "" {
		rec.CorrelationID = uuid.NewString()
	}
	rec.Status = models.ExecutionStatusRunning
	rec.StartedAt = &now
	if req.Token != "" && rec.ReportTokenHash == "" && req.InviteID == "" {
		rec.ReportTokenHash = hashExecutionToken(req.Token)
	}

	tx := database.DB.Begin()
	if err := upsertExecutionStart(tx, &rec); err != nil {
		tx.Rollback()
		logger.Error("StartExecutionRecord: %v", err)
		response.ServerError(c, "保存执行记录失败")
		return
	}
	addExecutionEvent(tx, rec.ID, "info", "start", "执行开始", "", models.NewJSONBFromMap(map[string]interface{}{
		"source":      rec.Source,
		"category":    rec.Category,
		"target_host": rec.TargetHost,
	}))
	if err := tx.Commit().Error; err != nil {
		response.ServerError(c, "保存执行记录失败")
		return
	}
	response.OK(c, gin.H{"id": rec.ID.String(), "correlationId": rec.CorrelationID})
}

func PostExecutionEvent(c *gin.Context) {
	var req executionEventPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}
	rec, ok := findAuthorizedExecutionRecord(c, req.RecordID, req.CorrelationID, req.InviteID, req.Token)
	if !ok {
		return
	}
	level := req.Level
	if level == "" {
		level = "info"
	}
	phase := req.Phase
	if phase == "" {
		phase = "progress"
	}
	if req.Message == "" {
		req.Message = "执行进度更新"
	}
	details := rawJSONOrObject(req.Details)
	if err := database.DB.Create(&models.ExecutionEvent{
		ExecutionID: rec.ID,
		Level:       level,
		Phase:       phase,
		Message:     req.Message,
		Output:      limitText(req.Output, executionSummaryLimit),
		Details:     details,
	}).Error; err != nil {
		response.ServerError(c, "保存执行事件失败")
		return
	}
	response.OKMsg(c, "执行事件已接收")
}

func FinishExecutionRecord(c *gin.Context) {
	var req executionFinishPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}
	rec, ok := findAuthorizedExecutionRecord(c, req.RecordID, req.CorrelationID, req.InviteID, req.Token)
	if !ok {
		return
	}
	status := normalizeExecutionStatus(req.Status)
	if status == "" {
		if req.ExitCode != nil && *req.ExitCode != 0 {
			status = models.ExecutionStatusFailed
		} else {
			status = models.ExecutionStatusSuccess
		}
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":         status,
		"exit_code":      req.ExitCode,
		"finished_at":    now,
		"stdout_summary": limitText(req.StdoutSummary, executionSummaryLimit),
		"stderr_summary": limitText(req.StderrSummary, executionSummaryLimit),
	}
	if rec.StartedAt != nil {
		updates["duration_ms"] = now.Sub(*rec.StartedAt).Milliseconds()
	}
	if len(req.Effects) > 0 {
		updates["effects"] = rawJSONOrObject(req.Effects)
	}
	if len(req.Metadata) > 0 {
		updates["metadata"] = rawJSONOrObject(req.Metadata)
	}
	tx := database.DB.Begin()
	if err := tx.Model(&models.ExecutionRecord{}).Where("id = ?", rec.ID).Updates(updates).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "更新执行记录失败")
		return
	}
	addExecutionEvent(tx, rec.ID, logLevelFromStatus(status), "finish", "执行结束: "+status, limitText(req.StdoutSummary, executionSummaryLimit), models.NewJSONBFromMap(map[string]interface{}{
		"exit_code": req.ExitCode,
		"stderr":    limitText(req.StderrSummary, executionSummaryLimit),
	}))
	if err := tx.Commit().Error; err != nil {
		response.ServerError(c, "更新执行记录失败")
		return
	}
	response.OKMsg(c, "执行结果已接收")
}

func GetExecutionRecords(c *gin.Context) {
	p := response.GetPagination(c)
	db := database.DB.Model(&models.ExecutionRecord{})
	if source := strings.TrimSpace(c.Query("source")); source != "" {
		db = db.Where("source = ?", source)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		db = db.Where("status = ?", status)
	}
	if rollback := strings.TrimSpace(c.Query("rollbackCapability")); rollback != "" {
		db = db.Where("rollback_capability = ?", rollback)
	}
	if target := strings.TrimSpace(c.Query("target")); target != "" {
		like := "%" + target + "%"
		db = db.Where("target_host ILIKE ? OR resource_name ILIKE ? OR resource_id ILIKE ?", like, like, like)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name ILIKE ? OR command ILIKE ? OR stdout_summary ILIKE ? OR stderr_summary ILIKE ?", like, like, like, like)
	}
	if start := strings.TrimSpace(c.Query("startDate")); start != "" {
		db = db.Where("created_at >= ?", start)
	}
	if end := strings.TrimSpace(c.Query("endDate")); end != "" {
		db = db.Where("created_at <= ?", end)
	}
	var total int64
	db.Count(&total)
	var list []models.ExecutionRecord
	if err := response.Paginate(db, p, "created_at DESC").Find(&list).Error; err != nil {
		response.ServerError(c, "查询执行记录失败")
		return
	}
	response.OKPage(c, list, total)
}

func GetExecutionRecordDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的执行记录 ID")
		return
	}
	var rec models.ExecutionRecord
	if err := database.DB.Where("id = ?", id).First(&rec).Error; response.HandleDBError(c, err, "执行记录不存在") {
		return
	}
	var events []models.ExecutionEvent
	database.DB.Where("execution_id = ?", id).Order("created_at ASC").Find(&events)
	impacts := findExecutionRollbackImpacts(rec)
	response.OK(c, gin.H{"record": rec, "events": events, "impacts": impacts})
}

func GetExecutionRecordEvents(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的执行记录 ID")
		return
	}
	p := response.GetPagination(c)
	db := database.DB.Model(&models.ExecutionEvent{}).Where("execution_id = ?", id)
	var total int64
	db.Count(&total)
	var list []models.ExecutionEvent
	if err := response.Paginate(db, p, "created_at ASC").Find(&list).Error; err != nil {
		response.ServerError(c, "查询执行事件失败")
		return
	}
	response.OKPage(c, list, total)
}

func GetExecutionRecordDependencies(c *gin.Context) {
	rec, ok := loadExecutionRecordParam(c)
	if !ok {
		return
	}
	var explicit []models.ExecutionDependency
	database.DB.Where("depends_on_execution_id = ? OR execution_id = ?", rec.ID, rec.ID).
		Order("created_at ASC").
		Find(&explicit)
	response.OK(c, gin.H{
		"explicit": explicit,
		"inferred": findExecutionRollbackImpacts(rec),
	})
}

func PreviewExecutionRollback(c *gin.Context) {
	rec, ok := loadExecutionRecordParam(c)
	if !ok {
		return
	}
	impacts := findExecutionRollbackImpacts(rec)
	response.OK(c, gin.H{
		"record":              rec,
		"impacts":             impacts,
		"hasBlockingImpact":   len(impacts) > 0,
		"rollbackCapability":  rec.RollbackCapability,
		"rollbackPlan":        rec.RollbackPlan,
		"rollbackAdvice":      rec.RollbackAdvice,
		"requiresUserConfirm": len(impacts) > 0,
	})
}

func RollbackExecutionRecord(c *gin.Context) {
	rec, ok := loadExecutionRecordParam(c)
	if !ok {
		return
	}
	var req struct {
		Confirmed bool `json:"confirmed"`
	}
	_ = c.ShouldBindJSON(&req)
	impacts := findExecutionRollbackImpacts(rec)
	if len(impacts) > 0 && !req.Confirmed {
		_ = database.DB.Model(&models.ExecutionRecord{}).Where("id = ?", rec.ID).
			Update("rollback_status", models.RollbackStatusBlocked).Error
		response.Conflict(c, "存在后续依赖影响，请确认后再回滚")
		return
	}
	if rec.RollbackCapability == models.RollbackCapabilityNone {
		response.BadRequest(c, "该记录没有自动回滚能力: "+rec.RollbackAdvice)
		return
	}

	username, _ := c.Get("username")
	now := time.Now()
	rollback := models.ExecutionRecord{
		CorrelationID:      uuid.NewString(),
		Source:             "rollback",
		Category:           rec.Category,
		Name:               "Rollback: " + rec.Name,
		Command:            extractRollbackCommand(rec.RollbackPlan),
		CommandDigest:      digestText(extractRollbackCommand(rec.RollbackPlan)),
		Status:             models.ExecutionStatusPending,
		CreatedBy:          fmt.Sprint(username),
		TargetHost:         rec.TargetHost,
		TargetIPs:          rec.TargetIPs,
		ResourceType:       rec.ResourceType,
		ResourceID:         rec.ResourceID,
		ResourceName:       rec.ResourceName,
		ParentExecutionID:  &rec.ID,
		StartedAt:          &now,
		Effects:            models.NewJSONBFromMap(map[string]interface{}{}),
		Metadata:           models.NewJSONBFromMap(map[string]interface{}{"rollback_of": rec.ID.String(), "impacts": impacts}),
		RollbackCapability: models.RollbackCapabilityNone,
		RollbackStatus:     models.RollbackStatusNotStarted,
		RollbackPlan:       models.NewJSONBFromMap(map[string]interface{}{}),
		RollbackAdvice:     "这是回滚记录本身，不再递归回滚。",
	}
	tx := database.DB.Begin()
	for _, impact := range impacts {
		dep := models.ExecutionDependency{
			ExecutionID:          impact.ID,
			DependsOnExecutionID: rec.ID,
			Relation:             "same_target_after",
			ImpactLevel:          "warning",
			Message:              "后续成功执行与当前记录目标或资源重叠，回滚可能影响该执行后的状态。",
			Details:              models.NewJSONBFromMap(map[string]interface{}{"rollback_requested": true}),
		}
		_ = tx.Where("execution_id = ? AND depends_on_execution_id = ? AND relation = ?", dep.ExecutionID, dep.DependsOnExecutionID, dep.Relation).
			FirstOrCreate(&dep).Error
	}
	if err := tx.Create(&rollback).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建回滚记录失败")
		return
	}
	if err := tx.Model(&models.ExecutionRecord{}).Where("id = ?", rec.ID).
		Updates(map[string]interface{}{"rollback_status": models.RollbackStatusPending}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "更新回滚状态失败")
		return
	}
	addExecutionEvent(tx, rollback.ID, "warn", "rollback", "已创建回滚记录，请按回滚计划执行或接入自动执行器", "", rec.RollbackPlan)
	if err := tx.Commit().Error; err != nil {
		response.ServerError(c, "创建回滚记录失败")
		return
	}
	response.OK(c, gin.H{"rollbackRecord": rollback, "impacts": impacts})
}

func loadExecutionRecordParam(c *gin.Context) (models.ExecutionRecord, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的执行记录 ID")
		return models.ExecutionRecord{}, false
	}
	var rec models.ExecutionRecord
	if err := database.DB.Where("id = ?", id).First(&rec).Error; response.HandleDBError(c, err, "执行记录不存在") {
		return models.ExecutionRecord{}, false
	}
	return rec, true
}

func buildExecutionRecord(req executionRecordPayload) models.ExecutionRecord {
	taskID := parseOptionalUUID(req.TaskID)
	parentID := parseOptionalUUID(req.ParentExecutionID)
	status := normalizeExecutionStatus(req.Status)
	if status == "" {
		status = models.ExecutionStatusPending
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "执行记录"
	}
	return models.ExecutionRecord{
		CorrelationID:      strings.TrimSpace(req.CorrelationID),
		Source:             defaultString(req.Source, "script"),
		Category:           strings.TrimSpace(req.Category),
		Name:               name,
		Command:            strings.TrimSpace(req.Command),
		CommandDigest:      digestText(req.Command),
		Status:             status,
		ExitCode:           req.ExitCode,
		CreatedBy:          strings.TrimSpace(req.CreatedBy),
		TriggerUser:        strings.TrimSpace(req.TriggerUser),
		TargetHost:         strings.TrimSpace(req.TargetHost),
		TargetIPs:          models.NewJSONBFromSlice(req.TargetIPs),
		ResourceType:       strings.TrimSpace(req.ResourceType),
		ResourceID:         strings.TrimSpace(req.ResourceID),
		ResourceName:       strings.TrimSpace(req.ResourceName),
		TaskID:             taskID,
		ParentExecutionID:  parentID,
		StdoutSummary:      limitText(req.StdoutSummary, executionSummaryLimit),
		StderrSummary:      limitText(req.StderrSummary, executionSummaryLimit),
		Effects:            rawJSONOrObject(req.Effects),
		Metadata:           rawJSONOrObject(req.Metadata),
		RollbackCapability: normalizeRollbackCapability(req.RollbackCapability),
		RollbackStatus:     models.RollbackStatusNotStarted,
		RollbackPlan:       rawJSONOrObject(req.RollbackPlan),
		RollbackAdvice:     strings.TrimSpace(req.RollbackAdvice),
	}
}

func upsertExecutionStart(tx *gorm.DB, rec *models.ExecutionRecord) error {
	var existing models.ExecutionRecord
	q := tx.Where("correlation_id = ?", rec.CorrelationID)
	if rec.ReportTokenHash != "" {
		q = q.Or("report_token_hash = ?", rec.ReportTokenHash)
	}
	if err := q.First(&existing).Error; err == nil {
		rec.ID = existing.ID
		updates := map[string]interface{}{
			"status":              rec.Status,
			"started_at":          rec.StartedAt,
			"source":              rec.Source,
			"category":            rec.Category,
			"name":                rec.Name,
			"command":             rec.Command,
			"command_digest":      rec.CommandDigest,
			"target_host":         rec.TargetHost,
			"target_ips":          rec.TargetIPs,
			"resource_type":       rec.ResourceType,
			"resource_id":         rec.ResourceID,
			"resource_name":       rec.ResourceName,
			"rollback_capability": rec.RollbackCapability,
			"rollback_plan":       rec.RollbackPlan,
			"rollback_advice":     rec.RollbackAdvice,
			"metadata":            rec.Metadata,
		}
		return tx.Model(&models.ExecutionRecord{}).Where("id = ?", existing.ID).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Create(rec).Error
}

func findAuthorizedExecutionRecord(c *gin.Context, recordID, correlationID, inviteID, token string) (models.ExecutionRecord, bool) {
	if !authorizeExecutionReport(inviteID, token, correlationID, recordID) {
		response.Unauthorized(c, "执行上报 token 无效")
		return models.ExecutionRecord{}, false
	}
	var rec models.ExecutionRecord
	q := database.DB
	switch {
	case strings.TrimSpace(recordID) != "":
		id, err := uuid.Parse(recordID)
		if err != nil {
			response.BadRequest(c, "无效的执行记录 ID")
			return models.ExecutionRecord{}, false
		}
		q = q.Where("id = ?", id)
	case strings.TrimSpace(correlationID) != "":
		q = q.Where("correlation_id = ?", correlationID)
	default:
		response.BadRequest(c, "缺少 record_id 或 correlation_id")
		return models.ExecutionRecord{}, false
	}
	if err := q.First(&rec).Error; response.HandleDBError(c, err, "执行记录不存在") {
		return models.ExecutionRecord{}, false
	}
	return rec, true
}

func authorizeExecutionReport(inviteID, token, correlationID, recordID string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	if inviteID != "" && validateK8sInviteToken(inviteID, token) {
		return true
	}
	var rec models.ExecutionRecord
	q := database.DB
	switch {
	case recordID != "":
		id, err := uuid.Parse(recordID)
		if err != nil {
			return false
		}
		q = q.Where("id = ?", id)
	case correlationID != "":
		q = q.Where("correlation_id = ?", correlationID)
	default:
		return false
	}
	if err := q.First(&rec).Error; err != nil || rec.ReportTokenHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(rec.ReportTokenHash), []byte(hashExecutionToken(token))) == 1
}

func validateK8sInviteToken(inviteID, token string) bool {
	id, err := uuid.Parse(strings.TrimSpace(inviteID))
	if err != nil {
		return false
	}
	var inv models.K8sBundleInvite
	if err := database.DB.Where("id = ?", id).First(&inv).Error; err != nil {
		return false
	}
	if time.Now().After(inv.ExpiresAt) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(inv.DownloadToken), []byte(token)) == 1
}

func findExecutionRollbackImpacts(rec models.ExecutionRecord) []models.ExecutionRecord {
	if rec.Status != models.ExecutionStatusSuccess {
		return nil
	}
	db := database.DB.Model(&models.ExecutionRecord{}).
		Where("id <> ? AND created_at > ? AND status = ?", rec.ID, rec.CreatedAt, models.ExecutionStatusSuccess)
	if rec.TargetHost != "" {
		db = db.Where("target_host = ?", rec.TargetHost)
	} else if rec.ResourceType != "" && rec.ResourceID != "" {
		db = db.Where("resource_type = ? AND resource_id = ?", rec.ResourceType, rec.ResourceID)
	} else {
		return nil
	}
	var impacts []models.ExecutionRecord
	db.Order("created_at ASC").Limit(20).Find(&impacts)
	return impacts
}

func addExecutionEvent(tx *gorm.DB, id uuid.UUID, level, phase, message, output string, details models.JSONB) {
	if level == "" {
		level = "info"
	}
	if phase == "" {
		phase = "progress"
	}
	if message == "" {
		message = "执行事件"
	}
	tx.Create(&models.ExecutionEvent{
		ExecutionID: id,
		Level:       level,
		Phase:       phase,
		Message:     message,
		Output:      limitText(output, executionSummaryLimit),
		Details:     details,
	})
}

func createTaskExecutionRecord(tx *gorm.DB, task models.Task, source, category, command, targetHost string, rollbackPlan map[string]interface{}) {
	capability := models.RollbackCapabilityManual
	advice := "请根据任务输出确认影响范围后执行人工回滚。"
	if rollbackPlan == nil {
		rollbackPlan = map[string]interface{}{}
		capability = models.RollbackCapabilityNone
		advice = "该任务未提供自动回滚计划。"
	}
	rec := models.ExecutionRecord{
		CorrelationID:      task.ID.String(),
		Source:             source,
		Category:           category,
		Name:               task.Name,
		Command:            command,
		CommandDigest:      digestText(command),
		Status:             task.Status,
		CreatedBy:          task.CreatedBy,
		TargetHost:         targetHost,
		TaskID:             &task.ID,
		TargetIPs:          task.TargetIDs,
		Effects:            models.NewJSONBFromMap(map[string]interface{}{}),
		Metadata:           models.NewJSONBFromMap(map[string]interface{}{"task_id": task.ID.String()}),
		RollbackCapability: capability,
		RollbackStatus:     models.RollbackStatusNotStarted,
		RollbackPlan:       models.NewJSONBFromMap(rollbackPlan),
		RollbackAdvice:     advice,
	}
	_ = tx.Create(&rec).Error
}

func syncTaskExecutionRecord(tx *gorm.DB, taskID uuid.UUID) {
	var task models.Task
	if err := tx.Where("id = ?", taskID).First(&task).Error; err != nil {
		return
	}
	var stdout strings.Builder
	var stderr strings.Builder
	var exitCode *int
	var subTasks []models.SubTask
	tx.Where("task_id = ?", taskID).Order("created_at ASC").Find(&subTasks)
	for _, st := range subTasks {
		if st.Output != "" {
			stdout.WriteString(st.ClientID + ":\n" + st.Output + "\n")
		}
		if st.Error != "" {
			stderr.WriteString(st.ClientID + ": " + st.Error + "\n")
		}
		if st.ExitCode != nil && exitCode == nil {
			exitCode = st.ExitCode
		}
	}
	now := time.Now()
	tx.Model(&models.ExecutionRecord{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":         task.Status,
		"exit_code":      exitCode,
		"finished_at":    task.FinishedAt,
		"duration_ms":    durationMillis(task.StartedAt, task.FinishedAt, now),
		"stdout_summary": limitText(stdout.String(), executionSummaryLimit),
		"stderr_summary": limitText(stderr.String(), executionSummaryLimit),
		"effects":        models.NewJSONBFromMap(map[string]interface{}{"success_count": task.SuccessCount, "failed_count": task.FailedCount}),
	})
}

func normalizeExecutionStatus(s string) string {
	switch strings.TrimSpace(s) {
	case models.ExecutionStatusPending, models.ExecutionStatusRunning, models.ExecutionStatusSuccess, models.ExecutionStatusFailed, models.ExecutionStatusCancelled:
		return strings.TrimSpace(s)
	case string(models.TaskStatusDispatched):
		return models.ExecutionStatusRunning
	default:
		return ""
	}
}

func normalizeRollbackCapability(s string) string {
	switch strings.TrimSpace(s) {
	case models.RollbackCapabilityAuto, models.RollbackCapabilityManual, models.RollbackCapabilityNone:
		return strings.TrimSpace(s)
	default:
		return models.RollbackCapabilityNone
	}
}

func rawJSONOrObject(raw json.RawMessage) models.JSONB {
	if len(raw) == 0 || string(raw) == "null" {
		return models.NewJSONBFromMap(map[string]interface{}{})
	}
	return models.JSONB(raw)
}

func parseOptionalUUID(s string) *uuid.UUID {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	return &id
}

func newExecutionReportToken() (string, string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(b)
	return token, hashExecutionToken(token), nil
}

func hashExecutionToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func digestText(s string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(sum[:])
}

func limitText(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "\n...<truncated>"
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func extractRollbackCommand(plan models.JSONB) string {
	var m map[string]interface{}
	if err := json.Unmarshal(plan, &m); err != nil {
		return ""
	}
	if v, ok := m["command"].(string); ok {
		return v
	}
	if v, ok := m["manual_command"].(string); ok {
		return v
	}
	if steps, ok := m["steps"].([]interface{}); ok && len(steps) > 0 {
		if s, ok := steps[0].(string); ok {
			return s
		}
	}
	return ""
}

func durationMillis(start, finish *time.Time, fallbackFinish time.Time) int64 {
	if start == nil {
		return 0
	}
	end := fallbackFinish
	if finish != nil {
		end = *finish
	}
	return end.Sub(*start).Milliseconds()
}
