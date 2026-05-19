package services

import (
	"errors"
	"fmt"
	"strings"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const EntitlementSourceCapabilityGap = "capability_gap"

// CapabilityGapResult is returned to CLI after a missing-capability request.
type CapabilityGapResult struct {
	PackKey         string     `json:"pack_key"`
	NodePath        string     `json:"node_path"`
	SkillKey        string     `json:"skill_key,omitempty"`
	Granted         bool       `json:"granted"`
	TreeSynced      bool       `json:"tree_synced"`
	AutoIterationID *uuid.UUID `json:"auto_iteration_id,omitempty"`
	Message         string     `json:"message,omitempty"`
}

// HandleCLICapabilityGap syncs tree nodes, grants permanent pack access, and optionally starts auto-iteration.
func HandleCLICapabilityGap(userID uuid.UUID, createdBy string, topic string, ctx map[string]string, in SkillExecutionIntent) (*CapabilityGapResult, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user id required")
	}
	intent := NormalizeSkillExecutionIntent(topic, ctx, in)
	packKey := strings.TrimSpace(intent.PackKey)
	if packKey == "" {
		packKey = packKeyForSkillTopic(intent.Topic)
	}
	if packKey == "" {
		return nil, fmt.Errorf("无法推断技能包")
	}
	treeSynced := false
	if err := SyncMissingBuiltinSkillTreeNodes(); err != nil {
		logger.Warn("HandleCLICapabilityGap: sync tree nodes: %v", err)
	} else {
		treeSynced = true
	}
	_ = SeedSkillCommercialProducts()
	nodePath := strings.TrimSpace(intent.NodePath)
	if nodePath == "" {
		nodePath = strings.TrimSpace(intent.CandidateNodePath)
	}
	if nodePath == "" {
		if n, ok := inferSkillTreeNode(intent.Topic, ctx); ok {
			nodePath = n.Path
			intent = mergeIntentWithNode(intent, n)
		}
	}
	if err := GrantPackEntitlement(userID, packKey, EntitlementSourceCapabilityGap); err != nil {
		return nil, err
	}
	out := &CapabilityGapResult{
		PackKey:    packKey,
		NodePath:   nodePath,
		SkillKey:   intent.SkillKey,
		Granted:    true,
		TreeSynced: treeSynced,
		Message:    "已授予该技能包永久使用权（来源: CLI 能力缺口）",
	}
	settings, err := GetAutoIterationSettings()
	if err != nil || !settings.Enabled {
		return out, nil
	}
	title := fmt.Sprintf("CLI 能力缺口: %s/%s", strings.TrimSpace(intent.Topic), strings.TrimSpace(intent.ProblemKey))
	cmd := strings.TrimSpace(intent.CommandKind)
	if cmd == "" {
		cmd = "ai-sre check " + strings.TrimSpace(intent.Topic)
	}
	row, iterErr := CreateCapabilityGapAutoIteration(title, cmd, intent.Topic, createdBy, userID)
	if iterErr != nil {
		if !errors.Is(iterErr, ErrAutoIterationDisabled) {
			logger.Warn("HandleCLICapabilityGap: auto iteration: %v", iterErr)
		}
		return out, nil
	}
	out.AutoIterationID = &row.ID
	out.Message = out.Message + "；已创建自动迭代任务 " + row.ID.String()
	return out, nil
}

// GrantPackEntitlement upserts a permanent pack entitlement for the user.
func GrantPackEntitlement(userID uuid.UUID, packKey, source string) error {
	packKey = strings.TrimSpace(packKey)
	source = strings.TrimSpace(source)
	if source == "" {
		source = EntitlementSourceCapabilityGap
	}
	if userID == uuid.Nil || packKey == "" {
		return fmt.Errorf("invalid grant")
	}
	if !models.IsKnownPackKey(packKey) {
		return fmt.Errorf("unknown pack key: %s", packKey)
	}
	var ent models.Entitlement
	err := database.DB.Where("user_id = ? AND feature_key = ? AND source = ?", userID, packKey, source).First(&ent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ent = models.Entitlement{
			ID:         uuid.New(),
			UserID:     userID,
			FeatureKey: packKey,
			Source:     source,
		}
		return database.DB.Create(&ent).Error
	}
	if err != nil {
		return err
	}
	ent.ValidUntil = nil
	return database.DB.Save(&ent).Error
}

func CreateCapabilityGapAutoIteration(title, command, topic, createdBy string, userID uuid.UUID) (*models.AutoIteration, error) {
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
	desc, cmd := FormatAutoIterationUserRequirement(title, userBody, topic)
	row := models.AutoIteration{
		Title:                      limitAuditText(title, 200),
		Description:                desc,
		Command:                    cmd,
		Status:                     models.AutoIterationStatusPending,
		Source:                     models.AutoIterationSourceCapabilityGap,
		RiskLevel:                  risk,
		RequiresSuperAdminApproval: risk == models.AutoIterationRiskHigh || settings.HighRiskRequiresApproval,
		Topic:                      strings.TrimSpace(topic),
		CreatedByUserID:            &userID,
		CreatedBy:                  limitAuditText(createdBy, 80),
		Metadata:                   MergeAgentTaskMetadata(nil),
	}
	if row.Title == "" {
		row.Title = "CLI 能力缺口迭代"
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return appendAutoIterationEvent(tx, row.ID, models.AutoIterationEventStateChange, "cli", createdBy, "CLI 触发能力缺口，已提交自动迭代", map[string]interface{}{
			"status": row.Status,
			"topic":  row.Topic,
		})
	})
	if err != nil {
		return nil, err
	}
	notifyAutoIterationDingTalkKind(DingTalkKindTaskCreated, row, "", createdBy)
	return &row, nil
}
