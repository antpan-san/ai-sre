package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/middleware"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	stripecustomer "github.com/stripe/stripe-go/v81/customer"
	stripeSub "github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/webhook"
	"gorm.io/gorm"
)

func resolvedBillingPackages(cfg *config.Config) []config.BillingPackage {
	defaults := defaultBillingPackages()
	if cfg == nil {
		return defaults
	}
	priceByPack := map[string]string{}
	if len(cfg.Billing.Packages) > 0 {
		for _, p := range cfg.Billing.Packages {
			if strings.TrimSpace(p.ID) != "" && strings.TrimSpace(p.StripePriceID) != "" {
				priceByPack[p.ID] = p.StripePriceID
			}
		}
	}
	if pid := strings.TrimSpace(cfg.Billing.StripePriceIDPro); pid != "" {
		priceByPack[models.PackKeyBackupPerformance] = pid
	}
	for i := range defaults {
		if pid := priceByPack[defaults[i].ID]; pid != "" {
			defaults[i].StripePriceID = pid
		}
	}
	return defaults
}

func defaultBillingPackages() []config.BillingPackage {
	return []config.BillingPackage{
		{ID: models.PackKeyK8sDelivery, DisplayName: "K8s 交付包", FeatureKeys: []string{models.FeatureKeyK8sDelivery, models.FeatureKeyK8sOps}},
		{ID: models.PackKeyNodeOps, DisplayName: "节点运维包", FeatureKeys: []string{models.FeatureKeyNodeOps, models.FeatureKeyServiceOps, models.FeatureKeyInfraOps}},
		{ID: models.PackKeyMonitoring, DisplayName: "监控包", FeatureKeys: []string{models.FeatureKeyMonitoring}},
		{ID: models.PackKeyBackupPerformance, DisplayName: "备份与性能包", FeatureKeys: []string{models.FeatureKeyBackupPerformance, models.FeatureKeyAdvanced}},
		{ID: models.PackKeyRuntimeObserve, DisplayName: "进程观测包", FeatureKeys: []string{models.FeatureKeyRuntimeObserve}},
		{ID: models.SkillPackK8s, DisplayName: "K8s AI 技能包", FeatureKeys: []string{models.FeatureKeyAIDiagnosis}},
		{ID: models.SkillPackKafka, DisplayName: "Kafka AI 技能包", FeatureKeys: []string{models.FeatureKeyAIDiagnosis}},
		{ID: models.SkillPackRedis, DisplayName: "Redis AI 技能包", FeatureKeys: []string{models.FeatureKeyAIDiagnosis}},
		{ID: models.SkillPackNginx, DisplayName: "Nginx AI 技能包", FeatureKeys: []string{models.FeatureKeyAIDiagnosis}},
		{ID: models.SkillPackMySQL, DisplayName: "MySQL AI 技能包", FeatureKeys: []string{models.FeatureKeyAIDiagnosis}},
		{ID: models.SkillPackPostgreSQL, DisplayName: "PostgreSQL AI 技能包", FeatureKeys: []string{models.FeatureKeyAIDiagnosis}},
		{ID: models.SkillPackElasticsearch, DisplayName: "Elasticsearch AI 技能包", FeatureKeys: []string{models.FeatureKeyAIDiagnosis}},
	}
}

func packageByStripePriceID(cfg *config.Config, priceID string) *config.BillingPackage {
	priceID = strings.TrimSpace(priceID)
	pkgs := resolvedBillingPackages(cfg)
	for i := range pkgs {
		if pkgs[i].StripePriceID == priceID {
			return &pkgs[i]
		}
	}
	return nil
}

func packageByID(cfg *config.Config, id string) *config.BillingPackage {
	id = strings.TrimSpace(id)
	pkgs := resolvedBillingPackages(cfg)
	for i := range pkgs {
		if pkgs[i].ID == id {
			return &pkgs[i]
		}
	}
	return nil
}

func stripePriceIDForPack(cfg *config.Config, packKey string) string {
	if p := packageByID(cfg, packKey); p != nil && strings.TrimSpace(p.StripePriceID) != "" {
		return strings.TrimSpace(p.StripePriceID)
	}
	var row models.FeatureBillingSetting
	if err := database.DB.Where("pack_key = ? AND stripe_price_id <> ''", packKey).First(&row).Error; err == nil {
		return strings.TrimSpace(row.StripePriceID)
	}
	return ""
}

func normalizeGrantedKeys(keys []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" || !models.IsKnownFeatureKey(k) {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func featureKeysForStripePrice(cfg *config.Config, priceID string) []string {
	if p := packageByStripePriceID(cfg, priceID); p != nil {
		return []string{p.ID}
	}
	var row models.FeatureBillingSetting
	if err := database.DB.Where("stripe_price_id = ?", strings.TrimSpace(priceID)).First(&row).Error; err == nil {
		if row.PackKey != "" {
			return []string{row.PackKey}
		}
		return []string{models.DefaultPackKeyForFeature(row.FeatureKey)}
	}
	legacy := strings.TrimSpace(cfg.Billing.StripePriceIDPro)
	if legacy != "" && legacy == strings.TrimSpace(priceID) {
		return []string{models.PackKeyBackupPerformance}
	}
	return nil
}

// ListBillingPackages 返回可展示的订阅档位（不包含 Stripe 密钥）。
func ListBillingPackages(c *gin.Context) {
	cfg := c.MustGet("config").(*config.Config)
	type row struct {
		ID          string   `json:"id"`
		DisplayName string   `json:"display_name"`
		FeatureKeys []string `json:"feature_keys"`
		StripeReady bool     `json:"stripe_ready"`
	}
	out := make([]row, 0)
	for _, p := range resolvedBillingPackages(cfg) {
		out = append(out, row{
			ID:          p.ID,
			DisplayName: p.DisplayName,
			FeatureKeys: normalizeGrantedKeys(p.FeatureKeys),
			StripeReady: stripePriceIDForPack(cfg, p.ID) != "",
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": out, "msg": "success"})
}

// GetBillingCapabilities returns feature/package capability state for Web and CLI.
func GetBillingCapabilities(c *gin.Context) {
	cfg := c.MustGet("config").(*config.Config)
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	features := make([]gin.H, 0)
	for _, fk := range models.AllFeatureKeysStable() {
		viewAllowed, viewPayload := middleware.CheckCapability(uid, role, fk, middleware.CapabilityActionView)
		execAllowed, execPayload := middleware.CheckCapability(uid, role, fk, middleware.CapabilityActionExecute)
		setting := featureSettingForPayload(fk)
		packKey := setting.PackKey
		if packKey == "" {
			packKey = models.DefaultPackKeyForFeature(fk)
		}
		features = append(features, gin.H{
			"feature_key":       fk,
			"pack_key":          packKey,
			"description":       setting.Description,
			"visible_enabled":   setting.VisibleEnabled,
			"execution_enabled": setting.ExecutionEnabled,
			"billing_enabled":   setting.BillingEnabled,
			"can_view":          viewAllowed,
			"can_execute":       execAllowed,
			"view_state":        viewPayload,
			"execute_state":     execPayload,
		})
	}

	packages := make([]gin.H, 0)
	for _, p := range resolvedBillingPackages(cfg) {
		entitled := models.IsSuperAdminRole(role) || activeEntitlementExists(uid, append([]string{p.ID}, p.FeatureKeys...)...)
		packages = append(packages, gin.H{
			"pack_key":     p.ID,
			"display_name": p.DisplayName,
			"feature_keys": normalizeGrantedKeys(p.FeatureKeys),
			"stripe_ready": stripePriceIDForPack(cfg, p.ID) != "",
			"entitled":     entitled,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
		"role":           role,
		"billing_exempt": models.IsSuperAdminRole(role),
		"features":       features,
		"packages":       packages,
		"ai_quota": gin.H{
			"free_daily_limit": 5,
			"timezone":         "Asia/Shanghai",
		},
	}, "msg": "success"})
}

func firstFeatureForPackage(packKey string) string {
	switch packKey {
	case models.PackKeyK8sDelivery:
		return models.FeatureKeyK8sDelivery
	case models.PackKeyNodeOps:
		return models.FeatureKeyNodeOps
	case models.PackKeyMonitoring:
		return models.FeatureKeyMonitoring
	case models.PackKeyBackupPerformance:
		return models.FeatureKeyBackupPerformance
	case models.PackKeyRuntimeObserve:
		return models.FeatureKeyRuntimeObserve
	default:
		return models.FeatureKeyAIDiagnosis
	}
}

func featureSettingForPayload(featureKey string) models.FeatureBillingSetting {
	row := models.FeatureBillingSetting{
		FeatureKey:       featureKey,
		PackKey:          models.DefaultPackKeyForFeature(featureKey),
		VisibleEnabled:   true,
		ExecutionEnabled: true,
		BillingEnabled:   false,
		Description:      models.DefaultFeatureDescription(featureKey),
	}
	var dbRow models.FeatureBillingSetting
	if err := database.DB.Where("feature_key = ?", featureKey).First(&dbRow).Error; err == nil {
		if dbRow.PackKey == "" {
			dbRow.PackKey = row.PackKey
		}
		if dbRow.Description == "" {
			dbRow.Description = row.Description
		}
		return dbRow
	}
	return row
}

// GetBillingMe returns subscription + entitlements + feature flags for the current user.
func GetBillingMe(c *gin.Context) {
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	featureFlags := loadAllFeatureBillingMap()

	if models.IsSuperAdminRole(role) {
		featureAccess := gin.H{}
		for _, fk := range models.AllFeatureKeysStable() {
			featureAccess[fk] = true
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"billing_exempt":      true,
				"subscription":        nil,
				"entitlements":        []models.Entitlement{},
				"feature_flags":       featureFlags,
				"feature_access":      featureAccess,
				"can_use_advanced":    true,
				"can_manage_advanced": true,
				"can_use_k8s_ops":     true,
				"can_use_service_ops": true,
				"can_use_infra_ops":   true,
			},
			"msg": "success",
		})
		return
	}

	var sub models.Subscription
	var subPayload interface{}
	if err := database.DB.Where("user_id = ?", uid).First(&sub).Error; err == nil {
		subPayload = sub
	}

	var ents []models.Entitlement
	database.DB.Where("user_id = ?", uid).Find(&ents)

	featureAccess := gin.H{}
	for _, fk := range models.AllFeatureKeysStable() {
		featureAccess[fk] = featureAccessAllowed(uid, fk)
	}

	canAdv := featureAccessAllowed(uid, models.FeatureKeyAdvanced)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"billing_exempt":      false,
			"subscription":        subPayload,
			"entitlements":        ents,
			"feature_flags":       featureFlags,
			"feature_access":      featureAccess,
			"can_use_advanced":    canAdv,
			"can_manage_advanced": models.IsAdminRole(role) && canAdv,
			"can_use_k8s_ops":     featureAccessAllowed(uid, models.FeatureKeyK8sOps),
			"can_use_service_ops": featureAccessAllowed(uid, models.FeatureKeyServiceOps),
			"can_use_infra_ops":   featureAccessAllowed(uid, models.FeatureKeyInfraOps),
		},
		"msg": "success",
	})
}

func featureAccessAllowed(uid uuid.UUID, featureKey string) bool {
	var fs models.FeatureBillingSetting
	if err := database.DB.Where("feature_key = ?", featureKey).First(&fs).Error; err != nil {
		return true
	}
	packKey := fs.PackKey
	if packKey == "" {
		packKey = models.DefaultPackKeyForFeature(featureKey)
	}
	if !fs.BillingEnabled {
		return true
	}
	var ent models.Entitlement
	err := database.DB.Where("user_id = ? AND feature_key IN ?", uid, []string{packKey, featureKey}).
		Where("valid_until IS NULL OR valid_until > ?", time.Now().UTC()).
		First(&ent).Error
	return err == nil
}

func activeEntitlementExists(uid uuid.UUID, keys ...string) bool {
	clean := make([]string, 0, len(keys))
	seen := make(map[string]struct{})
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" || !models.IsKnownEntitlementKey(key) {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, key)
	}
	if len(clean) == 0 {
		return false
	}
	var ent models.Entitlement
	err := database.DB.Where("user_id = ? AND feature_key IN ?", uid, clean).
		Where("valid_until IS NULL OR valid_until > ?", time.Now().UTC()).
		First(&ent).Error
	return err == nil
}

func loadAllFeatureBillingMap() map[string]bool {
	var rows []models.FeatureBillingSetting
	if err := database.DB.Find(&rows).Error; err != nil {
		return map[string]bool{}
	}
	m := make(map[string]bool)
	for _, r := range rows {
		m[r.FeatureKey] = r.BillingEnabled
	}
	return m
}

// AdminListFeatureBilling returns all feature billing toggles.
func AdminListFeatureBilling(c *gin.Context) {
	var rows []models.FeatureBillingSetting
	if err := database.DB.Order("feature_key").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": rows, "msg": "success"})
}

type featureBillingUpdateItem struct {
	FeatureKey       string `json:"feature_key" binding:"required"`
	PackKey          string `json:"pack_key"`
	VisibleEnabled   *bool  `json:"visible_enabled"`
	ExecutionEnabled *bool  `json:"execution_enabled"`
	BillingEnabled   *bool  `json:"billing_enabled"`
	StripePriceID    string `json:"stripe_price_id"`
	Description      string `json:"description"`
}

// AdminPutFeatureBilling updates feature billing rows (batch).
func AdminPutFeatureBilling(c *gin.Context) {
	var body struct {
		Items []featureBillingUpdateItem `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数无效"})
		return
	}
	for _, it := range body.Items {
		if !models.IsKnownFeatureKey(it.FeatureKey) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "未知功能键"})
			return
		}
		if it.PackKey != "" && !models.IsKnownPackKey(it.PackKey) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "未知功能包"})
			return
		}
		var row models.FeatureBillingSetting
		if err := database.DB.Where("feature_key = ?", it.FeatureKey).First(&row).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				row = models.FeatureBillingSetting{
					FeatureKey:       it.FeatureKey,
					PackKey:          models.DefaultPackKeyForFeature(it.FeatureKey),
					VisibleEnabled:   true,
					ExecutionEnabled: true,
					Description:      models.DefaultFeatureDescription(it.FeatureKey),
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "数据库错误"})
				return
			}
		}
		if it.BillingEnabled != nil {
			row.BillingEnabled = *it.BillingEnabled
		}
		if it.VisibleEnabled != nil {
			row.VisibleEnabled = *it.VisibleEnabled
		}
		if it.ExecutionEnabled != nil {
			row.ExecutionEnabled = *it.ExecutionEnabled
		}
		if it.PackKey != "" {
			row.PackKey = it.PackKey
		}
		row.StripePriceID = strings.TrimSpace(it.StripePriceID)
		if it.Description != "" {
			row.Description = it.Description
		}
		row.UpdatedAt = time.Now().UTC()
		if err := database.DB.Save(&row).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存失败"})
			return
		}
	}
	AdminListFeatureBilling(c)
}

type grantEntitlementBody struct {
	FeatureKey string     `json:"feature_key"`
	PackKey    string     `json:"pack_key"`
	ValidUntil *time.Time `json:"valid_until"`
}

// AdminGrantEntitlement upserts a manual entitlement for a user.
func AdminGrantEntitlement(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}
	var body grantEntitlementBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数无效"})
		return
	}
	entitlementKey := strings.TrimSpace(body.PackKey)
	if entitlementKey == "" {
		entitlementKey = strings.TrimSpace(body.FeatureKey)
	}
	if !models.IsKnownEntitlementKey(entitlementKey) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "未知功能包或功能键"})
		return
	}
	var ent models.Entitlement
	err = database.DB.Where("user_id = ? AND feature_key = ? AND source = ?", id, entitlementKey, "manual").First(&ent).Error
	if err == gorm.ErrRecordNotFound {
		ent = models.Entitlement{
			ID:         uuid.New(),
			UserID:     id,
			FeatureKey: entitlementKey,
			ValidUntil: body.ValidUntil,
			Source:     "manual",
		}
		if err := database.DB.Create(&ent).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建失败"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	} else {
		ent.ValidUntil = body.ValidUntil
		ent.Source = "manual"
		if err := database.DB.Save(&ent).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新失败"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": ent, "msg": "success"})
}

// CreateStripeCheckoutSession starts a hosted Stripe Checkout for the current user.
func CreateStripeCheckoutSession(c *gin.Context) {
	cfg := c.MustGet("config").(*config.Config)
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	if models.IsSuperAdminRole(role) {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"url":            "",
				"billing_exempt": true,
				"message":        "超级管理员无需订阅",
			},
			"msg": "success",
		})
		return
	}
	pkgs := resolvedBillingPackages(cfg)
	if strings.TrimSpace(cfg.Billing.StripeSecretKey) == "" || len(pkgs) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "Stripe 未配置或未定义订阅包（billing.packages 或 stripe_price_id_pro）"})
		return
	}

	var body struct {
		PackageID string `json:"package_id"`
		PackKey   string `json:"pack_key"`
	}
	_ = c.ShouldBindJSON(&body)
	pkgID := strings.TrimSpace(body.PackKey)
	if pkgID == "" {
		pkgID = strings.TrimSpace(body.PackageID)
	}
	var sel *config.BillingPackage
	if pkgID != "" {
		sel = packageByID(cfg, pkgID)
		if sel == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "未知 package_id"})
			return
		}
	} else {
		for i := range pkgs {
			if stripePriceIDForPack(cfg, pkgs[i].ID) != "" {
				sel = &pkgs[i]
				break
			}
		}
		if sel == nil {
			sel = &pkgs[0]
		}
	}

	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}
	var user models.User
	if err := database.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	stripe.Key = cfg.Billing.StripeSecretKey

	var sub models.Subscription
	_ = database.DB.Where("user_id = ?", uid).First(&sub).Error
	customerID := strings.TrimSpace(sub.StripeCustomerID)
	if customerID == "" {
		cus, err := stripecustomer.New(&stripe.CustomerParams{
			Email: stripe.String(user.Email),
			Metadata: map[string]string{
				"opsfleet_user_id": uid.String(),
			},
		})
		if err != nil {
			logger.Error("stripe customer: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "msg": "Stripe 创建客户失败"})
			return
		}
		customerID = cus.ID
		sub.ID = uuid.New()
		sub.UserID = uid
		sub.StripeCustomerID = customerID
		sub.Status = "incomplete"
		if err := database.DB.Save(&sub).Error; err != nil {
			logger.Error("save subscription stub: %v", err)
		}
	}

	base := strings.TrimSuffix(strings.TrimSpace(cfg.Billing.PublicAppBaseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:9080"
	}
	successURL := base + "/app/dashboard?billing=success"
	cancelURL := base + "/app/dashboard?billing=cancel"

	md := map[string]string{
		"opsfleet_user_id":    uid.String(),
		"opsfleet_package_id": sel.ID,
	}
	priceID := stripePriceIDForPack(cfg, sel.ID)
	if priceID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "当前功能包未绑定 Stripe Price"})
		return
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:   stripe.String(customerID),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		Metadata: md,
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: md,
		},
	}
	sess, err := checkoutsession.New(params)
	if err != nil {
		logger.Error("stripe checkout session: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "msg": "创建收银台失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"url": sess.URL}, "msg": "success"})
}

// StripeWebhook handles Stripe events (public, signature verified).
func StripeWebhook(c *gin.Context) {
	cfg := c.MustGet("config").(*config.Config)
	secret := strings.TrimSpace(cfg.Billing.StripeWebhookSecret)
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "webhook 未配置"})
		return
	}
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "读取 body 失败"})
		return
	}
	sig := c.GetHeader("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, secret)
	if err != nil {
		logger.Warn("stripe webhook verify: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "签名校验失败"})
		return
	}

	stripe.Key = cfg.Billing.StripeSecretKey

	switch event.Type {
	case "checkout.session.completed":
		var rawSess struct {
			Subscription json.RawMessage   `json:"subscription"`
			Metadata     map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(event.Data.Raw, &rawSess); err != nil {
			logger.Warn("stripe session unmarshal: %v", err)
			break
		}
		sid := strings.Trim(string(rawSess.Subscription), `"`)
		if sid == "" || sid == "null" {
			break
		}
		full, err := stripeSub.Get(sid, nil)
		if err != nil {
			logger.Error("stripe get sub: %v", err)
			break
		}
		if full.Metadata == nil {
			full.Metadata = map[string]string{}
		}
		if rawSess.Metadata != nil && full.Metadata["opsfleet_user_id"] == "" {
			if v := rawSess.Metadata["opsfleet_user_id"]; v != "" {
				full.Metadata["opsfleet_user_id"] = v
			}
		}
		if rawSess.Metadata != nil && full.Metadata["opsfleet_package_id"] == "" {
			if v := rawSess.Metadata["opsfleet_package_id"]; v != "" {
				full.Metadata["opsfleet_package_id"] = v
			}
		}
		applyStripeSubscription(cfg, full, nil)
	case "customer.subscription.updated", "customer.subscription.deleted":
		var full stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &full); err != nil {
			logger.Warn("stripe sub unmarshal: %v", err)
			break
		}
		applyStripeSubscription(cfg, &full, nil)
	default:
		// ignore
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func applyStripeSubscription(cfg *config.Config, full *stripe.Subscription, sess *stripe.CheckoutSession) {
	if full == nil {
		return
	}
	uidStr := ""
	if full.Metadata != nil {
		uidStr = full.Metadata["opsfleet_user_id"]
	}
	if uidStr == "" && sess != nil && sess.Metadata != nil {
		uidStr = sess.Metadata["opsfleet_user_id"]
	}
	if uidStr == "" && full.Customer != nil {
		var sub models.Subscription
		if err := database.DB.Where("stripe_customer_id = ?", full.Customer.ID).First(&sub).Error; err == nil {
			uidStr = sub.UserID.String()
		}
	}
	uid, err := uuid.Parse(uidStr)
	if err != nil {
		logger.Warn("stripe webhook missing opsfleet_user_id for sub %s", full.ID)
		return
	}

	priceID := ""
	if full.Items != nil && len(full.Items.Data) > 0 && full.Items.Data[0].Price != nil {
		priceID = full.Items.Data[0].Price.ID
	}

	var row models.Subscription
	err = database.DB.Where("user_id = ?", uid).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = models.Subscription{ID: uuid.New(), UserID: uid}
	} else if err != nil {
		logger.Error("subscription lookup: %v", err)
		return
	}
	if full.Customer != nil {
		row.StripeCustomerID = full.Customer.ID
	}
	row.StripeSubscriptionID = full.ID
	row.Status = string(full.Status)
	row.PlanID = priceID
	if full.CurrentPeriodEnd > 0 {
		t := time.Unix(full.CurrentPeriodEnd, 0).UTC()
		row.CurrentPeriodEnd = &t
	} else {
		row.CurrentPeriodEnd = nil
	}
	if err := database.DB.Save(&row).Error; err != nil {
		logger.Error("subscription save: %v", err)
		return
	}

	keys := featureKeysForStripePrice(cfg, priceID)
	if len(keys) == 0 && full.Metadata != nil {
		pid := strings.TrimSpace(full.Metadata["opsfleet_package_id"])
		if p := packageByID(cfg, pid); p != nil {
			keys = []string{p.ID}
		}
	}

	source := stripeEntitlementSource(full.ID)
	active := full.Status == stripe.SubscriptionStatusActive || full.Status == stripe.SubscriptionStatusTrialing
	if !active {
		if len(keys) == 0 {
			database.DB.Where("user_id = ? AND source = ?", uid, source).Delete(&models.Entitlement{})
			return
		}
		database.DB.Where("user_id = ? AND feature_key IN ? AND source IN ?", uid, keys, []string{"stripe", source}).Delete(&models.Entitlement{})
		return
	}

	if len(keys) == 0 {
		logger.Warn("stripe webhook: unknown price/package for user %s price=%s — clearing current subscription entitlements", uid.String(), priceID)
		database.DB.Where("user_id = ? AND source = ?", uid, source).Delete(&models.Entitlement{})
		return
	}

	validUntil := row.CurrentPeriodEnd
	tx := database.DB.Begin()
	if tx.Error != nil {
		logger.Error("stripe ent tx: %v", tx.Error)
		return
	}
	if err := tx.Where("user_id = ? AND source = ? AND feature_key IN ?", uid, "stripe", keys).
		Delete(&models.Entitlement{}).Error; err != nil {
		tx.Rollback()
		logger.Error("stripe ent delete legacy: %v", err)
		return
	}
	if err := tx.Where("user_id = ? AND source = ? AND feature_key NOT IN ?", uid, source, keys).
		Delete(&models.Entitlement{}).Error; err != nil {
		tx.Rollback()
		logger.Error("stripe ent delete current stray: %v", err)
		return
	}
	for _, fk := range keys {
		var ent models.Entitlement
		err := tx.Where("user_id = ? AND feature_key = ? AND source = ?", uid, fk, source).First(&ent).Error
		if err == gorm.ErrRecordNotFound {
			ent = models.Entitlement{
				ID:         uuid.New(),
				UserID:     uid,
				FeatureKey: fk,
				ValidUntil: validUntil,
				Source:     source,
			}
			if err := tx.Create(&ent).Error; err != nil {
				tx.Rollback()
				logger.Error("stripe ent create: %v", err)
				return
			}
		} else if err != nil {
			tx.Rollback()
			logger.Error("stripe ent lookup: %v", err)
			return
		} else {
			ent.ValidUntil = validUntil
			ent.Source = source
			if err := tx.Save(&ent).Error; err != nil {
				tx.Rollback()
				logger.Error("stripe ent save: %v", err)
				return
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		logger.Error("stripe ent commit: %v", err)
	}
}

func stripeEntitlementSource(subscriptionID string) string {
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return "stripe"
	}
	return "stripe:" + subscriptionID
}
