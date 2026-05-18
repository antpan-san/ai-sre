package services

import (
	"crypto/sha256"
	"encoding/hex"
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
	Page     int
	PageSize int
}

type AutoIterationSettingsView struct {
	Enabled                  bool   `json:"enabled"`
	MaxConcurrent            int    `json:"max_concurrent"`
	HighRiskRequiresApproval bool   `json:"high_risk_requires_approval"`
	GitHubRepo               string `json:"github_repo,omitempty"`
	HasDingTalkWebhook       bool   `json:"has_dingtalk_webhook"`
	Notes                    string `json:"notes,omitempty"`
	UpdatedAt                string `json:"updated_at,omitempty"`
	UpdatedBy                string `json:"updated_by,omitempty"`
}

type CLIFeedbackAnalyzeResult struct {
	FeedbackID      string `json:"feedback_id"`
	Classification  string `json:"classification"`
	NeedIteration   bool   `json:"need_iteration"`
	UserMessage     string `json:"user_message"`
	NextAction      string `json:"next_action"`
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
		GitHubRepo:               cfg.GitHubRepo,
		HasDingTalkWebhook:       cfg.DingTalkWebhook != "",
		Notes:                    row.Notes,
		UpdatedAt:                row.UpdatedAt.Format(time.RFC3339),
		UpdatedBy:                row.UpdatedBy,
	}, nil
}

func UpdateAutoIterationSettings(enabled *bool, maxConcurrent *int, highRisk *bool, notes, updatedBy string) (*AutoIterationSettingsView, error) {
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

func CreateManualAutoIteration(title, description, topic, createdBy string, userID uuid.UUID) (*models.AutoIteration, error) {
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
	row := models.AutoIteration{
		Title:                      limitAuditText(title, 200),
		Description:                limitAuditText(description, 2000),
		Status:                     models.AutoIterationStatusDraft,
		Source:                     models.AutoIterationSourceManual,
		RiskLevel:                  risk,
		RequiresSuperAdminApproval: risk == models.AutoIterationRiskHigh || settings.HighRiskRequiresApproval,
		Topic:                      strings.TrimSpace(topic),
		CreatedByUserID:            &userID,
		CreatedBy:                  limitAuditText(createdBy, 80),
		Metadata:                   models.NewJSONBFromMap(map[string]interface{}{}),
	}
	if row.Title == "" {
		row.Title = "手动迭代任务"
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return appendAutoIterationEvent(tx, row.ID, models.AutoIterationEventStateChange, "super_admin", createdBy, "任务已创建", map[string]interface{}{
			"status": row.Status,
		})
	})
	if err != nil {
		return nil, err
	}
	notifyAutoIterationDingTalk("【自动迭代】新任务", fmt.Sprintf("标题: %s\n状态: %s\n来源: %s", row.Title, row.Status, row.Source))
	return &row, nil
}

func notifyAutoIterationDingTalk(title, body string) {
	if err := SendAutoIterationDingTalk(title, body); err != nil {
		logger.Warn("auto_iteration dingtalk: %v", err)
	}
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
	return transitionAutoIteration(id, "super_admin", actorName,
		[]string{models.AutoIterationStatusDraft, models.AutoIterationStatusPending, models.AutoIterationStatusPaused, models.AutoIterationStatusFailed},
		models.AutoIterationStatusRunning, "迭代已开始", map[string]interface{}{
			"assigned_agent_id": nil,
			"last_error":        "",
		})
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

func ApproveAutoIteration(id uuid.UUID, userID uuid.UUID, actorName, notes string) (*models.AutoIteration, error) {
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		if row.Status != models.AutoIterationStatusAwaitingApproval && row.Status != models.AutoIterationStatusPending {
			return ErrAutoIterationInvalidState
		}
		if row.RiskLevel == models.AutoIterationRiskHigh && row.RequiresSuperAdminApproval {
			// Route already super_admin only.
		}
		now := time.Now().UTC()
		updates := map[string]interface{}{
			"status":              models.AutoIterationStatusApproved,
			"approved_by_user_id": userID,
			"approved_by":         limitAuditText(actorName, 80),
			"approved_at":         &now,
		}
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return err
		}
		row.Status = models.AutoIterationStatusApproved
		return appendAutoIterationEvent(tx, id, models.AutoIterationEventStateChange, "super_admin", actorName,
			"已批准上线: "+limitAuditText(notes, 500), map[string]interface{}{"approved": true})
	})
	if err != nil {
		return nil, err
	}
	notifyAutoIterationDingTalk("【自动迭代】已批准", fmt.Sprintf("标题: %s\n审批人: %s", row.Title, actorName))
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
		if row.Status != models.AutoIterationStatusApproved && row.Status != models.AutoIterationStatusCompleted && row.Status != models.AutoIterationStatusFailed {
			return ErrAutoIterationInvalidState
		}
		if err := tx.Model(&row).Update("status", models.AutoIterationStatusCancelled).Error; err != nil {
			return err
		}
		row.Status = models.AutoIterationStatusCancelled
		return appendAutoIterationEvent(tx, id, models.AutoIterationEventStateChange, "super_admin", actorName,
			"已回滚: "+limitAuditText(reason, 500), map[string]interface{}{"rollback": true})
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
	cfg := config.ResolvedAutoIterationConfig()
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
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
	cfg := config.ResolvedAutoIterationConfig()
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		msg := "钉钉通知已发送"
		sent := false
		if cfg.DingTalkWebhook == "" {
			msg = "未配置钉钉 webhook，跳过通知"
		} else {
			body := fmt.Sprintf("任务: %s\n状态: %s\n风险: %s", row.Title, row.Status, row.RiskLevel)
			if err := SendAutoIterationDingTalk("【自动迭代】通知", body); err != nil {
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
	needIteration := settings.Enabled && (classification == "bug" || classification == "improvement")
	userMessage := "感谢反馈，我们已记录。"
	nextAction := "none"
	if needIteration {
		userMessage = "感谢反馈，平台将评估是否纳入自动迭代（无需进一步操作）。"
		nextAction = "wait_review"
	}
	fb := models.AutoIterationFeedback{
		UserID:         userID,
		CLIBindingID:   bindingID,
		Topic:          strings.TrimSpace(topic),
		Classification: classification,
		NeedIteration:  needIteration,
		UserMessage:    userMessage,
		RawPayload:     models.NewJSONBFromMap(sanitizeFeedbackPayload(payload)),
	}
	if err := database.DB.Create(&fb).Error; err != nil {
		return nil, err
	}
	if needIteration {
		title := fmt.Sprintf("CLI 反馈: %s", limitAuditText(topic, 40))
		if title == "CLI 反馈: " {
			title = "CLI 反馈"
		}
		iter, err := CreateManualAutoIteration(title, summary, topic, "cli_feedback", userID)
		if err == nil && iter != nil {
			fb.AutoIterationID = &iter.ID
			_ = database.DB.Model(&fb).Update("auto_iteration_id", iter.ID)
			_ = database.DB.Model(iter).Updates(map[string]interface{}{
				"source":      models.AutoIterationSourceCLIFeedback,
				"status":      models.AutoIterationStatusPending,
				"command":     limitAuditText(command, 2000),
				"summary":     limitAuditText(summary, 2000),
				"feedback_id": fb.ID,
			})
			if classification == "bug" {
				_ = database.DB.Model(iter).Updates(map[string]interface{}{
					"risk_level":                   models.AutoIterationRiskHigh,
					"requires_super_admin_approval": true,
				})
			}
			notifyAutoIterationDingTalk("【自动迭代】CLI 反馈入队", fmt.Sprintf("标题: %s\n分类: %s\nTopic: %s", iter.Title, classification, topic))
		}
	}
	return &CLIFeedbackAnalyzeResult{
		FeedbackID:     fb.ID.String(),
		Classification: classification,
		NeedIteration:  needIteration,
		UserMessage:    userMessage,
		NextAction:     nextAction,
	}, nil
}

func classifyFeedback(topic, summary string, payload map[string]interface{}) string {
	text := strings.ToLower(topic + " " + summary)
	if strings.Contains(text, "bug") || strings.Contains(text, "error") || strings.Contains(text, "fail") {
		return "bug"
	}
	if strings.Contains(text, "improve") || strings.Contains(text, "feature") {
		return "improvement"
	}
	if payload != nil {
		if v, ok := payload["classification"].(string); ok && v != "" {
			return strings.TrimSpace(v)
		}
	}
	return "general"
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

func CodeAgentHeartbeat(bindingID uuid.UUID) error {
	now := time.Now().UTC()
	return database.DB.Model(&models.CodeAgentBinding{}).Where("id = ?", bindingID).
		Updates(map[string]interface{}{"last_heartbeat_at": &now, "status": models.CodeAgentStatusActive}).Error
}

func CodeAgentPullTask(bindingID uuid.UUID) (*models.AutoIteration, error) {
	var row models.AutoIteration
	err := database.DB.Where("status IN ?", []string{models.AutoIterationStatusPending, models.AutoIterationStatusRunning}).
		Where("assigned_agent_id IS NULL OR assigned_agent_id = ?", bindingID).
		Order("created_at ASC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = database.DB.Model(&row).Updates(map[string]interface{}{
		"assigned_agent_id": bindingID,
		"status":            models.AutoIterationStatusRunning,
	})
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

func CodeAgentReportResult(iterationID, bindingID uuid.UUID, success bool, summary string) error {
	toStatus := models.AutoIterationStatusCompleted
	if !success {
		toStatus = models.AutoIterationStatusFailed
	}
	settings, _ := GetAutoIterationSettings()
	var row models.AutoIteration
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND assigned_agent_id = ?", iterationID, bindingID).First(&row).Error; err != nil {
			return err
		}
		if success && settings != nil && settings.HighRiskRequiresApproval && row.RiskLevel == models.AutoIterationRiskHigh {
			toStatus = models.AutoIterationStatusAwaitingApproval
		}
		if err := tx.Model(&row).Updates(map[string]interface{}{
			"status":  toStatus,
			"summary": limitAuditText(summary, 2000),
		}).Error; err != nil {
			return err
		}
		return appendAutoIterationEvent(tx, iterationID, models.AutoIterationEventStateChange, "worker", "code-agent",
			"Worker 上报结果", map[string]interface{}{"success": success, "status": toStatus})
	})
	if err == nil && toStatus == models.AutoIterationStatusAwaitingApproval {
		notifyAutoIterationDingTalk("【自动迭代】待审批", fmt.Sprintf("标题: %s\n请 super_admin 在控制台批准上线", row.Title))
	}
	return err
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
