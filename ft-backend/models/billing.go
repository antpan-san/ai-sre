package models

import (
	"time"

	"github.com/google/uuid"
)

// 功能分级：feature_key 是页面/接口能力，pack_key 是订阅售卖单元。
// 计费开启时非 super_admin 需具备 feature 对应 pack 的 entitlement。
const (
	FeatureKeyK8sOps            = "feature.k8s_ops"            // legacy: K8s 部署、bundle、relay
	FeatureKeyServiceOps        = "feature.service_ops"        // legacy: 服务交付、Linux 服务
	FeatureKeyInfraOps          = "feature.infra_ops"          // legacy: 代理、监控、初始化
	FeatureKeyAdvanced          = "feature.advanced"           // legacy: 备份、性能分析、报告
	FeatureKeyK8sDelivery       = "feature.k8s_delivery"       // K8s 在线/离线交付
	FeatureKeyNodeOps           = "feature.node_ops"           // 节点初始化、Shell、文件分发、Linux 服务
	FeatureKeyMonitoring        = "feature.monitoring"         // exporter/监控告警安装配置
	FeatureKeyBackupPerformance = "feature.backup_performance" // 备份、性能分析、报告
	FeatureKeyAIDiagnosis       = "feature.ai_diagnosis"       // AI 诊断、问答、Runbook

	PackKeyK8sDelivery       = "pack.k8s_delivery"
	PackKeyNodeOps           = "pack.node_ops"
	PackKeyMonitoring        = "pack.monitoring"
	PackKeyBackupPerformance = "pack.backup_performance"

	SkillPackK8s           = "skillpack.k8s"
	SkillPackKafka         = "skillpack.kafka"
	SkillPackRedis         = "skillpack.redis"
	SkillPackNginx         = "skillpack.nginx"
	SkillPackMySQL         = "skillpack.mysql"
	SkillPackElasticsearch = "skillpack.elasticsearch"
)

var knownFeatureKeys = map[string]struct{}{
	FeatureKeyAdvanced:          {},
	FeatureKeyK8sOps:            {},
	FeatureKeyServiceOps:        {},
	FeatureKeyInfraOps:          {},
	FeatureKeyK8sDelivery:       {},
	FeatureKeyNodeOps:           {},
	FeatureKeyMonitoring:        {},
	FeatureKeyBackupPerformance: {},
	FeatureKeyAIDiagnosis:       {},
}

var knownPackKeys = map[string]struct{}{
	PackKeyK8sDelivery:       {},
	PackKeyNodeOps:           {},
	PackKeyMonitoring:        {},
	PackKeyBackupPerformance: {},
	SkillPackK8s:             {},
	SkillPackKafka:           {},
	SkillPackRedis:           {},
	SkillPackNginx:           {},
	SkillPackMySQL:           {},
	SkillPackElasticsearch:   {},
}

// IsKnownFeatureKey 用于管理端配置与人工授权校验。
func IsKnownFeatureKey(featureKey string) bool {
	_, ok := knownFeatureKeys[featureKey]
	return ok
}

func IsKnownPackKey(packKey string) bool {
	_, ok := knownPackKeys[packKey]
	return ok
}

func IsKnownEntitlementKey(key string) bool {
	return IsKnownPackKey(key) || IsKnownFeatureKey(key)
}

// AllFeatureKeysStable 返回稳定排序的功能键列表（用于 seed、测试）。
func AllFeatureKeysStable() []string {
	return []string{
		FeatureKeyK8sDelivery,
		FeatureKeyNodeOps,
		FeatureKeyMonitoring,
		FeatureKeyBackupPerformance,
		FeatureKeyAIDiagnosis,
		FeatureKeyK8sOps,
		FeatureKeyServiceOps,
		FeatureKeyInfraOps,
		FeatureKeyAdvanced,
	}
}

func AllPackKeysStable() []string {
	return []string{
		PackKeyK8sDelivery,
		PackKeyNodeOps,
		PackKeyMonitoring,
		PackKeyBackupPerformance,
		SkillPackK8s,
		SkillPackKafka,
		SkillPackRedis,
		SkillPackNginx,
		SkillPackMySQL,
		SkillPackElasticsearch,
	}
}

func DefaultPackKeyForFeature(featureKey string) string {
	switch featureKey {
	case FeatureKeyK8sDelivery, FeatureKeyK8sOps:
		return PackKeyK8sDelivery
	case FeatureKeyNodeOps, FeatureKeyServiceOps, FeatureKeyInfraOps:
		return PackKeyNodeOps
	case FeatureKeyMonitoring:
		return PackKeyMonitoring
	case FeatureKeyBackupPerformance, FeatureKeyAdvanced:
		return PackKeyBackupPerformance
	case FeatureKeyAIDiagnosis:
		return SkillPackK8s
	default:
		return featureKey
	}
}

func DefaultFeatureDescription(featureKey string) string {
	switch featureKey {
	case FeatureKeyK8sDelivery:
		return "K8s 交付包（在线部署、离线包、installRef、集群清理、制品分发）"
	case FeatureKeyNodeOps:
		return "节点运维包（初始化、时间同步、安全加固、磁盘优化、Shell、文件分发、Linux 服务）"
	case FeatureKeyMonitoring:
		return "监控包（Prometheus 与各类 exporter 安装、配置、下发）"
	case FeatureKeyBackupPerformance:
		return "备份与性能包（备份恢复、性能分析、真实报告生成）"
	case FeatureKeyAIDiagnosis:
		return "AI 诊断技能包（未购买时每日免费 5 次）"
	case FeatureKeyK8sOps:
		return "K8s 交付（兼容旧功能键）"
	case FeatureKeyServiceOps:
		return "服务/节点运维（兼容旧功能键）"
	case FeatureKeyInfraOps:
		return "基础设施运维（兼容旧功能键）"
	case FeatureKeyAdvanced:
		return "高级功能（兼容旧功能键）"
	default:
		return featureKey
	}
}

// FeatureBillingSetting controls feature visibility, execution and billing.
type FeatureBillingSetting struct {
	FeatureKey       string    `gorm:"primaryKey;size:80" json:"feature_key"`
	PackKey          string    `gorm:"size:80;index" json:"pack_key"`
	VisibleEnabled   bool      `gorm:"not null;default:true" json:"visible_enabled"`
	ExecutionEnabled bool      `gorm:"not null;default:true" json:"execution_enabled"`
	BillingEnabled   bool      `gorm:"not null;default:false" json:"billing_enabled"`
	StripePriceID    string    `gorm:"size:128" json:"stripe_price_id"`
	Description      string    `gorm:"size:512" json:"description"`
	UpdatedAt        time.Time `json:"updated_at"`
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

// Entitlement grants a user access to a pack_key or legacy feature_key until ValidUntil (nil = no expiry).
type Entitlement struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_ent_user_feature" json:"user_id"`
	FeatureKey string     `gorm:"size:80;not null;index:idx_ent_user_feature" json:"feature_key"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	Source     string     `gorm:"size:64" json:"source"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AIUsage stores successful free AI calls for account/IP subjects.
type AIUsage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Subject   string    `gorm:"size:120;not null;uniqueIndex:idx_ai_usage_subject_day_pack" json:"subject"`
	PackKey   string    `gorm:"size:80;not null;uniqueIndex:idx_ai_usage_subject_day_pack" json:"pack_key"`
	UsageDate string    `gorm:"size:10;not null;uniqueIndex:idx_ai_usage_subject_day_pack" json:"usage_date"`
	Count     int       `gorm:"not null;default:0" json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
