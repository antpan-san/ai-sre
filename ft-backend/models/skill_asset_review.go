package models

import (
	"github.com/google/uuid"
)

const (
	SkillAssetReviewActionApprove   = "approve"
	SkillAssetReviewActionReject    = "reject"
	SkillAssetReviewActionDeprecate = "deprecate"

	SkillAssetPublishModeMerge      = "merge"
	SkillAssetPublishModeStandalone = "standalone"
)

// SkillAssetReview is an audit trail row for approve / reject / deprecate actions.
type SkillAssetReview struct {
	BaseModel
	SkillAssetID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"skill_asset_id"`
	Action            string     `gorm:"size:32;not null;index" json:"action"`
	ActorUserID       *uuid.UUID `gorm:"type:uuid;index" json:"actor_user_id,omitempty"`
	ActorName         string     `gorm:"size:80" json:"actor_name"`
	Notes             string     `gorm:"size:2000" json:"notes,omitempty"`
	PublishMode       string     `gorm:"size:32" json:"publish_mode,omitempty"`
	MergedWithBuiltin bool       `gorm:"not null;default:false" json:"merged_with_builtin"`
	PublishedPackPath string     `gorm:"size:500" json:"published_pack_path,omitempty"`
	DiffSummary       JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"diff_summary"`
}

func (SkillAssetReview) TableName() string {
	return "skill_asset_reviews"
}
