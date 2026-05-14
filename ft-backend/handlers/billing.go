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
	if cfg == nil {
		return nil
	}
	if len(cfg.Billing.Packages) > 0 {
		out := make([]config.BillingPackage, 0, len(cfg.Billing.Packages))
		for _, p := range cfg.Billing.Packages {
			if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.StripePriceID) == "" {
				continue
			}
			out = append(out, p)
		}
		return out
	}
	if pid := strings.TrimSpace(cfg.Billing.StripePriceIDPro); pid != "" {
		return []config.BillingPackage{{
			ID:            "pro_legacy",
			DisplayName:   "Pro（兼容单包）",
			StripePriceID: pid,
			FeatureKeys:   []string{models.FeatureKeyAdvanced},
		}}
	}
	return nil
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
		return normalizeGrantedKeys(p.FeatureKeys)
	}
	legacy := strings.TrimSpace(cfg.Billing.StripePriceIDPro)
	if legacy != "" && legacy == strings.TrimSpace(priceID) {
		return []string{models.FeatureKeyAdvanced}
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
	}
	out := make([]row, 0)
	for _, p := range resolvedBillingPackages(cfg) {
		out = append(out, row{
			ID:          p.ID,
			DisplayName: p.DisplayName,
			FeatureKeys: normalizeGrantedKeys(p.FeatureKeys),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": out, "msg": "success"})
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
				"billing_exempt":       true,
				"subscription":         nil,
				"entitlements":         []models.Entitlement{},
				"feature_flags":        featureFlags,
				"feature_access":       featureAccess,
				"can_use_advanced":     true,
				"can_manage_advanced":  true,
				"can_use_k8s_ops":      true,
				"can_use_service_ops":  true,
				"can_use_infra_ops":    true,
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
	if !fs.BillingEnabled {
		return true
	}
	var ent models.Entitlement
	err := database.DB.Where("user_id = ? AND feature_key = ?", uid, featureKey).
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
	FeatureKey     string `json:"feature_key" binding:"required"`
	BillingEnabled *bool  `json:"billing_enabled"`
	Description    string `json:"description"`
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
		var row models.FeatureBillingSetting
		if err := database.DB.Where("feature_key = ?", it.FeatureKey).First(&row).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				row = models.FeatureBillingSetting{FeatureKey: it.FeatureKey}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "数据库错误"})
				return
			}
		}
		if it.BillingEnabled != nil {
			row.BillingEnabled = *it.BillingEnabled
		}
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
	FeatureKey string     `json:"feature_key" binding:"required"`
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
	if !models.IsKnownFeatureKey(body.FeatureKey) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "未知功能键"})
		return
	}
	var ent models.Entitlement
	err = database.DB.Where("user_id = ? AND feature_key = ? AND source = ?", id, body.FeatureKey, "manual").First(&ent).Error
	if err == gorm.ErrRecordNotFound {
		ent = models.Entitlement{
			ID:         uuid.New(),
			UserID:     id,
			FeatureKey: body.FeatureKey,
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
	}
	_ = c.ShouldBindJSON(&body)
	pkgID := strings.TrimSpace(body.PackageID)
	var sel *config.BillingPackage
	if pkgID != "" {
		sel = packageByID(cfg, pkgID)
		if sel == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "未知 package_id"})
			return
		}
	} else {
		sel = &pkgs[0]
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

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:   stripe.String(customerID),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(sel.StripePriceID), Quantity: stripe.Int64(1)},
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

	active := full.Status == stripe.SubscriptionStatusActive || full.Status == stripe.SubscriptionStatusTrialing
	if !active {
		database.DB.Where("user_id = ? AND source = ?", uid, "stripe").Delete(&models.Entitlement{})
		return
	}

	keys := featureKeysForStripePrice(cfg, priceID)
	if len(keys) == 0 && full.Metadata != nil {
		pid := strings.TrimSpace(full.Metadata["opsfleet_package_id"])
		if p := packageByID(cfg, pid); p != nil {
			keys = normalizeGrantedKeys(p.FeatureKeys)
		}
	}

	if len(keys) == 0 {
		logger.Warn("stripe webhook: unknown price/package for user %s price=%s — clearing stripe entitlements", uid.String(), priceID)
		database.DB.Where("user_id = ? AND source = ?", uid, "stripe").Delete(&models.Entitlement{})
		return
	}

	validUntil := row.CurrentPeriodEnd
	tx := database.DB.Begin()
	if tx.Error != nil {
		logger.Error("stripe ent tx: %v", tx.Error)
		return
	}
	if err := tx.Where("user_id = ? AND source = ? AND feature_key NOT IN ?", uid, "stripe", keys).
		Delete(&models.Entitlement{}).Error; err != nil {
		tx.Rollback()
		logger.Error("stripe ent delete stray: %v", err)
		return
	}
	for _, fk := range keys {
		var ent models.Entitlement
		err := tx.Where("user_id = ? AND feature_key = ? AND source = ?", uid, fk, "stripe").First(&ent).Error
		if err == gorm.ErrRecordNotFound {
			ent = models.Entitlement{
				ID:         uuid.New(),
				UserID:     uid,
				FeatureKey: fk,
				ValidUntil: validUntil,
				Source:     "stripe",
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
			ent.Source = "stripe"
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
