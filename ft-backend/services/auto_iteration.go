package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrAutoIterationNotFound     = errors.New("auto_iteration_not_found")
	ErrAutoIterationInvalidState = errors.New("invalid_state")
	ErrAutoIterationDisabled     = errors.New("auto_iteration_disabled")
	ErrHighRiskNeedsSuperAdmin   = errors.New("high_risk_requires_super_admin")
)

type AutoIterationListFilter struct {
	Status   string
	Topic    string
	Source   string
	Keyword  string
	Page     int
	PageSize int
}

type AutoIterationSettingsView struct {
	Enabled                  bool   `json:"enabled"`
	MaxConcurrent            int    `json:"max_concurrent"`
	HighRiskRequiresApproval bool   `json:"high_risk_requires_approval"`
	AutoDispatchEnabled      bool   `json:"auto_dispatch_enabled"`
	LowRiskAutoDeployEnabled bool   `json:"low_risk_auto_deploy_enabled"`
	GitHubSyncEnabled        bool   `json:"github_sync_enabled"`
	DingTalkNotifyEnabled    bool   `json:"dingtalk_notify_enabled"`
	GitHubRepo               string `json:"github_repo,omitempty"`
	HasDingTalkWebhook       bool   `json:"has_dingtalk_webhook"`
	Notes                    string `json:"notes,omitempty"`
	UpdatedAt                string `json:"updated_at,omitempty"`
	UpdatedBy                string `json:"updated_by,omitempty"`
}

type CLIFeedbackAnalyzeResult struct {
	FeedbackID             string `json:"feedback_id"`
	Classification         string `json:"classification"`
	NeedIteration          bool   `json:"need_iteration"`
	UserMessage            string `json:"user_message"`
	NextAction             string `json:"next_action"`
	Action                 string `json:"action,omitempty"`
	AutoIterationCreated   bool   `json:"auto_iteration_created"`
	AutoIterationID        string `json:"auto_iteration_id,omitempty"`
}

func EnsureAutoIterationSettings() error {
	var row models.AutoIterationSettings
	err := database.DB.First(&row, "id = ?", 1).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	cfg := config.ResolvedAutoIterationConfig()
	row = models.AutoIterationSettings{
		ID:                       1,
		Enabled:                  cfg.Enabled,
		MaxConcurrent:            cfg.MaxConcurrent,
		HighRiskRequiresApproval: cfg.HighRiskRequiresApproval,
		AutoDispatchEnabled:      true,
		LowRiskAutoDeployEnabled: false,
		GitHubSyncEnabled:        true,
		DingTalkNotifyEnabled:    true,
		UpdatedAt:                time.Now().UTC(),
	}
	return database.DB.Create(&row).Error
}

func GetAutoIterationSettings() (*AutoIterationSettingsView, error) {
	if err := EnsureAutoIterationSettings(); err != nil {
		return nil, err
	}
	var row models.AutoIterationSettings
	if err := database.DB.First(&row, "id = ?", 1).Error; err != nil {
		return nil, err
	}
	cfg := config.ResolvedAutoIterationConfig()
	return &AutoIterationSettingsView{
		Enabled:                  row.Enabled,
		MaxConcurrent:            row.MaxConcurrent,
		HighRiskRequiresApproval: row.HighRiskRequiresApproval,
		AutoDispatchEnabled:      row.AutoDispatchEnabled,
		LowRiskAutoDeployEnabled: row.LowRiskAutoDeployEnabled,
		GitHubSyncEnabled:        row.GitHubSyncEnabled,
		DingTalkNotifyEnabled:    row.DingTalkNotifyEnabled,
		GitHubRepo:               cfg.GitHubRepo,
		HasDingTalkWebhook:       cfg.DingTalkWebhook != "",
		Notes:                    row.Notes,
		UpdatedAt:                row.UpdatedAt.Format(time.RFC3339),
		UpdatedBy:                row.UpdatedBy,
	}, nil
}

func UpdateAutoIterationSettings(enabled *bool, maxConcurrent *int, highRisk *bool, autoDispatch, lowRiskDeploy, githubSync, dingTalk *bool, notes, updatedBy string) (*AutoIterationSettingsView, error) {
	if err := EnsureAutoIterationSettings(); err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
		"updated_by": limitAuditText(updatedBy, 80),
	}
	if enabled != nil {
		updates["enabled"] = *enabled
	}
	if maxConcurrent != nil && *maxConcurrent > 0 {
		updates["max_concurrent"] = *maxConcurrent
	}
	if highRisk != nil {
		updates["high_risk_requires_approval"] = *highRisk
	}
	if autoDispatch != nil {
		updates["auto_dispatch_enabled"] = *autoDispatch
	}
	if lowRiskDeploy != nil {
		updates["low_risk_auto_deploy_enabled"] = *lowRiskDeploy
	}
	if githubSync != nil {
		updates["github_sync_enabled"] = *githubSync
	}
	if dingTalk != nil {
		updates["dingtalk_notify_enabled"] = *dingTalk
	}
	if notes != "" {
		updates["notes"] = limitAuditText(notes, 2000)
	}
	if err := database.DB.Model(&models.AutoIterationSettings{}).Where("id = ?", 1).Updates(updates).Error; err != nil {
		return nil, err
	}
	return GetAutoIterationSettings()
}

func ListAutoIterations(filter AutoIterationListFilter) ([]models.AutoIteration, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	q := database.DB.Model(&models.AutoIteration{})
	if s := strings.TrimSpace(filter.Status); s != "" {
		q = q.Where("status = ?", s)
	}
	if t := strings.TrimSpace(filter.Topic); t != "" {
		q = q.Where("topic = ?", t)
	}
	if src := strings.TrimSpace(filter.Source); src != "" {
		q = q.Where("source = ?", src)
	}
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("title ILIKE ? OR description ILIKE ? OR command ILIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.AutoIteration
	if err := q.Order("created_at DESC").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func GetAutoIteration(id uuid.UUID) (*models.AutoIteration, error) {
	var row models.AutoIteration
	if err := database.DB.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func appendAutoIterationEvent(tx *gorm.DB, iterationID uuid.UUID, eventType, actorType, actorName, message string, payload map[string]interface{}) error {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	ev := models.AutoIterationEvent{
		AutoIterationID: iterationID,
		EventType:       eventType,
		ActorType:       actorType,
		ActorName:       limitAuditText(actorName, 80),
		Message:         limitAuditText(message, 4000),
		Payload:         models.NewJSONBFromMap(payload),
	}
	return tx.Create(&ev).Error
}

func CreateManualAutoIteration(title, description, command, topic, createdBy string, userID uuid.UUID, autoStart bool) (*models.AutoIteration, error) {
	settings, err := GetAutoIterationSettings()
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, ErrAutoIterationDisabled
	}
	risk := models.AutoIterationRiskLow
	if settings.HighRiskRequiresApproval {
		risk = models.AutoIterationRiskMedium
	}
	userBody := strings.TrimSpace(command)
	if userBody == "" {
		userBody = strings.TrimSpace(description)
	}
	desc, cmd := FormatAutoIterationUserRequirement(title, userBody, topic)
	initialStatus := models.AutoIterationStatusDraft
	createMsg := "任务已创建（草稿）"
	if autoStart {
		initialStatus = models.AutoIterationStatusPending
		createMsg = "任务已提交，等待本机 Worker 拉取"
	}
	row := models.AutoIteration{
		Title:                      limitAuditText(title, 200),
		Description:                desc,
		Command:                    cmd,
		Status:                     initialStatus,
		Source:                     models.AutoIterationSourceManual,
		RiskLevel:                  risk,
		RequiresSuperAdminApproval: risk == models.AutoIterationRiskHigh || settings.HighRiskRequiresApproval,
		Topic:                      strings.TrimSpace(topic),
		CreatedByUserID:            &userID,
		CreatedBy:                  limitAuditText(createdBy, 80),
		Metadata:                   MergeAgentTaskMetadata(nil),
	}
	if row.Title == "" {
		row.Title = "手动迭代任务"
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return appendAutoIterationEvent(tx, row.ID, models.AutoIterationEventStateChange, "super_admin", createdBy, createMsg, map[string]interface{}{
			"status":     row.Status,
			"auto_start": autoStart,
		})
	})
	if err != nil {
		return nil, err
	}
	notifyAutoIterationDingTalkKind(DingTalkKindTaskCreated, row, "", "")
	return &row, nil
}

func notifyAutoIterationDingTalkKind(kind AutoIterationDingTalkKind, row models.AutoIteration, summary, actor string) {
	settings, err := GetAutoIterationSettings()
	if err == nil && settings != nil && !settings.DingTalkNotifyEnabled {
		return
	}
	f := autoIterationDingTalkFields{
		Title:   row.Title,
		TaskID:  row.ID.String(),
		Source:  autoIterationSourceLabelForDingTalk(row.Source),
		Topic:   strings.TrimSpace(row.Topic),
		Status:  row.Status,
		Summary: summary,
		Actor:   actor,
	}
	if err := SendAutoIterationDingTalkMarkdown(kind, f); err != nil {
		logger.Warn("auto_iteration dingtalk: %v", err)
	}
}

// notifyAutoIterationWorkerResult sends DingTalk when a worker finishes (completed / failed / awaiting_approval).
func notifyAutoIterationWorkerResult(row models.AutoIteration, status, summary string) {
	var kind AutoIterationDingTalkKind
	switch status {
	case models.AutoIterationStatusCompleted:
		kind = DingTalkKindWorkerCompleted
	case models.AutoIterationStatusFailed:
		kind = DingTalkKindWorkerFailed
	case models.AutoIterationStatusAwaitingApproval:
		kind = DingTalkKindWorkerAwaitingReview
	default:
		return
	}
	notifyAutoIterationDingTalkKind(kind, row, summary, "")
}

func transitionAutoIteration(id uuid.UUID, actorType, actorName string, allowedFrom []string, toStatus, message string, extra map[string]interface{}) (*models.AutoIteration, error) {
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		ok := false
		for _, s := range allowedFrom {
			if row.Status == s {
				ok = true
				break
			}
		}
		if !ok {
			return ErrAutoIterationInvalidState
		}
		updates := map[string]interface{}{"status": toStatus}
		if extra != nil {
			for k, v := range extra {
				updates[k] = v
			}
		}
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return err
		}
		row.Status = toStatus
		payload := map[string]interface{}{"from": allowedFrom, "to": toStatus}
		return appendAutoIterationEvent(tx, id, models.AutoIterationEventStateChange, actorType, actorName, message, payload)
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func StartAutoIteration(id uuid.UUID, actorName string) (*models.AutoIteration, error) {
	var row models.AutoIteration
	if err := database.DB.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	// Already queued or in progress — idempotent success (UI may still offer「启动」).
	switch row.Status {
	case models.AutoIterationStatusPending:
		return &row, nil
	case models.AutoIterationStatusRunning:
		return &row, nil
	case models.AutoIterationStatusDraft, models.AutoIterationStatusPaused:
		return transitionAutoIteration(id, "super_admin", actorName,
			[]string{row.Status},
			models.AutoIterationStatusPending, "已加入队列，等待本机 Worker 拉取", nil)
	default:
		return nil, ErrAutoIterationInvalidState
	}
}

func PauseAutoIteration(id uuid.UUID, actorName string) (*models.AutoIteration, error) {
	return transitionAutoIteration(id, "super_admin", actorName,
		[]string{models.AutoIterationStatusRunning},
		models.AutoIterationStatusPaused, "迭代已暂停", nil)
}

func ResumeAutoIteration(id uuid.UUID, actorName string) (*models.AutoIteration, error) {
	return transitionAutoIteration(id, "super_admin", actorName,
		[]string{models.AutoIterationStatusPaused},
		models.AutoIterationStatusRunning, "迭代已继续", nil)
}

func CancelAutoIteration(id uuid.UUID, actorName string) (*models.AutoIteration, error) {
	return transitionAutoIteration(id, "super_admin", actorName,
		[]string{models.AutoIterationStatusDraft, models.AutoIterationStatusPending, models.AutoIterationStatusRunning, models.AutoIterationStatusPaused, models.AutoIterationStatusAwaitingApproval},
		models.AutoIterationStatusCancelled, "迭代已取消", nil)
}

func ApproveAutoIteration(id uuid.UUID, userID uuid.UUID, actorName, notes string, force bool) (*models.AutoIteration, error) {
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		allowed := row.Status == models.AutoIterationStatusAwaitingApproval
		if force && row.Status == models.AutoIterationStatusPending {
			allowed = true
		}
		if !allowed {
			return ErrAutoIterationInvalidState
		}
		if row.RiskLevel == models.AutoIterationRiskHigh && row.RequiresSuperAdminApproval {
			// Route already super_admin only.
		}
		now := time.Now().UTC()
		toStatus := models.AutoIterationStatusApproved
		if row.Status == models.AutoIterationStatusAwaitingApproval && row.AssignedAgentID == nil {
			toStatus = models.AutoIterationStatusPending
		}
		updates := map[string]interface{}{
			"status":              toStatus,
			"approved_by_user_id": userID,
			"approved_by":         limitAuditText(actorName, 80),
			"approved_at":         &now,
		}
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return err
		}
		row.Status = toStatus
		return appendAutoIterationEvent(tx, id, models.AutoIterationEventStateChange, "super_admin", actorName,
			"已批准上线: "+limitAuditText(notes, 500), map[string]interface{}{"approved": true})
	})
	if err != nil {
		return nil, err
	}
	notifyAutoIterationDingTalkKind(DingTalkKindApproved, row, "", actorName)
	return &row, nil
}

func RejectAutoIteration(id uuid.UUID, actorName, reason string) (*models.AutoIteration, error) {
	return transitionAutoIteration(id, "super_admin", actorName,
		[]string{models.AutoIterationStatusAwaitingApproval, models.AutoIterationStatusPending},
		models.AutoIterationStatusRejected, "已驳回: "+limitAuditText(reason, 500),
		map[string]interface{}{"rejected_reason": limitAuditText(reason, 2000)})
}

func RollbackAutoIteration(id uuid.UUID, actorName, reason string) (*models.AutoIteration, error) {
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		allowed := map[string]bool{
			models.AutoIterationStatusApproved:         true,
			models.AutoIterationStatusCompleted:        true,
			models.AutoIterationStatusFailed:           true,
			models.AutoIterationStatusRollbackRequired: true,
		}
		if !allowed[row.Status] {
			return ErrAutoIterationInvalidState
		}
		toStatus := models.AutoIterationStatusRolledBack
		if row.Status != models.AutoIterationStatusRollbackRequired {
			toStatus = models.AutoIterationStatusRollbackRequired
		}
		if err := tx.Model(&row).Update("status", toStatus).Error; err != nil {
			return err
		}
		row.Status = toStatus
		return appendAutoIterationEvent(tx, id, models.AutoIterationEventStateChange, "super_admin", actorName,
			"回滚状态: "+toStatus+": "+limitAuditText(reason, 500), map[string]interface{}{"rollback": true, "status": toStatus})
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func RunAutoIterationTests(id uuid.UUID, actorName string) (*models.AutoIteration, error) {
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		return appendAutoIterationEvent(tx, id, models.AutoIterationEventTest, "super_admin", actorName, "已触发重新测试", map[string]interface{}{
			"queued": true,
		})
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func SyncAutoIterationGitHub(id uuid.UUID, actorName string) (*models.AutoIteration, error) {
	settings, err := GetAutoIterationSettings()
	if err != nil {
		return nil, err
	}
	if settings != nil && !settings.GitHubSyncEnabled {
		return nil, fmt.Errorf("github sync disabled in settings")
	}
	cfg := config.ResolvedAutoIterationConfig()
	var row models.AutoIteration
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		meta := mergeAutoIterationMeta(row.Metadata, map[string]interface{}{
			"github_sync": "queued",
		})
		if err := tx.Model(&row).Update("metadata", meta).Error; err != nil {
			return err
		}
		row.Metadata = meta
		return appendAutoIterationEvent(tx, id, models.AutoIterationEventWorker, "super_admin", actorName, "GitHub 同步已排队", map[string]interface{}{
			"github_repo": cfg.GitHubRepo != "",
		})
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func ResendAutoIterationNotification(id uuid.UUID, actorName string) (*models.AutoIteration, error) {
	settings, err := GetAutoIterationSettings()
	if err != nil {
		return nil, err
	}
	if settings != nil && !settings.DingTalkNotifyEnabled {
		return nil, fmt.Errorf("dingtalk notify disabled in settings")
	}
	cfg := config.ResolvedAutoIterationConfig()
	var row models.AutoIteration
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		msg := "钉钉通知已发送"
		sent := false
		if cfg.DingTalkWebhook == "" {
			msg = "未配置钉钉 webhook，跳过通知"
		} else {
			if err := SendAutoIterationDingTalkMarkdown(DingTalkKindResend, autoIterationDingTalkFields{
				Title:  row.Title,
				TaskID: row.ID.String(),
				Source: autoIterationSourceLabelForDingTalk(row.Source),
				Topic:  strings.TrimSpace(row.Topic),
				Status: row.Status,
				Extra:  "风险: " + row.RiskLevel,
			}); err != nil {
				msg = "钉钉发送失败"
			} else {
				sent = true
			}
		}
		return appendAutoIterationEvent(tx, id, models.AutoIterationEventNotification, "super_admin", actorName, msg, map[string]interface{}{
			"resent": true,
			"sent":   sent,
		})
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func ListAutoIterationEvents(iterationID uuid.UUID, afterID uuid.UUID, limit int) ([]models.AutoIterationEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := database.DB.Where("auto_iteration_id = ?", iterationID).Order("created_at ASC").Limit(limit)
	if afterID != uuid.Nil {
		var anchor models.AutoIterationEvent
		if err := database.DB.Where("id = ?", afterID).First(&anchor).Error; err == nil {
			q = q.Where("created_at > ?", anchor.CreatedAt)
		}
	}
	var rows []models.AutoIterationEvent
	return rows, q.Find(&rows).Error
}

func AnalyzeCLIFeedback(userID uuid.UUID, bindingID *uuid.UUID, topic, command, summary string, payload map[string]interface{}) (*CLIFeedbackAnalyzeResult, error) {
	settings, err := GetAutoIterationSettings()
	if err != nil {
		return nil, err
	}
	classification := classifyFeedback(topic, summary, payload)
	needIteration := settings.Enabled && iterationClassificationsNeedTask(classification)
	userMessage := "感谢反馈，我们已记录。"
	nextAction := "none"
	action := ""
	var autoID string
	autoCreated := false

	fb := models.AutoIterationFeedback{
		UserID:         userID,
		CLIBindingID:   bindingID,
		Topic:          strings.TrimSpace(topic),
		Classification: classification,
		NeedIteration:  needIteration,
		UserMessage:    userMessage,
		RawPayload:     models.NewJSONBFromMap(enrichFeedbackPayload(topic, command, summary, payload)),
	}
	if err := database.DB.Create(&fb).Error; err != nil {
		return nil, err
	}
	if needIteration {
		failureKind := classification
		switch failureKind {
		case "improvement":
			failureKind = "product_gap"
		case "diagnosis_insufficient", "ai_failure":
			// keep as-is for title/description
		}
		plan, planErr := handleProductGapFulfillment(userID, "cli_feedback", command, topic, failureKind, summary, SkillExecutionIntent{Topic: topic})
		if planErr == nil && plan != nil {
			action = plan.Action
			autoCreated = plan.AutoIterationCreated
			autoID = plan.AutoIterationID
			userMessage = publicFulfillmentMessage(plan.Message, userMessage)
			switch plan.Action {
			case FulfillmentActionAutoIterationCreated, FulfillmentActionAwaitingApproval:
				nextAction = plan.Action
				if autoID != "" {
					if uid, err := uuid.Parse(autoID); err == nil {
						fb.AutoIterationID = &uid
						_ = database.DB.Model(&fb).Updates(map[string]interface{}{
							"auto_iteration_id": uid,
							"user_message":      userMessage,
						})
						_ = database.DB.Model(&models.AutoIteration{}).Where("id = ?", uid).Update("feedback_id", fb.ID).Error
						if execID := feedbackExecutionID(payload); execID != "" {
							_ = PatchExecutionRecordMetadata(execID, map[string]interface{}{
								"auto_iteration_id": autoID,
								"feedback_id":       fb.ID.String(),
							})
						}
					}
				}
			case FulfillmentActionManualReview:
				nextAction = "manual_review"
			default:
				nextAction = "wait_review"
			}
		} else {
			userMessage = "感谢反馈，平台将评估是否纳入自动迭代。"
			nextAction = "wait_review"
		}
	}
	return &CLIFeedbackAnalyzeResult{
		FeedbackID:           fb.ID.String(),
		Classification:       classification,
		NeedIteration:        needIteration,
		UserMessage:          userMessage,
		NextAction:           nextAction,
		Action:               action,
		AutoIterationCreated: autoCreated,
		AutoIterationID:      autoID,
	}, nil
}

func classifyFeedback(topic, summary string, payload map[string]interface{}) string {
	if payload != nil {
		if v, ok := payload["classification"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	text := strings.ToLower(topic + " " + summary)
	if strings.Contains(text, "bug") || strings.Contains(text, "panic") {
		return "bug"
	}
	if strings.Contains(text, "diagnosis_insufficient") || strings.Contains(text, "证据不足") || strings.Contains(text, "信息不足") {
		return "diagnosis_insufficient"
	}
	if strings.Contains(text, "ai_failure") || strings.Contains(text, "ai 调用失败") {
		return "ai_failure"
	}
	if strings.Contains(text, "product_gap") || strings.Contains(text, "improve") || strings.Contains(text, "feature") || strings.Contains(text, "能力缺口") {
		return "product_gap"
	}
	if strings.Contains(text, "error") || strings.Contains(text, "fail") {
		return "bug"
	}
	return "general"
}

func iterationClassificationsNeedTask(classification string) bool {
	switch strings.TrimSpace(classification) {
	case "bug", "improvement", "product_gap", "diagnosis_insufficient", "ai_failure":
		return true
	default:
		return false
	}
}

func sanitizeFeedbackPayload(payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(payload))
	deny := []string{"agent", "github", "webhook", "token", "password", "secret", "backup", "dingtalk"}
	for k, v := range payload {
		kl := strings.ToLower(k)
		skip := false
		for _, d := range deny {
			if strings.Contains(kl, d) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out[k] = v
	}
	return out
}

func enrichFeedbackPayload(topic, command, summary string, payload map[string]interface{}) map[string]interface{} {
	out := sanitizeFeedbackPayload(payload)
	if strings.TrimSpace(topic) != "" {
		out["topic"] = strings.TrimSpace(topic)
	}
	if strings.TrimSpace(command) != "" {
		out["command"] = limitAuditText(command, 500)
	}
	if strings.TrimSpace(summary) != "" {
		out["summary"] = limitAuditText(summary, 500)
	}
	for _, k := range []string{"request_id", "execution_id", "evidence_digest", "root_cause_digest", "recommendation_digest", "used_ai", "rule_hit", "evidence_completeness", "classification"} {
		if payload == nil {
			continue
		}
		if v, ok := payload[k]; ok && v != nil && fmt.Sprint(v) != "" {
			out[k] = v
		}
	}
	return out
}

func feedbackExecutionID(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload["execution_id"].(string); ok {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fmt.Sprint(payload["execution_id"]))
}

func CodeAgentHeartbeat(bindingID uuid.UUID) error {
	now := time.Now().UTC()
	return database.DB.Model(&models.CodeAgentBinding{}).Where("id = ?", bindingID).
		Updates(map[string]interface{}{"last_heartbeat_at": &now, "status": models.CodeAgentStatusActive}).Error
}

func CountRunningAutoIterations() (int64, error) {
	var n int64
	err := database.DB.Model(&models.AutoIteration{}).
		Where("status = ?", models.AutoIterationStatusRunning).
		Count(&n).Error
	return n, err
}

func CodeAgentPullTask(bindingID uuid.UUID) (*models.AutoIteration, error) {
	settings, err := GetAutoIterationSettings()
	if err != nil {
		return nil, err
	}
	if settings != nil && !settings.AutoDispatchEnabled {
		return nil, nil
	}
	maxConcurrent := 2
	if settings != nil && settings.MaxConcurrent > 0 {
		maxConcurrent = settings.MaxConcurrent
	}
	running, err := CountRunningAutoIterations()
	if err != nil {
		return nil, err
	}
	if running >= int64(maxConcurrent) {
		return nil, nil
	}
	var row models.AutoIteration
	err = database.DB.Where("status = ?", models.AutoIterationStatusPending).
		Where("assigned_agent_id IS NULL").
		Where("requires_super_admin_approval = ?", false).
		Order("created_at ASC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Reclaim orphaned running tasks (e.g. manual Start set running without agent).
		err = database.DB.Where("status = ?", models.AutoIterationStatusRunning).
			Where("assigned_agent_id IS NULL").
			Order("created_at ASC").First(&row).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = database.DB.Where("status = ?", models.AutoIterationStatusRunning).
			Where("assigned_agent_id = ?", bindingID).
			Order("created_at ASC").First(&row).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{"assigned_agent_id": bindingID}
	if row.Status == models.AutoIterationStatusPending {
		updates["status"] = models.AutoIterationStatusRunning
	}
	_ = database.DB.Model(&row).Updates(updates)
	if row.Status == models.AutoIterationStatusPending {
		row.Status = models.AutoIterationStatusRunning
	}
	return &row, nil
}

func CodeAgentReportEvent(iterationID, bindingID uuid.UUID, message string, payload map[string]interface{}) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var row models.AutoIteration
		if err := tx.Where("id = ? AND assigned_agent_id = ?", iterationID, bindingID).First(&row).Error; err != nil {
			return err
		}
		return appendAutoIterationEvent(tx, iterationID, models.AutoIterationEventWorker, "worker", "code-agent", message, payload)
	})
}

// CodeAgentTaskResult carries worker completion details (public fields only).
type CodeAgentTaskResult struct {
	Success                 bool
	Summary                 string
	GitHubSync              string // ok | failed | skipped
	DeployStatus            string // ok | failed | skipped
	RollbackRequired        bool
	SkillPackEnhanced       string // yes | no | partial
	SkillPackEnhanceReason  string
}

func CodeAgentReportResult(iterationID, bindingID uuid.UUID, result CodeAgentTaskResult) error {
	toStatus := models.AutoIterationStatusCompleted
	if !result.Success {
		if result.RollbackRequired {
			toStatus = models.AutoIterationStatusRollbackRequired
		} else {
			toStatus = models.AutoIterationStatusFailed
		}
	}
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND assigned_agent_id = ?", iterationID, bindingID).First(&row).Error; err != nil {
			return err
		}
		if result.Success && row.RequiresSuperAdminApproval {
			toStatus = models.AutoIterationStatusAwaitingApproval
		}
		meta := mergeAutoIterationMeta(row.Metadata, map[string]interface{}{
			"github_sync":   strings.TrimSpace(result.GitHubSync),
			"deploy_status": strings.TrimSpace(result.DeployStatus),
		})
		if v := strings.TrimSpace(result.SkillPackEnhanced); v != "" {
			meta = mergeAutoIterationMeta(meta, map[string]interface{}{
				"skill_pack_enhanced": v,
			})
		}
		if v := strings.TrimSpace(result.SkillPackEnhanceReason); v != "" {
			meta = mergeAutoIterationMeta(meta, map[string]interface{}{
				"skill_pack_enhance_reason": limitAuditText(v, 500),
			})
		}
		if result.GitHubSync == "failed" {
			meta = mergeAutoIterationMeta(meta, map[string]interface{}{"github_sync_retry": true})
		}
		updates := map[string]interface{}{
			"status":  toStatus,
			"summary": limitAuditText(result.Summary, 2000),
			"metadata": meta,
		}
		if !result.Success && result.Summary != "" {
			updates["last_error"] = limitAuditText(result.Summary, 2000)
		}
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return err
		}
		row.Status = toStatus
		row.Metadata = meta
		return appendAutoIterationEvent(tx, iterationID, models.AutoIterationEventStateChange, "worker", "code-agent",
			"Worker 上报结果", map[string]interface{}{
				"success":           result.Success,
				"status":            toStatus,
				"github_sync":       result.GitHubSync,
				"deploy_status":     result.DeployStatus,
				"rollback_required": result.RollbackRequired,
			})
	})
	if err == nil {
		notifyAutoIterationWorkerResult(row, toStatus, result.Summary)
	}
	return err
}

func mergeAutoIterationMeta(existing models.JSONB, patch map[string]interface{}) models.JSONB {
	base := map[string]interface{}{}
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &base)
	}
	for k, v := range patch {
		if strings.TrimSpace(fmt.Sprint(v)) == "" {
			continue
		}
		base[k] = v
	}
	return models.NewJSONBFromMap(base)
}

func ResolveCodeAgentBinding(tokenHash, fingerprint string) (*models.CodeAgentBinding, error) {
	var row models.CodeAgentBinding
	err := database.DB.Where("token_hash = ? AND fingerprint_hash = ? AND status = ?",
		tokenHash, fingerprint, models.CodeAgentStatusActive).First(&row).Error
	return &row, err
}

func EnsureCodeAgentBinding(tokenPlain, fingerprintPlain string) error {
	tokenPlain = strings.TrimSpace(tokenPlain)
	fingerprintPlain = strings.TrimSpace(fingerprintPlain)
	if tokenPlain == "" || len(fingerprintPlain) != 64 {
		return fmt.Errorf("invalid agent bootstrap credentials")
	}
	th := hashSecretForAgent(tokenPlain)
	fh := hashSecretForAgent(fingerprintPlain)
	var row models.CodeAgentBinding
	err := database.DB.Where("token_hash = ? AND fingerprint_hash = ?", th, fh).First(&row).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return database.DB.Create(&models.CodeAgentBinding{
		Name:            "default-agent",
		TokenHash:       th,
		FingerprintHash: fh,
		Status:          models.CodeAgentStatusActive,
	}).Error
}

func hashSecretForAgent(s string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(sum[:])
}

// HashSecretForAgent is exported for middleware.
func HashSecretForAgent(s string) string { return hashSecretForAgent(s) }

// CodeAgentTaskView is the payload returned to the local code-agent worker.
func CodeAgentTaskView(t *models.AutoIteration) map[string]interface{} {
	if t == nil {
		return nil
	}
	out := map[string]interface{}{
		"id":                              t.ID.String(),
		"title":                           t.Title,
		"description":                     t.Description,
		"topic":                           t.Topic,
		"command":                         t.Command,
		"status":                          t.Status,
		"source":                          t.Source,
		"risk_level":                      t.RiskLevel,
		"summary":                         t.Summary,
		"requires_super_admin_approval":   t.RequiresSuperAdminApproval,
		"dev_spec":                        AutoIterationDevSpecVer,
		"dev_skill":                       AutoIterationDevSkillPath,
		"release_skill":                   ".cursor/skills/release-deploy/SKILL.md",
	}
	if t.FeedbackID != nil {
		out["feedback_id"] = t.FeedbackID.String()
	}
	if len(t.Metadata) > 0 {
		out["metadata"] = t.Metadata
	}
	if settings, err := GetAutoIterationSettings(); err == nil && settings != nil {
		out["worker_options"] = map[string]interface{}{
			"github_sync_enabled":         settings.GitHubSyncEnabled,
			"low_risk_auto_deploy_enabled": settings.LowRiskAutoDeployEnabled,
			"dingtalk_notify_enabled":     settings.DingTalkNotifyEnabled,
		}
	}
	return out
}
