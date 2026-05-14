package middleware

import (
	"net/http"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	CapabilityActionView     = "view"
	CapabilityActionPreview  = "preview"
	CapabilityActionExecute  = "execute"
	CapabilityActionReport   = "report"
	CapabilityActionDownload = "download"
	CapabilityActionAICall   = "ai_call"
)

func capabilitySetting(featureKey string) (models.FeatureBillingSetting, error) {
	setting := models.FeatureBillingSetting{
		FeatureKey:       featureKey,
		PackKey:          models.DefaultPackKeyForFeature(featureKey),
		VisibleEnabled:   true,
		ExecutionEnabled: true,
		BillingEnabled:   false,
		Description:      models.DefaultFeatureDescription(featureKey),
	}
	var row models.FeatureBillingSetting
	if err := database.DB.Where("feature_key = ?", featureKey).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return setting, nil
		}
		return setting, err
	}
	if row.PackKey == "" {
		row.PackKey = setting.PackKey
	}
	if row.Description == "" {
		row.Description = setting.Description
	}
	// Old rows added before these columns existed are healed at seed time; this
	// fallback keeps capability checks usable even if seed has not run yet.
	if !row.VisibleEnabled && row.UpdatedAt.IsZero() {
		row.VisibleEnabled = true
	}
	if !row.ExecutionEnabled && row.UpdatedAt.IsZero() {
		row.ExecutionEnabled = true
	}
	return row, nil
}

func entitlementSourceForUser(uid uuid.UUID, keys ...string) (string, bool) {
	clean := make([]string, 0, len(keys))
	seen := map[string]struct{}{}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, key)
	}
	if len(clean) == 0 {
		return "", false
	}
	var ent models.Entitlement
	err := database.DB.Where("user_id = ? AND feature_key IN ?", uid, clean).
		Where("valid_until IS NULL OR valid_until > ?", time.Now().UTC()).
		Order("valid_until DESC NULLS FIRST").
		First(&ent).Error
	if err != nil {
		return "", false
	}
	return ent.Source, true
}

func actionRequiresExecution(action string) bool {
	return action != CapabilityActionView && action != CapabilityActionPreview
}

func CheckCapability(uid uuid.UUID, role, featureKey, action string) (bool, gin.H) {
	if !models.IsKnownFeatureKey(featureKey) {
		return false, gin.H{"code": 403, "msg": "未知功能权益", "feature_key": featureKey}
	}

	setting, err := capabilitySetting(featureKey)
	if err != nil {
		return false, gin.H{"code": 500, "msg": "计费配置读取失败", "feature_key": featureKey}
	}
	packKey := setting.PackKey
	if packKey == "" {
		packKey = models.DefaultPackKeyForFeature(featureKey)
	}

	if !setting.VisibleEnabled && !actionRequiresExecution(action) {
		return false, gin.H{
			"code":               403,
			"msg":                "当前功能暂未开放",
			"biz":                "FEATURE_HIDDEN_" + featureKey,
			"feature_key":        featureKey,
			"pack_key":           packKey,
			"checkout_available": false,
		}
	}
	if actionRequiresExecution(action) && !setting.ExecutionEnabled {
		return false, gin.H{
			"code":               403,
			"msg":                "当前功能执行开关已关闭",
			"biz":                "FEATURE_DISABLED_" + featureKey,
			"feature_key":        featureKey,
			"pack_key":           packKey,
			"checkout_available": false,
		}
	}

	if !actionRequiresExecution(action) {
		return true, gin.H{
			"feature_key":        featureKey,
			"pack_key":           packKey,
			"entitlement_source": "",
			"billing_required":   false,
			"checkout_available": false,
		}
	}
	if models.IsSuperAdminRole(role) || !setting.BillingEnabled {
		source := ""
		if models.IsSuperAdminRole(role) {
			source = "super_admin"
		}
		return true, gin.H{
			"feature_key":        featureKey,
			"pack_key":           packKey,
			"entitlement_source": source,
			"billing_required":   setting.BillingEnabled,
		}
	}
	if uid == uuid.Nil {
		return false, gin.H{"code": 401, "msg": "未授权"}
	}
	source, ok := entitlementSourceForUser(uid, packKey, featureKey)
	if ok {
		return true, gin.H{
			"feature_key":        featureKey,
			"pack_key":           packKey,
			"entitlement_source": source,
			"billing_required":   true,
		}
	}
	return false, gin.H{
		"code":               403,
		"msg":                "当前功能需订阅后使用",
		"biz":                "PAYWALL_" + packKey,
		"feature_key":        featureKey,
		"pack_key":           packKey,
		"reason":             "missing_entitlement",
		"checkout_available": true,
	}
}

func RequireCapability(featureKey, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, _ := c.Get("role")
		role, _ := roleVal.(string)
		uid := models.UserIDFromContext(c.MustGet("userID"))
		allowed, payload := CheckCapability(uid, role, featureKey, action)
		if !allowed {
			code, _ := payload["code"].(int)
			if code == 0 {
				code = http.StatusForbidden
			}
			c.JSON(code, payload)
			c.Abort()
			return
		}
		c.Set("feature_key", payload["feature_key"])
		c.Set("pack_key", payload["pack_key"])
		c.Set("entitlement_source", payload["entitlement_source"])
		c.Next()
	}
}

// RequireEntitlementOrSuperAdmin enforces feature billing for all users except
// the system-level super admin role. Kept for backward compatibility.
func RequireEntitlementOrSuperAdmin(featureKey string) gin.HandlerFunc {
	return RequireCapability(featureKey, CapabilityActionExecute)
}
