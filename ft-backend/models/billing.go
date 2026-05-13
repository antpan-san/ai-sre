package models

import (
	"time"

	"github.com/google/uuid"
)

// FeatureKeyAdvanced 与「高级功能」API 粗粒度绑定（备份 + 性能）。
const FeatureKeyAdvanced = "feature.advanced"

// FeatureBillingSetting toggles whether non-admin users must have entitlement for a feature_key.
type FeatureBillingSetting struct {
	FeatureKey     string `gorm:"primaryKey;size:80" json:"feature_key"`
	BillingEnabled bool   `gorm:"not null;default:false" json:"billing_enabled"`
	Description    string `gorm:"size:512" json:"description"`
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
