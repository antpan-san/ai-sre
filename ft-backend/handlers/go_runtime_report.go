package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/middleware"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type goRuntimeReportBody struct {
	Command string          `json:"command"`
	Host    string          `json:"host"`
	Watch   json.RawMessage `json:"watch"`
	Client  aiClientInfo    `json:"client"`
}

func CheckCLIGoRuntimeAuth(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	allowed, payload := middleware.CheckCapability(ident.UserID, ident.Role, models.FeatureKeyRuntimeObserve, middleware.CapabilityActionExecute)
	if !allowed {
		code, _ := payload["code"].(int)
		if code == 0 {
			code = http.StatusForbidden
		}
		c.JSON(code, payload)
		return
	}
	bindingID := ""
	if ident.CLIBindingID != nil {
		bindingID = ident.CLIBindingID.String()
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": gin.H{
		"user_id":             ident.UserID,
		"username":            ident.Username,
		"role":                ident.Role,
		"auth_kind":           ident.AuthKind,
		"cli_binding_id":      bindingID,
		"feature_key":         models.FeatureKeyRuntimeObserve,
		"pack_key":            models.PackKeyRuntimeObserve,
		"fingerprint_matched": true,
	}})
}

func PostCLIGoRuntimeReport(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	allowed, payload := middleware.CheckCapability(ident.UserID, ident.Role, models.FeatureKeyRuntimeObserve, middleware.CapabilityActionExecute)
	if !allowed {
		code, _ := payload["code"].(int)
		if code == 0 {
			code = http.StatusForbidden
		}
		c.JSON(code, payload)
		return
	}
	var body goRuntimeReportBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数: " + err.Error()})
		return
	}
	if len(body.Watch) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "watch 不能为空"})
		return
	}
	if len(body.Watch) > runtimeWatchSampleMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "watch 体积过大"})
		return
	}
	var watch map[string]interface{}
	if err := json.Unmarshal(body.Watch, &watch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "watch JSON 无效"})
		return
	}
	target := mapFromAny(watch["target"])
	summary := mapFromAny(watch["summary"])
	ns := cleanString(target["namespace"])
	pod := cleanString(target["pod"])
	container := cleanString(target["container"])
	source := cleanString(target["source"])
	pid := cleanString(target["pid"])
	targetName := cleanString(target["target"])
	if ns == "" {
		ns = "local"
	}
	if pod == "" {
		if pid != "" {
			pod = "pid-" + pid
		} else if targetName != "" {
			pod = targetName
		} else {
			pod = "go-process"
		}
	}
	interval := intFromAny(watch["interval_seconds"])
	if interval <= 0 {
		interval = 10
	}
	plain, hash, err := newRuntimeWatchTokenPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "生成会话令牌失败"})
		return
	}
	_ = plain
	now := time.Now().UTC()
	sess := models.RuntimeWatchSession{
		UserID:          ident.UserID,
		Namespace:       ns,
		Pod:             pod,
		Container:       container,
		IntervalSec:     interval,
		Status:          "stopped",
		SampleTokenHash: hash,
		MachineNote:     "CLI 自动诊断",
	}
	sample := models.RuntimeWatchSample{
		SessionID:  sess.ID,
		ObservedAt: now,
		Payload:    models.JSONB(append(json.RawMessage(nil), body.Watch...)),
	}
	targetHost := cleanString(target["node"])
	if targetHost == "" {
		targetHost = strings.TrimSpace(body.Host)
	}
	resourceType := "go_process"
	if source == "kubernetes" {
		resourceType = "k8s_pod"
	}
	resourceName := targetName
	if resourceName == "" {
		resourceName = strings.Trim(strings.Join([]string{ns, pod, container}, "/"), "/")
	}
	meta := map[string]interface{}{
		"record_kind":      "go_runtime",
		"feature_key":      models.FeatureKeyRuntimeObserve,
		"pack_key":         models.PackKeyRuntimeObserve,
		"auth_kind":        ident.AuthKind,
		"cli_binding_id":   "",
		"fingerprint_hash": ident.FingerprintHash,
		"client": gin.H{
			"version":          strings.TrimSpace(body.Client.Version),
			"binding_id":       strings.TrimSpace(body.Client.BindingID),
			"fingerprint_hash": strings.TrimSpace(body.Client.FingerprintHash),
		},
		"target":  target,
		"summary": summary,
	}
	if ident.CLIBindingID != nil {
		meta["cli_binding_id"] = ident.CLIBindingID.String()
	}
	effects := map[string]interface{}{
		"summary": summary,
		"watch":   watch,
	}
	status := models.ExecutionStatusSuccess
	rec := models.ExecutionRecord{
		CorrelationID:      uuid.NewString(),
		Source:             "cli",
		Category:           "go_runtime",
		Name:               "Go Runtime 诊断",
		Command:            limitText(body.Command, 2000),
		CommandDigest:      digestText(body.Command),
		Status:             status,
		CreatedBy:          ident.Username,
		TriggerUser:        ident.Username,
		TargetHost:         targetHost,
		ResourceType:       resourceType,
		ResourceID:         cleanString(target["container_id"]),
		ResourceName:       resourceName,
		StartedAt:          &now,
		FinishedAt:         &now,
		StdoutSummary:      goRuntimeSummaryText(summary),
		Effects:            models.NewJSONBFromMap(effects),
		Metadata:           models.NewJSONBFromMap(meta),
		RollbackCapability: models.RollbackCapabilityNone,
		RollbackStatus:     models.RollbackStatusNotStarted,
		RollbackPlan:       models.NewJSONBFromMap(map[string]interface{}{}),
		RollbackAdvice:     "Go runtime 诊断只读采集，不产生可回滚变更。",
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&sess).Error; err != nil {
			return err
		}
		sample.SessionID = sess.ID
		if err := tx.Create(&sample).Error; err != nil {
			return err
		}
		meta["runtime_watch_session_id"] = sess.ID.String()
		meta["runtime_watch_sample_id"] = sample.ID.String()
		rec.Metadata = models.NewJSONBFromMap(meta)
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		return tx.Create(&models.ExecutionEvent{
			ExecutionID: rec.ID,
			Level:       "info",
			Phase:       "finish",
			Message:     "Go Runtime 诊断完成",
			Output:      goRuntimeSummaryText(summary),
			Details:     models.NewJSONBFromMap(meta),
		}).Error
	})
	if err != nil {
		logger.Error("go runtime report insert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存 Go runtime 报告失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": gin.H{
		"execution_record_id":      rec.ID,
		"runtime_watch_session_id": sess.ID,
		"runtime_watch_sample_id":  sample.ID,
	}})
}

func resolveCLIBearerIdentity(c *gin.Context) (*aiIdentity, bool) {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "Authorization header format must be Bearer {token}"})
		return nil, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "Authorization token is empty"})
		return nil, false
	}
	ident, err := resolveCLIIdentity(token, strings.TrimSpace(c.GetHeader("X-OpsFleet-CLI-Fingerprint")))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": err.Error()})
		return nil, false
	}
	return ident, true
}

func mapFromAny(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func cleanString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	case json.Number:
		return t.String()
	case nil:
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func intFromAny(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	default:
		return 0
	}
}

func goRuntimeSummaryText(summary map[string]interface{}) string {
	level := cleanString(summary["level"])
	title := cleanString(summary["title"])
	evidence := cleanString(summary["evidence"])
	if level == "" && title == "" {
		return "Go Runtime 诊断完成"
	}
	parts := []string{}
	if level != "" {
		parts = append(parts, "["+level+"]")
	}
	if title != "" {
		parts = append(parts, title)
	}
	if evidence != "" {
		parts = append(parts, evidence)
	}
	return strings.Join(parts, " ")
}
