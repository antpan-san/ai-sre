package models

import (
	"time"

	"github.com/google/uuid"
)

// 功能分级：各 feature_key 对应一类管理端能力；计费开启时非 super_admin 需具备对应 entitlement。
const (
	FeatureKeyAdvanced   = "feature.advanced"    // 备份、性能分析、报告
	FeatureKeyK8sOps    = "feature.k8s_ops"    // K8s 部署、集群、bundle、relay
	FeatureKeyServiceOps = "feature.service_ops" // 服务部署、Linux 服务、控制台 service-deploy
	FeatureKeyInfraOps  = "feature.infra_ops"  // 代理、监控告警、初始化工具
)

var knownFeatureKeys = map[string]struct{}{
	FeatureKeyAdvanced:   {},
	FeatureKeyK8sOps:     {},
	FeatureKeyServiceOps: {},
	FeatureKeyInfraOps:   {},
}

// IsKnownFeatureKey 用于管理端配置与人工授权校验。
func IsKnownFeatureKey(featureKey string) bool {
	_, ok := knownFeatureKeys[featureKey]
	return ok
}

// AllFeatureKeysStable 返回稳定排序的功能键列表（用于 seed、测试）。
func AllFeatureKeysStable() []string {
	return []string{
		FeatureKeyInfraOps,
		FeatureKeyK8sOps,
		FeatureKeyServiceOps,
		FeatureKeyAdvanced,
	}
}

// FeatureBillingSetting toggles whether non-super-admin users must have entitlement for a feature_key.
type FeatureBillingSetting struct {
	FeatureKey     string    `gorm:"primaryKey;size:80" json:"feature_key"`
	BillingEnabled bool      `gorm:"not null;default:false" json:"billing_enabled"`
	Description    string    `gorm:"size:512" json:"description"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Subscription is a per-user Stripe (or other PSP) subscription row.
type Subscription struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID               uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Status               string     `gorm:"size:32;not null" json:"status"`
	PlanID               string     `gorm:"size:64" json:"plan_id"`
	StripeCustomerID     string     `gorm:"size:128;index" json:"stripe_customer_id"`
	StripeSubscriptionID string     `gorm:"size:128;index" json:"stripe_subscription_id"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Entitlement grants a user access to a feature_key until ValidUntil (nil = no expiry).
type Entitlement struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_ent_user_feature" json:"user_id"`
	FeatureKey string     `gorm:"size:80;not null;index:idx_ent_user_feature" json:"feature_key"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	Source     string     `gorm:"size:64" json:"source"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
