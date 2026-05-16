package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SkillAssetListItem is a console-facing summary of a skill asset row.
type SkillAssetListItem struct {
	ID                 string     `json:"id"`
	Topic              string     `json:"topic"`
	Name               string     `json:"name"`
	DisplayName        string     `json:"display_name"`
	Status             string     `json:"status"`
	Source             string     `json:"source"`
	CreatedBy          string     `json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
	ApprovedBy         string     `json:"approved_by,omitempty"`
	ApprovedAt         *time.Time `json:"approved_at,omitempty"`
	CurrentVersionID   string     `json:"current_version_id,omitempty"`
	VersionLabel       string     `json:"version_label,omitempty"`
	ObservationSummary string     `json:"observation_summary,omitempty"`
}

// SkillAssetDetail includes the current version payload for review.
type SkillAssetDetail struct {
	SkillAssetListItem
	VersionStatus string                 `json:"version_status"`
	Content       map[string]interface{} `json:"content"`
	Checksum      string                 `json:"checksum"`
	VersionNotes  string                 `json:"version_notes"`
}

// ListSkillAssets returns paginated skill assets for super_admin review.
func ListSkillAssets(status, topic string, page, pageSize int) ([]SkillAssetListItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	q := database.DB.Model(&models.SkillAsset{})
	status = strings.TrimSpace(strings.ToLower(status))
	if status != "" {
		q = q.Where("status = ?", status)
	}
	topic = strings.TrimSpace(strings.ToLower(topic))
	if topic != "" {
		q = q.Where("topic = ?", topic)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.SkillAsset
	if err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]SkillAssetListItem, 0, len(rows))
	for _, a := range rows {
		item := skillAssetListItemFromModel(&a)
		if a.CurrentVersionID != nil {
			var ver models.SkillAssetVersion
			if err := database.DB.Where("id = ?", *a.CurrentVersionID).First(&ver).Error; err == nil {
				item.CurrentVersionID = ver.ID.String()
				item.VersionLabel = ver.Version
				item.ObservationSummary = observationSummaryFromContent(ver.Content)
			}
		}
		out = append(out, item)
	}
	return out, total, nil
}

// GetSkillAssetDetail loads one asset and its current version content.
func GetSkillAssetDetail(assetID uuid.UUID) (*SkillAssetDetail, error) {
	var asset models.SkillAsset
	if err := database.DB.Where("id = ?", assetID).First(&asset).Error; err != nil {
		return nil, err
	}
	detail := &SkillAssetDetail{
		SkillAssetListItem: skillAssetListItemFromModel(&asset),
	}
	if asset.CurrentVersionID == nil {
		return detail, nil
	}
	var ver models.SkillAssetVersion
	if err := database.DB.Where("id = ?", *asset.CurrentVersionID).First(&ver).Error; err != nil {
		return detail, nil
	}
	detail.CurrentVersionID = ver.ID.String()
	detail.VersionLabel = ver.Version
	detail.VersionStatus = ver.Status
	detail.Checksum = ver.Checksum
	detail.VersionNotes = ver.Notes
	detail.Content = jsonbToMap(ver.Content)
	detail.ObservationSummary = observationSummaryFromContent(ver.Content)
	return detail, nil
}

// ApproveSkillAsset marks the asset approved, publishes a generated skill pack, and finalizes linked plans.
func ApproveSkillAsset(assetID uuid.UUID, adminID uuid.UUID, adminName, notes string) (*SkillPack, string, error) {
	reg := DefaultSkillRegistry()
	if reg.DataDir() == "" {
		return nil, "", fmt.Errorf("OPSFLEET_AI_SKILL_DATA_DIR 未配置，无法发布技能包")
	}
	var pack *SkillPack
	var path string
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var asset models.SkillAsset
		if err := tx.Where("id = ?", assetID).First(&asset).Error; err != nil {
			return err
		}
		if asset.Status == models.SkillAssetStatusApproved {
			return fmt.Errorf("already_approved")
		}
		if asset.CurrentVersionID == nil {
			return fmt.Errorf("no_version")
		}
		var ver models.SkillAssetVersion
		if err := tx.Where("id = ? AND skill_asset_id = ?", *asset.CurrentVersionID, asset.ID).First(&ver).Error; err != nil {
			return err
		}
		built, err := SkillPackFromDiagnosticAsset(&asset, &ver)
		if err != nil {
			return err
		}
		if !ValidateSkillDraft(built) {
			return fmt.Errorf("invalid_pack")
		}
		now := time.Now().UTC()
		if err := tx.Model(&asset).Updates(map[string]interface{}{
			"status":              models.SkillAssetStatusApproved,
			"approved_by_user_id": adminID,
			"approved_by":         limitSkillText(adminName, 80),
			"approved_at":         &now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&ver).Update("status", models.SkillAssetStatusApproved).Error; err != nil {
			return err
		}
		if notes != "" {
			_ = tx.Model(&ver).Update("notes", limitSkillText(notes, 1000))
		}
		if err := finalizeDiagnosticPlansForAsset(tx, ver.Content); err != nil {
			return err
		}
		pack = built
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	var errPub error
	path, errPub = reg.SaveGenerated(pack)
	if errPub != nil {
		return pack, "", errPub
	}
	return pack, path, nil
}

// RejectSkillAsset deprecates an asset pending review.
func RejectSkillAsset(assetID uuid.UUID, adminName, reason string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var asset models.SkillAsset
		if err := tx.Where("id = ?", assetID).First(&asset).Error; err != nil {
			return err
		}
		if asset.Status == models.SkillAssetStatusApproved {
			return fmt.Errorf("already_approved")
		}
		updates := map[string]interface{}{
			"status":      models.SkillAssetStatusDeprecated,
			"approved_by": limitSkillText(adminName, 80),
		}
		if err := tx.Model(&asset).Updates(updates).Error; err != nil {
			return err
		}
		if asset.CurrentVersionID != nil {
			verUpdates := map[string]interface{}{"status": models.SkillAssetStatusDeprecated}
			if reason != "" {
				verUpdates["notes"] = limitSkillText(reason, 1000)
			}
			_ = tx.Model(&models.SkillAssetVersion{}).Where("id = ?", *asset.CurrentVersionID).Updates(verUpdates)
		}
		return nil
	})
}

// UserDiagnosticSkillOverlay returns a temporary registered skill for unlocked review assets.
func UserDiagnosticSkillOverlay(userID uuid.UUID, topic string) *RegisteredSkill {
	if userID == uuid.Nil {
		return nil
	}
	topic = strings.TrimSpace(strings.ToLower(topic))
	if topic == "" {
		return nil
	}
	var unlocks []models.UserSkillUnlock
	if err := database.DB.Where("user_id = ?", userID).Find(&unlocks).Error; err != nil || len(unlocks) == 0 {
		return nil
	}
	assetIDs := make([]uuid.UUID, 0, len(unlocks))
	for _, u := range unlocks {
		assetIDs = append(assetIDs, u.SkillAssetID)
	}
	var assets []models.SkillAsset
	if err := database.DB.Where("id IN ? AND topic = ? AND status = ?", assetIDs, topic, models.SkillAssetStatusReview).Find(&assets).Error; err != nil || len(assets) == 0 {
		return nil
	}
	asset := assets[0]
	if asset.CurrentVersionID == nil {
		return nil
	}
	var ver models.SkillAssetVersion
	if err := database.DB.Where("id = ?", *asset.CurrentVersionID).First(&ver).Error; err != nil {
		return nil
	}
	pack, err := SkillPackFromDiagnosticAsset(&asset, &ver)
	if err != nil || pack == nil {
		return nil
	}
	return &RegisteredSkill{Pack: *pack, Source: SkillSourceGenerated, Version: "unlock:" + ver.Version}
}

// MergeRegisteredSkills combines registry skill with diagnostic overlay extra guidance.
func MergeRegisteredSkills(base, overlay *RegisteredSkill) *RegisteredSkill {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}
	merged := *base
	pack := base.Pack
	if strings.TrimSpace(overlay.Pack.ExtraGuidance) != "" {
		pack.ExtraGuidance = strings.TrimSpace(pack.ExtraGuidance)
		if pack.ExtraGuidance != "" {
			pack.ExtraGuidance += "\n\n"
		}
		pack.ExtraGuidance += overlay.Pack.ExtraGuidance
	}
	merged.Pack = pack
	return &merged
}

// SkillPackFromDiagnosticAsset builds a publishable skill pack from a diagnostic asset version.
func SkillPackFromDiagnosticAsset(asset *models.SkillAsset, ver *models.SkillAssetVersion) (*SkillPack, error) {
	if asset == nil || ver == nil {
		return nil, fmt.Errorf("missing asset or version")
	}
	content := jsonbToMap(ver.Content)
	topic := strings.TrimSpace(strings.ToLower(asset.Topic))
	if topic == "" {
		topic = strings.TrimSpace(strings.ToLower(fmt.Sprint(content["topic"])))
	}
	if topic == "" {
		return nil, fmt.Errorf("topic missing")
	}
	name := diagnosticPackName(topic)
	display := strings.TrimSpace(asset.DisplayName)
	if display == "" {
		display = "诊断沉淀: " + strings.ToUpper(topic)
	}
	steps := analysisStepsFromDiagnosticContent(content)
	extra := buildDiagnosticExtraGuidance(content)
	return &SkillPack{
		Name:          name,
		DisplayName:   display,
		Topics:        []string{topic},
		MatchKeywords: []string{topic, "diagnose", "readonly", "kubectl"},
		Input:         []string{"namespace", "pod", "issue"},
		AnalysisSteps: steps,
		OutputFormat:  []string{"root_cause", "solution", "verification_commands"},
		ExtraGuidance: extra,
	}, nil
}

func diagnosticPackName(topic string) string {
	t := sanitizeSkillToken(topic)
	if t == "" {
		t = "unknown"
	}
	return t + "_diagnostic_readonly"
}

func analysisStepsFromDiagnosticContent(content map[string]interface{}) []string {
	out := make([]string, 0, 8)
	if raw, ok := content["steps"]; ok {
		switch steps := raw.(type) {
		case []interface{}:
			for _, item := range steps {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				title := strings.TrimSpace(fmt.Sprint(m["title"]))
				if title != "" {
					out = append(out, title)
				}
			}
		}
	}
	if len(out) < 2 {
		out = []string{
			"阅读诊断任务单采集的只读 kubectl 证据",
			"归纳根因并给出可验证命令与缓解建议",
		}
	}
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func buildDiagnosticExtraGuidance(content map[string]interface{}) string {
	var b strings.Builder
	if s := strings.TrimSpace(fmt.Sprint(content["observation_summary"])); s != "" && s != "<nil>" {
		b.WriteString("【诊断观察摘要】\n")
		b.WriteString(s)
		b.WriteString("\n")
	}
	if obs, ok := content["observations"].(map[string]interface{}); ok && len(obs) > 0 {
		b.WriteString("\n【关键证据键】\n")
		n := 0
		for k := range obs {
			b.WriteString("- ")
			b.WriteString(k)
			b.WriteString("\n")
			n++
			if n >= 24 {
				break
			}
		}
	}
	b.WriteString("\n来源：CLI 只读诊断任务单；发布前已由超级管理员审核。")
	return strings.TrimSpace(b.String())
}

func finalizeDiagnosticPlansForAsset(tx *gorm.DB, content models.JSONB) error {
	m := jsonbToMap(content)
	planID := strings.TrimSpace(fmt.Sprint(m["source_plan_id"]))
	if planID == "" || planID == "<nil>" {
		return nil
	}
	id, err := uuid.Parse(planID)
	if err != nil {
		return nil
	}
	return tx.Model(&models.DiagnosticPlan{}).
		Where("id = ? AND status = ?", id, models.DiagnosticPlanStatusObserved).
		Update("status", models.DiagnosticPlanStatusFinalized).Error
}

func skillAssetListItemFromModel(a *models.SkillAsset) SkillAssetListItem {
	item := SkillAssetListItem{
		ID:          a.ID.String(),
		Topic:       a.Topic,
		Name:        a.Name,
		DisplayName: a.DisplayName,
		Status:      a.Status,
		Source:      a.Source,
		CreatedBy:   a.CreatedBy,
		CreatedAt:   a.CreatedAt,
		ApprovedBy:  a.ApprovedBy,
		ApprovedAt:  a.ApprovedAt,
	}
	if a.CurrentVersionID != nil {
		item.CurrentVersionID = a.CurrentVersionID.String()
	}
	return item
}

func observationSummaryFromContent(content models.JSONB) string {
	m := jsonbToMap(content)
	return strings.TrimSpace(fmt.Sprint(m["observation_summary"]))
}

func jsonbToMap(j models.JSONB) map[string]interface{} {
	if len(j) == 0 {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(j, &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

func limitSkillText(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}
