package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type aiClientInfo struct {
	Version         string `json:"version"`
	BindingID       string `json:"binding_id"`
	FingerprintHash string `json:"fingerprint_hash"`
}

type aiIdentity struct {
	Subject         string
	UserID          uuid.UUID
	Username        string
	Role            string
	AuthKind        string
	CLIBindingID    *uuid.UUID
	FingerprintHash string
}

type aiQuotaDecision struct {
	PackKey           string
	EntitlementSource string
	FreeDailyLimit    int
	UsedBefore        int
	RemainingBefore   int
	ConsumesFree      bool
	BillingExempt     bool
}

func resolveAIIdentity(c *gin.Context) (*aiIdentity, bool) {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if auth == "" {
		return &aiIdentity{Subject: "ip:" + c.ClientIP(), AuthKind: "anonymous"}, true
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "Authorization header format must be Bearer {token}"})
		return nil, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "Authorization token is empty"})
		return nil, false
	}
	if strings.Count(token, ".") == 2 {
		cfgVal, _ := c.Get("config")
		cfg, _ := cfgVal.(*config.Config)
		if cfg != nil {
			if claims, err := utils.ValidateToken(token, cfg.JWT.SecretKey); err == nil {
				uid, _ := uuid.Parse(claims.UserID)
				if uid != uuid.Nil {
					return &aiIdentity{
						Subject:  "user:" + uid.String(),
						UserID:   uid,
						Username: claims.Username,
						Role:     claims.Role,
						AuthKind: "jwt",
					}, true
				}
			}
		}
	}
	ident, err := resolveCLIIdentity(token, strings.TrimSpace(c.GetHeader("X-OpsFleet-CLI-Fingerprint")))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": err.Error()})
		return nil, false
	}
	return ident, true
}

func resolveCLIIdentity(token, fingerprint string) (*aiIdentity, error) {
	if !isHexLen(fingerprint, 64) {
		return nil, fmt.Errorf("CLI 机器指纹缺失或无效")
	}
	var binding models.CLIBinding
	err := database.DB.Where("token_hash = ?", hashSecret(token)).First(&binding).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("CLI token 无效")
		}
		logger.Error("cli token lookup: %v", err)
		return nil, fmt.Errorf("CLI token 校验失败")
	}
	now := time.Now().UTC()
	if binding.RevokedAt != nil || now.After(binding.ExpiresAt) {
		return nil, fmt.Errorf("CLI token 已失效")
	}
	if subtle.ConstantTimeCompare([]byte(binding.FingerprintHash), []byte(strings.ToLower(fingerprint))) != 1 {
		return nil, fmt.Errorf("CLI token 与当前机器不匹配")
	}
	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", binding.UserID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("CLI token 关联账号不存在")
	}
	_ = database.DB.Model(&binding).Update("last_used_at", now).Error
	id := binding.ID
	return &aiIdentity{
		Subject:         "user:" + binding.UserID.String(),
		UserID:          binding.UserID,
		Username:        user.Username,
		Role:            user.Role,
		AuthKind:        "cli",
		CLIBindingID:    &id,
		FingerprintHash: binding.FingerprintHash,
	}, nil
}

func beginAIQuotaForIdentity(c *gin.Context, packKey string, ident *aiIdentity) (func(bool), aiQuotaDecision, bool) {
	decision := aiQuotaDecision{PackKey: packKey, FreeDailyLimit: aiFreeDailyLimit, EntitlementSource: "free"}
	if ident != nil && ident.UserID != uuid.Nil {
		if models.IsSuperAdminRole(ident.Role) {
			decision.EntitlementSource = "super_admin"
			decision.BillingExempt = true
			decision.RemainingBefore = -1
			return func(bool) {}, decision, true
		}
		var ent models.Entitlement
		if err := database.DB.Where("user_id = ? AND feature_key = ?", ident.UserID, packKey).
			Where("valid_until IS NULL OR valid_until > ?", time.Now().UTC()).
			First(&ent).Error; err == nil {
			decision.EntitlementSource = ent.Source
			if decision.EntitlementSource == "" {
				decision.EntitlementSource = "entitlement"
			}
			decision.RemainingBefore = -1
			return func(bool) {}, decision, true
		}
	}
	subject := "ip:" + c.ClientIP()
	if ident != nil && ident.Subject != "" {
		subject = ident.Subject
	}
	date := aiQuotaDate()
	var usage models.AIUsage
	err := database.DB.Where("subject = ? AND pack_key = ? AND usage_date = ?", subject, packKey, date).First(&usage).Error
	if err == gorm.ErrRecordNotFound {
		usage = models.AIUsage{ID: uuid.New(), Subject: subject, PackKey: packKey, UsageDate: date, Count: 0}
		if err := database.DB.Create(&usage).Error; err != nil {
			logger.Error("ai quota create: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "AI 免费额度检查失败"})
			return nil, decision, false
		}
	} else if err != nil {
		logger.Error("ai quota lookup: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "AI 免费额度检查失败"})
		return nil, decision, false
	}
	decision.UsedBefore = usage.Count
	decision.RemainingBefore = aiFreeDailyLimit - usage.Count
	decision.ConsumesFree = true
	if usage.Count >= aiFreeDailyLimit {
		c.JSON(http.StatusForbidden, gin.H{
			"code":               403,
			"msg":                "今日免费 AI 诊断次数已用完，请订阅对应技能包",
			"biz":                "PAYWALL_" + packKey,
			"feature_key":        models.FeatureKeyAIDiagnosis,
			"pack_key":           packKey,
			"reason":             "ai_free_quota_exhausted",
			"checkout_available": true,
			"ai_quota": gin.H{
				"free_daily_limit": aiFreeDailyLimit,
				"used":             usage.Count,
				"remaining":        0,
				"usage_date":       date,
				"timezone":         "Asia/Shanghai",
			},
		})
		return func(bool) {}, decision, false
	}
	return func(success bool) {
		if !success {
			return
		}
		if err := database.DB.Model(&models.AIUsage{}).
			Where("id = ?", usage.ID).
			UpdateColumn("count", gorm.Expr("count + 1")).Error; err != nil {
			logger.Error("ai quota increment: %v", err)
		}
	}, decision, true
}

func recordAIExecution(ident *aiIdentity, category, name, command, requestID, packKey, status, answer, errText string, ctxKV map[string]string, client aiClientInfo, decision aiQuotaDecision) {
	if ident == nil || ident.UserID == uuid.Nil {
		return
	}
	now := time.Now()
	meta := map[string]interface{}{
		"record_kind":        "ai_call",
		"feature_key":        models.FeatureKeyAIDiagnosis,
		"pack_key":           packKey,
		"skill_pack":         packKey,
		"auth_kind":          ident.AuthKind,
		"entitlement_source": decision.EntitlementSource,
		"quota_used":         quotaUsedAfter(decision, status),
		"quota_remaining":    quotaRemainingAfter(decision, status),
		"request_id":         strings.TrimSpace(requestID),
		"context":            summarizeAIContext(ctxKV),
		"client": gin.H{
			"version":          strings.TrimSpace(client.Version),
			"binding_id":       strings.TrimSpace(client.BindingID),
			"fingerprint_hash": strings.TrimSpace(client.FingerprintHash),
		},
	}
	if ident.CLIBindingID != nil {
		meta["cli_binding_id"] = ident.CLIBindingID.String()
		meta["fingerprint_hash"] = ident.FingerprintHash
	}
	effects := map[string]interface{}{
		"answer_summary": headSample(answer, 800),
	}
	if strings.TrimSpace(errText) != "" {
		effects["error"] = limitText(errText, 1200)
	}
	rec := models.ExecutionRecord{
		CorrelationID:      defaultString(requestID, uuid.NewString()),
		Source:             "ai",
		Category:           category,
		Name:               name,
		Command:            limitText(command, 2000),
		CommandDigest:      digestText(command),
		Status:             status,
		CreatedBy:          ident.Username,
		TriggerUser:        ident.Username,
		StartedAt:          &now,
		FinishedAt:         &now,
		DurationMs:         0,
		StdoutSummary:      headSample(answer, 1200),
		StderrSummary:      limitText(errText, 1200),
		Effects:            models.NewJSONBFromMap(effects),
		Metadata:           models.NewJSONBFromMap(meta),
		RollbackCapability: models.RollbackCapabilityNone,
		RollbackStatus:     models.RollbackStatusNotStarted,
		RollbackPlan:       models.NewJSONBFromMap(map[string]interface{}{}),
		RollbackAdvice:     "AI 调用记录不可回滚。",
	}
	if err := database.DB.Create(&rec).Error; err != nil {
		logger.Error("record ai execution: %v", err)
		return
	}
	_ = database.DB.Create(&models.ExecutionEvent{
		ExecutionID: rec.ID,
		Level:       logLevelFromStatus(status),
		Phase:       "finish",
		Message:     "AI 调用结束: " + status,
		Output:      headSample(answer, 1200),
		Details:     models.NewJSONBFromMap(meta),
	}).Error
}

func quotaUsedAfter(d aiQuotaDecision, status string) int {
	if !d.ConsumesFree {
		return d.UsedBefore
	}
	if status == models.ExecutionStatusSuccess {
		return d.UsedBefore + 1
	}
	return d.UsedBefore
}

func quotaRemainingAfter(d aiQuotaDecision, status string) int {
	if !d.ConsumesFree {
		return d.RemainingBefore
	}
	used := quotaUsedAfter(d, status)
	if used >= d.FreeDailyLimit {
		return 0
	}
	return d.FreeDailyLimit - used
}

func summarizeAIContext(kv map[string]string) map[string]interface{} {
	out := map[string]interface{}{"keys": []string{}, "bytes": 0}
	if len(kv) == 0 {
		return out
	}
	keys := make([]string, 0, len(kv))
	total := 0
	for k, v := range kv {
		keys = append(keys, k)
		total += len(k) + len(v)
	}
	sort.Strings(keys)
	out["keys"] = keys
	out["bytes"] = total
	return out
}

func decodeJSONMap(v models.JSONB) map[string]interface{} {
	var out map[string]interface{}
	_ = json.Unmarshal(v, &out)
	if out == nil {
		out = map[string]interface{}{}
	}
	return out
}
