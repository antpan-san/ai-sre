package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	SkillAssetStatusDraft      = "draft"
	SkillAssetStatusReview     = "review"
	SkillAssetStatusApproved   = "approved"
	SkillAssetStatusDeprecated = "deprecated"

	DiagnosticPlanStatusPending   = "pending"
	DiagnosticPlanStatusObserved  = "observed"
	DiagnosticPlanStatusFinalized = "finalized"
	DiagnosticPlanStatusExpired   = "expired"
)

type SkillAsset struct {
	BaseModel
	Topic            string     `gorm:"size:80;not null;index" json:"topic"`
	Name             string     `gorm:"size:120;not null;uniqueIndex" json:"name"`
	DisplayName      string     `gorm:"size:200" json:"display_name"`
	Status           string     `gorm:"size:32;not null;default:'draft';index" json:"status"`
	Source           string     `gorm:"size:64;not null;default:'diagnosis'" json:"source"`
	CreatedByUserID  *uuid.UUID `gorm:"type:uuid;index" json:"created_by_user_id,omitempty"`
	CreatedBy        string     `gorm:"size:80" json:"created_by"`
	ApprovedByUserID *uuid.UUID `gorm:"type:uuid;index" json:"approved_by_user_id,omitempty"`
	ApprovedBy       string     `gorm:"size:80" json:"approved_by"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	CurrentVersionID *uuid.UUID `gorm:"type:uuid;index" json:"current_version_id,omitempty"`
	QualityLabels    JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"quality_labels"`
}

func (SkillAsset) TableName() string {
	return "skill_assets"
}

type SkillAssetVersion struct {
	BaseModel
	SkillAssetID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_skill_asset_version;index" json:"skill_asset_id"`
	Version      string    `gorm:"size:64;not null;uniqueIndex:idx_skill_asset_version" json:"version"`
	Status       string    `gorm:"size:32;not null;default:'draft';index" json:"status"`
	Content      JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"content"`
	Checksum     string    `gorm:"size:64;not null;index" json:"checksum"`
	Notes        string    `gorm:"size:1000" json:"notes"`
}

func (SkillAssetVersion) TableName() string {
	return "skill_asset_versions"
}

type UserSkillUnlock struct {
	BaseModel
	UserID              uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_user_skill_unlock" json:"user_id"`
	SkillAssetID        uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_user_skill_unlock" json:"skill_asset_id"`
	SkillAssetVersionID *uuid.UUID `gorm:"type:uuid;index" json:"skill_asset_version_id,omitempty"`
	Source              string     `gorm:"size:64;not null;default:'diagnosis_unlock'" json:"source"`
	ValidUntil          *time.Time `json:"valid_until,omitempty"`
}

func (UserSkillUnlock) TableName() string {
	return "user_skill_unlocks"
}

type DiagnosticPlan struct {
	BaseModel
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Username        string     `gorm:"size:80;not null;index" json:"username"`
	CLIBindingID    *uuid.UUID `gorm:"type:uuid;index" json:"cli_binding_id,omitempty"`
	FingerprintHash string     `gorm:"size:64;not null;index" json:"fingerprint_hash"`
	Topic           string     `gorm:"size:80;not null;index" json:"topic"`
	Command         string     `gorm:"size:2000" json:"command"`
	RequestID       string     `gorm:"size:80;index" json:"request_id"`
	Status          string     `gorm:"size:32;not null;default:'pending';index" json:"status"`
	PlanTokenHash   string     `gorm:"size:64;not null" json:"-"`
	ExpiresAt       time.Time  `gorm:"not null;index" json:"expires_at"`
	Steps           JSONB      `gorm:"type:jsonb;not null;default:'[]'" json:"steps"`
	Observations    JSONB      `gorm:"type:jsonb;not null;default:'{}'" json:"observations"`
	Summary         string     `gorm:"size:1200" json:"summary"`
}

func (DiagnosticPlan) TableName() string {
	return "diagnostic_plans"
}
