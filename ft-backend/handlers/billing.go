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

// GetBillingMe returns subscription + entitlements + feature flags for the current user.
func GetBillingMe(c *gin.Context) {
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	if role == "admin" {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"billing_exempt": true,
				"subscription":   nil,
				"entitlements":   []models.Entitlement{},
				"feature_flags":  loadAllFeatureBillingMap(),
			},
			"msg": "success",
		})
		return
	}

	var sub models.Subscription
	_ = database.DB.Where("user_id = ?", uid).First(&sub).Error

	var ents []models.Entitlement
	database.DB.Where("user_id = ?", uid).Find(&ents)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"billing_exempt": false,
			"subscription":   sub,
			"entitlements":   ents,
			"feature_flags":  loadAllFeatureBillingMap(),
		},
		"msg": "success",
	})
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
	var ent models.Entitlement
	err = database.DB.Where("user_id = ? AND feature_key = ?", id, body.FeatureKey).First(&ent).Error
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
	if strings.TrimSpace(cfg.Billing.StripeSecretKey) == "" || strings.TrimSpace(cfg.Billing.StripePriceIDPro) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "Stripe 未配置（stripe_secret_key / stripe_price_id_pro）"})
		return
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

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:   stripe.String(customerID),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(cfg.Billing.StripePriceIDPro), Quantity: stripe.Int64(1)},
		},
		Metadata: map[string]string{
			"opsfleet_user_id": uid.String(),
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"opsfleet_user_id": uid.String(),
			},
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
			Subscription json.RawMessage     `json:"subscription"`
			Metadata     map[string]string   `json:"metadata"`
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
		applyStripeSubscription(full, nil)
	case "customer.subscription.updated", "customer.subscription.deleted":
		var full stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &full); err != nil {
			logger.Warn("stripe sub unmarshal: %v", err)
			break
		}
		applyStripeSubscription(&full, nil)
	default:
		// ignore
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func applyStripeSubscription(full *stripe.Subscription, sess *stripe.CheckoutSession) {
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
	row.PlanID = ""
	if full.Items != nil && len(full.Items.Data) > 0 && full.Items.Data[0].Price != nil {
		row.PlanID = full.Items.Data[0].Price.ID
	}
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
		database.DB.Where("user_id = ? AND feature_key = ? AND source = ?", uid, models.FeatureKeyAdvanced, "stripe").Delete(&models.Entitlement{})
		return
	}
	var ent models.Entitlement
	err = database.DB.Where("user_id = ? AND feature_key = ?", uid, models.FeatureKeyAdvanced).First(&ent).Error
	if err == gorm.ErrRecordNotFound {
		ent = models.Entitlement{
			ID:         uuid.New(),
			UserID:     uid,
			FeatureKey: models.FeatureKeyAdvanced,
			ValidUntil: row.CurrentPeriodEnd,
			Source:     "stripe",
		}
		_ = database.DB.Create(&ent).Error
	} else if err == nil {
		ent.ValidUntil = row.CurrentPeriodEnd
		ent.Source = "stripe"
		_ = database.DB.Save(&ent).Error
	}
}
