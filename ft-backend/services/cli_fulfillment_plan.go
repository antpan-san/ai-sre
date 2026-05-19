package services

import (
	"fmt"
	"strings"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	FulfillmentActionSubscriptionRequired = "subscription_required"
	FulfillmentActionGrantedRetry         = "granted_retry"
	FulfillmentActionManualReview         = "manual_review"
	FulfillmentActionAutoIterationCreated = "auto_iteration_created"
	FulfillmentActionAwaitingApproval     = "awaiting_approval"
	FulfillmentActionUnsupported          = "unsupported"
)

// FulfillmentPlanResult is the public CLI-facing response (no secrets / YAML / entitlements).
type FulfillmentPlanResult struct {
	Action               string `json:"action"`
	Message              string `json:"message,omitempty"`
	AutoIterationCreated bool   `json:"auto_iteration_created"`
	AutoIterationID      string `json:"auto_iteration_id,omitempty"`
	RetryAllowed         bool   `json:"retry_allowed"`
}

// HandleCLIFulfillmentPlan decides how to fulfill a capability-layer CLI request.
func HandleCLIFulfillmentPlan(userID uuid.UUID, createdBy, catalogDigest, command, topic, failureKind, failureMessage string, ctx map[string]string, in SkillExecutionIntent) (*FulfillmentPlanResult, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user id required")
	}
	if strings.TrimSpace(catalogDigest) == "" {
		return nil, fmt.Errorf("command_catalog_digest required")
	}
	intent := NormalizeSkillExecutionIntent(topic, ctx, in)
	topic = strings.TrimSpace(intent.Topic)
	if topic == "" {
		topic = strings.TrimSpace(command)
	}
	failureKind = strings.ToLower(strings.TrimSpace(failureKind))
	switch failureKind {
	case "subscription_required", "paywall":
		return &FulfillmentPlanResult{
			Action:  FulfillmentActionSubscriptionRequired,
			Message: publicFulfillmentMessage(failureMessage, "当前订阅不足，请购买对应能力或联系管理员"),
		}, nil
	case "unsupported":
		return &FulfillmentPlanResult{
			Action:  FulfillmentActionUnsupported,
			Message: publicFulfillmentMessage(failureMessage, "当前产品暂不支持该需求，已记录反馈"),
		}, nil
	case "capability_not_found", "":
		return handleCapabilityFulfillment(userID, createdBy, command, topic, ctx, intent)
	default:
		return handleProductGapFulfillment(userID, createdBy, command, topic, failureKind, failureMessage, intent)
	}
}

func handleCapabilityFulfillment(userID uuid.UUID, createdBy, command, topic string, ctx map[string]string, intent SkillExecutionIntent) (*FulfillmentPlanResult, error) {
	gap, err := HandleCLICapabilityGap(userID, createdBy, topic, ctx, intent)
	if err != nil {
		return nil, err
	}
	out := &FulfillmentPlanResult{
		Action:       FulfillmentActionGrantedRetry,
		Message:      publicFulfillmentMessage(gap.Message, "已授予技能包，请重试"),
		RetryAllowed: gap.Granted,
	}
	if gap.AutoIterationID != nil {
		settings, _ := GetAutoIterationSettings()
		row, _ := GetAutoIteration(*gap.AutoIterationID)
		if row != nil && row.Status == models.AutoIterationStatusAwaitingApproval {
			out.Action = FulfillmentActionAwaitingApproval
			out.AutoIterationCreated = true
			out.AutoIterationID = gap.AutoIterationID.String()
			out.Message = "已创建高风险自动迭代任务，等待 super_admin 审批"
			out.RetryAllowed = false
			return out, nil
		}
		if settings != nil && settings.AutoDispatchEnabled && row != nil && row.Status == models.AutoIterationStatusPending {
			out.Action = FulfillmentActionAutoIterationCreated
			out.AutoIterationCreated = true
			out.AutoIterationID = gap.AutoIterationID.String()
			out.Message = publicFulfillmentMessage(gap.Message, "已创建自动迭代任务")
			out.RetryAllowed = gap.Granted
		}
	}
	return out, nil
}

func handleProductGapFulfillment(userID uuid.UUID, createdBy, command, topic, failureKind, failureMessage string, intent SkillExecutionIntent) (*FulfillmentPlanResult, error) {
	settings, err := GetAutoIterationSettings()
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return &FulfillmentPlanResult{
			Action:  FulfillmentActionManualReview,
			Message: "自动迭代未启用，已记录需求，请联系管理员",
		}, nil
	}
	risk, high := assessFulfillmentRisk(command, failureKind, failureMessage, intent)
	title := fmt.Sprintf("CLI 反馈: %s", limitAuditText(topic, 40))
	if title == "CLI 反馈: " {
		title = "CLI 能力反馈"
	}
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		cmd = "ai-sre check " + strings.TrimSpace(intent.Topic)
	}
	status := models.AutoIterationStatusPending
	requiresApproval := high && settings.HighRiskRequiresApproval
	if requiresApproval {
		status = models.AutoIterationStatusAwaitingApproval
	}
	if !settings.AutoDispatchEnabled {
		status = models.AutoIterationStatusDraft
	}
	userBody := strings.TrimSpace(cmd)
	desc, formattedCmd := FormatAutoIterationUserRequirement(title, userBody, topic)
	row := models.AutoIteration{
		Title:                      limitAuditText(title, 200),
		Description:                desc,
		Command:                    formattedCmd,
		Status:                     status,
		Source:                     autoIterationSourceForFailure(failureKind),
		RiskLevel:                  risk,
		RequiresSuperAdminApproval: requiresApproval,
		Topic:                      strings.TrimSpace(topic),
		CreatedByUserID:            &userID,
		CreatedBy:                  limitAuditText(createdBy, 80),
		Metadata:                   MergeAgentTaskMetadata(nil),
	}
	if err := databaseCreateAutoIteration(&row, createdBy, failureKind); err != nil {
		return nil, err
	}
	if settings.DingTalkNotifyEnabled {
		notifyAutoIterationDingTalkKind(DingTalkKindCLIFeedbackQueued, row, "", failureKind)
	}
	action := FulfillmentActionAutoIterationCreated
	msg := "已创建自动迭代任务"
	if status == models.AutoIterationStatusAwaitingApproval {
		action = FulfillmentActionAwaitingApproval
		msg = "已创建高风险任务，等待 super_admin 审批"
	} else if status == models.AutoIterationStatusDraft {
		action = FulfillmentActionManualReview
		msg = "已记录需求（自动派发已关闭）"
	}
	return &FulfillmentPlanResult{
		Action:               action,
		Message:              msg,
		AutoIterationCreated: action == FulfillmentActionAutoIterationCreated || action == FulfillmentActionAwaitingApproval,
		AutoIterationID:      row.ID.String(),
		RetryAllowed:         false,
	}, nil
}

func databaseCreateAutoIteration(row *models.AutoIteration, actor, failureKind string) error {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return appendAutoIterationEvent(tx, row.ID, models.AutoIterationEventStateChange, "cli", actor,
			"CLI fulfillment plan: "+limitAuditText(failureKind, 80), map[string]interface{}{"status": row.Status})
	})
}

func assessFulfillmentRisk(command, failureKind, failureMessage string, intent SkillExecutionIntent) (risk string, high bool) {
	text := strings.ToLower(command + " " + failureKind + " " + failureMessage + " " + intent.Topic + " " + intent.ProblemKey)
	highMarkers := []string{
		"migration", "auth", "billing", "permission", "rbac", "delete data", "drop table",
		"生产部署", "deploy script", "new command", "new flag", "删除", "鉴权", "计费",
		"大范围重构", "refactor entire",
	}
	for _, m := range highMarkers {
		if strings.Contains(text, m) {
			return models.AutoIterationRiskHigh, true
		}
	}
	if failureKind == "bug" {
		return models.AutoIterationRiskMedium, false
	}
	return models.AutoIterationRiskLow, false
}

func publicFulfillmentMessage(raw, fallback string) string {
	msg := strings.TrimSpace(raw)
	if msg == "" {
		return fallback
	}
	deny := []string{"prompt", "yaml", "webhook", "token", "password", "secret", "entitlement", "权益"}
	lower := strings.ToLower(msg)
	for _, d := range deny {
		if strings.Contains(lower, d) {
			return fallback
		}
	}
	return limitAuditText(msg, 500)
}

func autoIterationSourceForFailure(failureKind string) string {
	switch strings.ToLower(strings.TrimSpace(failureKind)) {
	case "diagnosis_insufficient":
		return models.AutoIterationSourceDiagnosisGap
	case "product_gap", "capability_not_found":
		return models.AutoIterationSourceCLIFeedback
	case "rule_candidate":
		return models.AutoIterationSourceRuleCandidate
	case "ai_failure", "ai_cost_reduction":
		return models.AutoIterationSourceAICostReduce
	case "skill_refine":
		return models.AutoIterationSourceSkillRefine
	default:
		return models.AutoIterationSourceCLIFeedback
	}
}
