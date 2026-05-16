package services

import (
	"fmt"
	"strings"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SkillApproveDiff previews registry merge impact before publishing.
type SkillApproveDiff struct {
	Topic              string                 `json:"topic"`
	GeneratedPackName  string                 `json:"generated_pack_name"`
	RegistryPackName   string                 `json:"registry_pack_name,omitempty"`
	RegistrySource     string                 `json:"registry_source,omitempty"`
	MergePreview       bool                   `json:"merge_preview"`
	GeneratedSummary   map[string]interface{} `json:"generated_summary"`
	RegistrySummary    map[string]interface{} `json:"registry_summary,omitempty"`
	MergedSummary      map[string]interface{} `json:"merged_summary,omitempty"`
	FieldsChanged      []string               `json:"fields_changed,omitempty"`
}

// BuildSkillApproveDiff compares a pending asset against the on-disk registry pack.
func BuildSkillApproveDiff(assetID uuid.UUID, mergeWithRegistry bool) (*SkillApproveDiff, error) {
	var asset models.SkillAsset
	if err := database.DB.Where("id = ?", assetID).First(&asset).Error; err != nil {
		return nil, err
	}
	if asset.CurrentVersionID == nil {
		return nil, fmt.Errorf("no_version")
	}
	var ver models.SkillAssetVersion
	if err := database.DB.Where("id = ? AND skill_asset_id = ?", *asset.CurrentVersionID, asset.ID).First(&ver).Error; err != nil {
		return nil, err
	}
	generated, err := SkillPackFromDiagnosticAsset(&asset, &ver)
	if err != nil {
		return nil, err
	}
	topic := strings.TrimSpace(generated.Topics[0])
	out := &SkillApproveDiff{
		Topic:             topic,
		GeneratedPackName: generated.Name,
		MergePreview:      mergeWithRegistry,
		GeneratedSummary:  summarizeSkillPack(generated),
	}
	reg := DefaultSkillRegistry()
	if reg == nil || reg.DataDir() == "" {
		return out, nil
	}
	base := reg.Match(topic, nil)
	if base == nil {
		return out, nil
	}
	out.RegistryPackName = base.Pack.Name
	out.RegistrySource = string(base.Source)
	out.RegistrySummary = summarizeSkillPack(&base.Pack)
	if mergeWithRegistry {
		merged, _ := MergeSkillPackWithRegistry(reg, generated)
		if merged != nil {
			out.MergedSummary = summarizeSkillPack(merged)
			out.FieldsChanged = diffSkillPackFields(&base.Pack, merged)
		}
	} else {
		out.FieldsChanged = diffSkillPackFields(&base.Pack, generated)
	}
	return out, nil
}

func summarizeSkillPack(p *SkillPack) map[string]interface{} {
	if p == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"name":                 p.Name,
		"display_name":         p.DisplayName,
		"topics":               p.Topics,
		"analysis_steps_count": len(p.AnalysisSteps),
		"match_keywords_count": len(p.MatchKeywords),
		"extra_guidance_bytes": len(strings.TrimSpace(p.ExtraGuidance)),
	}
}

func diffSkillPackFields(base, next *SkillPack) []string {
	if base == nil || next == nil {
		return nil
	}
	var changed []string
	if base.Name != next.Name {
		changed = append(changed, "name")
	}
	if base.DisplayName != next.DisplayName {
		changed = append(changed, "display_name")
	}
	if len(base.AnalysisSteps) != len(next.AnalysisSteps) {
		changed = append(changed, "analysis_steps")
	}
	if len(base.MatchKeywords) != len(next.MatchKeywords) {
		changed = append(changed, "match_keywords")
	}
	if strings.TrimSpace(base.ExtraGuidance) != strings.TrimSpace(next.ExtraGuidance) {
		changed = append(changed, "extra_guidance")
	}
	return changed
}

// ListSkillAssetReviews returns audit rows for one asset.
func ListSkillAssetReviews(assetID uuid.UUID, limit int) ([]models.SkillAssetReview, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []models.SkillAssetReview
	err := database.DB.Where("skill_asset_id = ?", assetID).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// DeprecateSkillAsset marks an approved asset deprecated without deleting registry files.
func DeprecateSkillAsset(assetID uuid.UUID, adminID uuid.UUID, adminName, reason string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var asset models.SkillAsset
		if err := tx.Where("id = ?", assetID).First(&asset).Error; err != nil {
			return err
		}
		if asset.Status != models.SkillAssetStatusApproved {
			return fmt.Errorf("not_approved")
		}
		updates := map[string]interface{}{
			"status":            models.SkillAssetStatusDeprecated,
			"deprecated_reason": limitSkillText(reason, 2000),
		}
		if err := tx.Model(&asset).Updates(updates).Error; err != nil {
			return err
		}
		if asset.CurrentVersionID != nil {
			_ = tx.Model(&models.SkillAssetVersion{}).
				Where("id = ?", *asset.CurrentVersionID).
				Update("status", models.SkillAssetStatusDeprecated)
		}
		return insertSkillAssetReview(tx, asset.ID, models.SkillAssetReviewActionDeprecate, adminID, adminName, reason, "", false, "", nil)
	})
}

func insertSkillAssetReview(tx *gorm.DB, assetID uuid.UUID, action string, actorID uuid.UUID, actorName, notes, publishMode string, merged bool, path string, diff map[string]interface{}) error {
	if diff == nil {
		diff = map[string]interface{}{}
	}
	row := models.SkillAssetReview{
		SkillAssetID:      assetID,
		Action:            action,
		ActorUserID:       &actorID,
		ActorName:         limitSkillText(actorName, 80),
		Notes:             limitSkillText(notes, 2000),
		PublishMode:       publishMode,
		MergedWithBuiltin: merged,
		PublishedPackPath: limitSkillText(path, 500),
		DiffSummary:       models.NewJSONBFromMap(diff),
	}
	if actorID == uuid.Nil {
		row.ActorUserID = nil
	}
	return tx.Create(&row).Error
}

func inferRiskLevel(content map[string]interface{}) string {
	if content == nil {
		return "low"
	}
	bytes := 0
	if obs, ok := content["observations"].(map[string]interface{}); ok {
		for k, v := range obs {
			bytes += len(k) + len(fmt.Sprint(v))
		}
	}
	if s := strings.TrimSpace(fmt.Sprint(content["observation_summary"])); s != "" && s != "<nil>" {
		bytes += len(s)
	}
	switch {
	case bytes > 256*1024:
		return "high"
	case bytes > 64*1024:
		return "medium"
	default:
		return "low"
	}
}
