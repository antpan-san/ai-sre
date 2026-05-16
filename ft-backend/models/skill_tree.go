package models

import "time"

const (
	SkillTreeVersionStatusDraft    = "draft"
	SkillTreeVersionStatusActive   = "active"
	SkillTreeVersionStatusArchived = "archived"

	SkillTreeNodeStatusActive   = "active"
	SkillTreeNodeStatusDisabled = "disabled"
)

// SkillTreeVersion is an immutable-ish revision of the capability tree.
type SkillTreeVersion struct {
	BaseModel
	TreeRev     string     `gorm:"size:80;not null;uniqueIndex" json:"tree_rev"`
	Status      string     `gorm:"size:32;not null;default:'draft';index" json:"status"`
	Title       string     `gorm:"size:200" json:"title"`
	Notes       string     `gorm:"size:2000" json:"notes"`
	PublishedBy string     `gorm:"size:80" json:"published_by"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

func (SkillTreeVersion) TableName() string {
	return "skill_tree_versions"
}

// SkillTreeNodeRecord is a persisted tree node (no physical delete; use status=disabled).
type SkillTreeNodeRecord struct {
	BaseModel
	TreeRev       string `gorm:"size:80;not null;uniqueIndex:idx_skill_tree_node_rev_path;index" json:"tree_rev"`
	Path          string `gorm:"size:300;not null;uniqueIndex:idx_skill_tree_node_rev_path" json:"path"`
	ParentPath    string `gorm:"size:300;index" json:"parent_path"`
	NodeType      string `gorm:"size:32;not null;index" json:"node_type"`
	Title         string `gorm:"size:200;not null" json:"title"`
	Description   string `gorm:"size:2000" json:"description"`
	Topic         string `gorm:"size:80;index" json:"topic"`
	SkillKey      string `gorm:"size:160;index" json:"skill_key"`
	ProblemKey    string `gorm:"size:120;index" json:"problem_key"`
	CapabilityKey string `gorm:"size:160;index" json:"capability_key"`
	PackKey       string `gorm:"size:80;index" json:"pack_key"`
	FeatureKey    string `gorm:"size:80;index" json:"feature_key"`
	ExecutionMode string `gorm:"size:80" json:"execution_mode"`
	CLIVisible    bool   `gorm:"not null;default:true" json:"cli_visible"`
	Status        string `gorm:"size:32;not null;default:'active';index" json:"status"`
	SortOrder     int    `gorm:"not null;default:0;index" json:"sort_order"`
	Metadata      JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
}

func (SkillTreeNodeRecord) TableName() string {
	return "skill_tree_nodes"
}
