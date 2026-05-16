package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const diagnosticPlanTTL = 5 * time.Minute
const diagnosticObservationMaxBytes = 512 * 1024

type diagnosticPlanStep struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Argv           []string `json:"argv"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	EvidenceKey    string   `json:"evidence_key"`
}

type cliDiagnosticPlanRequest struct {
	Topic     string            `json:"topic" binding:"required"`
	Context   map[string]string `json:"context"`
	Command   string            `json:"command"`
	RequestID string            `json:"request_id"`
	Client    aiClientInfo      `json:"client"`
}

func CreateCLIDiagnosticPlan(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	var req cliDiagnosticPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	topic := strings.TrimSpace(strings.ToLower(req.Topic))
	if topic == "" {
		response.BadRequest(c, "topic 不能为空")
		return
	}
	commitQuota, quotaDecision, quotaOK := beginAIQuotaForIdentity(c, skillPackForTopic(topic), ident)
	if !quotaOK {
		_ = commitQuota
		recordAIExecution(ident, "diagnostic_plan", "诊断任务单: "+topic, defaultString(req.Command, "ai-sre analyze "+topic), req.RequestID, quotaDecision.PackKey, models.ExecutionStatusFailed, "", "ai_free_quota_exhausted", req.Context, req.Client, quotaDecision)
		return
	}

	steps, err := buildReadonlyDiagnosticPlan(topic, req.Context)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	for _, st := range steps {
		if !allowedReadonlyDiagnosticCommand(st.Argv) {
			logger.Warn("blocked unsafe diagnostic plan step topic=%s argv=%v", topic, st.Argv)
			response.ServerError(c, "诊断任务单包含不允许的命令")
			return
		}
	}
	token, err := randomTokenHex(32)
	if err != nil {
		response.ServerError(c, "生成诊断任务单失败")
		return
	}
	now := time.Now().UTC()
	plan := models.DiagnosticPlan{
		UserID:          ident.UserID,
		Username:        ident.Username,
		CLIBindingID:    ident.CLIBindingID,
		FingerprintHash: ident.FingerprintHash,
		Topic:           topic,
		Command:         limitText(req.Command, 2000),
		RequestID:       strings.TrimSpace(req.RequestID),
		Status:          models.DiagnosticPlanStatusPending,
		PlanTokenHash:   hashSecret(token),
		ExpiresAt:       now.Add(diagnosticPlanTTL),
		Steps:           models.NewJSONBFromSlice(steps),
		Observations:    models.NewJSONBFromMap(map[string]interface{}{}),
	}
	if err := database.DB.Create(&plan).Error; err != nil {
		logger.Error("create diagnostic plan: %v", err)
		response.ServerError(c, "保存诊断任务单失败")
		return
	}
	commitQuota(true)
	recordAIExecution(ident, "diagnostic_plan", "诊断任务单: "+topic, defaultString(req.Command, "ai-sre analyze "+topic), req.RequestID, quotaDecision.PackKey, models.ExecutionStatusSuccess, "", "", req.Context, req.Client, quotaDecision)
	response.OK(c, gin.H{
		"plan_id":               plan.ID.String(),
		"plan_token":            token,
		"topic":                 topic,
		"expires_at":            plan.ExpiresAt,
		"requires_confirmation": true,
		"steps":                 steps,
		"policy": gin.H{
			"mode":          "readonly_preview",
			"non_tty_needs": "--yes",
		},
		"quota": gin.H{
			"pack_key":           quotaDecision.PackKey,
			"entitlement_source": quotaDecision.EntitlementSource,
			"remaining_before":   quotaDecision.RemainingBefore,
		},
	})
}

type cliDiagnosticObservationRequest struct {
	PlanID       string            `json:"plan_id" binding:"required"`
	PlanToken    string            `json:"plan_token" binding:"required"`
	Observations map[string]string `json:"observations"`
	Summary      string            `json:"summary"`
	Client       aiClientInfo      `json:"client"`
}

func PostCLIDiagnosticPlanObservations(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	var req cliDiagnosticObservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	planID, err := uuid.Parse(strings.TrimSpace(req.PlanID))
	if err != nil {
		response.BadRequest(c, "plan_id 无效")
		return
	}
	if strings.TrimSpace(req.PlanToken) == "" {
		response.BadRequest(c, "plan_token 不能为空")
		return
	}
	raw, _ := json.Marshal(req.Observations)
	if len(raw) > diagnosticObservationMaxBytes {
		response.BadRequest(c, "诊断证据体积过大")
		return
	}

	var plan models.DiagnosticPlan
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", planID).First(&plan).Error; err != nil {
			return err
		}
		if plan.UserID != ident.UserID {
			return fmt.Errorf("forbidden")
		}
		if plan.CLIBindingID != nil {
			if ident.CLIBindingID == nil || *plan.CLIBindingID != *ident.CLIBindingID {
				return fmt.Errorf("binding")
			}
		}
		if subtle.ConstantTimeCompare([]byte(plan.FingerprintHash), []byte(ident.FingerprintHash)) != 1 {
			return fmt.Errorf("fingerprint")
		}
		if plan.Status != models.DiagnosticPlanStatusPending {
			return fmt.Errorf("used")
		}
		if time.Now().UTC().After(plan.ExpiresAt) {
			_ = tx.Model(&plan).Update("status", models.DiagnosticPlanStatusExpired).Error
			return fmt.Errorf("expired")
		}
		if subtle.ConstantTimeCompare([]byte(plan.PlanTokenHash), []byte(hashSecret(req.PlanToken))) != 1 {
			return fmt.Errorf("token")
		}
		plan.Status = models.DiagnosticPlanStatusObserved
		plan.Observations = models.NewJSONBFromMap(stringMapToAny(req.Observations))
		plan.Summary = limitText(req.Summary, 1200)
		return tx.Model(&plan).Updates(map[string]interface{}{
			"status":       plan.Status,
			"observations": plan.Observations,
			"summary":      plan.Summary,
		}).Error
	})
	if err != nil {
		switch err.Error() {
		case "forbidden", "binding", "fingerprint", "token":
			response.Unauthorized(c, "诊断任务单无效或不属于当前 CLI")
		case "used":
			response.BadRequest(c, "诊断任务单已使用")
		case "expired":
			response.BadRequest(c, "诊断任务单已过期")
		default:
			if err == gorm.ErrRecordNotFound {
				response.NotFound(c, "诊断任务单不存在")
			} else {
				logger.Error("post diagnostic observations: %v", err)
				response.ServerError(c, "保存诊断证据失败")
			}
		}
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return ensureDiagnosticPlanSkillUnlock(tx, &plan)
	}); err != nil {
		logger.Warn("ensure diagnostic plan skill unlock failed plan=%s: %v", plan.ID, err)
	}
	recordAIExecution(ident, "diagnostic_plan_observations", "诊断证据上报: "+plan.Topic, "", "", skillPackForTopic(plan.Topic), models.ExecutionStatusSuccess, plan.Summary, "", nil, req.Client, aiQuotaDecision{PackKey: skillPackForTopic(plan.Topic)})
	response.OK(c, gin.H{"plan_id": plan.ID.String(), "status": models.DiagnosticPlanStatusObserved})
}

func ensureDiagnosticPlanSkillUnlock(tx *gorm.DB, plan *models.DiagnosticPlan) error {
	if tx == nil || plan == nil || plan.UserID == uuid.Nil {
		return nil
	}
	topic := strings.TrimSpace(strings.ToLower(plan.Topic))
	if topic == "" {
		topic = "unknown"
	}
	name := "diagnostic." + sanitizeSkillAssetName(topic) + ".readonly-plan"
	var asset models.SkillAsset
	if err := tx.Where("name = ?", name).First(&asset).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		asset = models.SkillAsset{
			Topic:           topic,
			Name:            name,
			DisplayName:     "只读诊断计划: " + topic,
			Status:          models.SkillAssetStatusReview,
			Source:          "diagnostic_plan",
			CreatedByUserID: &plan.UserID,
			CreatedBy:       plan.Username,
			QualityLabels:   models.NewJSONBFromMap(map[string]interface{}{"review_required": true}),
		}
		if err := tx.Create(&asset).Error; err != nil {
			return err
		}
	}
	content := map[string]interface{}{
		"topic":                topic,
		"mode":                 "readonly_plan",
		"source_plan_id":       plan.ID.String(),
		"source_plan_status":   plan.Status,
		"steps":                json.RawMessage(plan.Steps),
		"observations":         json.RawMessage(plan.Observations),
		"observation_summary":  plan.Summary,
		"requires_super_admin": true,
	}
	raw, _ := json.Marshal(content)
	sum := sha256.Sum256(raw)
	checksum := hex.EncodeToString(sum[:])
	versionName := "v" + time.Now().UTC().Format("20060102150405")
	var version models.SkillAssetVersion
	if err := tx.Where("skill_asset_id = ? AND checksum = ?", asset.ID, checksum).First(&version).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		version = models.SkillAssetVersion{
			SkillAssetID: asset.ID,
			Version:      versionName,
			Status:       models.SkillAssetStatusReview,
			Content:      models.NewJSONBFromMap(content),
			Checksum:     checksum,
			Notes:        "created from CLI readonly diagnostic observations",
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
	}
	if err := tx.Model(&asset).Updates(map[string]interface{}{
		"current_version_id": version.ID,
		"status":             models.SkillAssetStatusReview,
	}).Error; err != nil {
		return err
	}
	versionID := version.ID
	var unlock models.UserSkillUnlock
	if err := tx.Where("user_id = ? AND skill_asset_id = ?", plan.UserID, asset.ID).First(&unlock).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		unlock = models.UserSkillUnlock{
			UserID:              plan.UserID,
			SkillAssetID:        asset.ID,
			SkillAssetVersionID: &versionID,
			Source:              "diagnostic_plan",
		}
		return tx.Create(&unlock).Error
	}
	return tx.Model(&unlock).Updates(map[string]interface{}{
		"skill_asset_version_id": versionID,
		"source":                 "diagnostic_plan",
		"valid_until":            nil,
	}).Error
}

func sanitizeSkillAssetName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}

func stringMapToAny(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func buildReadonlyDiagnosticPlan(topic string, kv map[string]string) ([]diagnosticPlanStep, error) {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "k8s", "kubernetes":
		return buildK8sReadonlyDiagnosticPlan(kv), nil
	default:
		return nil, fmt.Errorf("当前仅支持 k8s 只读诊断任务单")
	}
}

func buildK8sReadonlyDiagnosticPlan(kv map[string]string) []diagnosticPlanStep {
	ns := cleanK8sNameFromMap(kv, "namespace", "default")
	pod := cleanK8sNameFromMap(kv, "pod", "")
	steps := []diagnosticPlanStep{
		{ID: "kubectl_version", Title: "读取 kubectl 客户端版本", Argv: []string{"kubectl", "version", "--client=true", "-o", "yaml"}, TimeoutSeconds: 12, EvidenceKey: "kubectl_version"},
		{ID: "kubectl_context", Title: "读取当前 kubeconfig 上下文", Argv: []string{"kubectl", "config", "current-context"}, TimeoutSeconds: 8, EvidenceKey: "kubectl_config_context"},
		{ID: "kubectl_nodes", Title: "读取节点状态", Argv: []string{"kubectl", "get", "nodes", "-o", "wide"}, TimeoutSeconds: 15, EvidenceKey: "kubectl_nodes"},
		{ID: "kubectl_pods_all", Title: "读取全局 Pod 摘要", Argv: []string{"kubectl", "get", "pods", "-A", "-o", "wide"}, TimeoutSeconds: 20, EvidenceKey: "kubectl_pods_all"},
		{ID: "kubectl_events_recent", Title: "读取最近集群事件", Argv: []string{"kubectl", "get", "events", "-A", "--sort-by=.metadata.creationTimestamp"}, TimeoutSeconds: 20, EvidenceKey: "kubectl_events_recent"},
	}
	if pod != "" && !k8sAnalyzePodFlagIsIssueKeywordServer(pod) {
		steps = append([]diagnosticPlanStep{
			{ID: "kubectl_focus_describe", Title: "读取目标 Pod describe", Argv: []string{"kubectl", "describe", "pod", "-n", ns, pod}, TimeoutSeconds: 35, EvidenceKey: "kubectl_focus_describe"},
			{ID: "kubectl_focus_events", Title: "读取目标 Pod 事件", Argv: []string{"kubectl", "get", "events", "-n", ns, "--field-selector=involvedObject.name=" + pod, "-o", "wide"}, TimeoutSeconds: 18, EvidenceKey: "kubectl_focus_events"},
			{ID: "kubectl_focus_logs_current", Title: "读取目标 Pod 当前日志", Argv: []string{"kubectl", "logs", "-n", ns, pod, "--all-containers=true", "--tail=600"}, TimeoutSeconds: 35, EvidenceKey: "kubectl_focus_logs_current"},
			{ID: "kubectl_focus_logs_previous", Title: "读取目标 Pod previous 日志", Argv: []string{"kubectl", "logs", "-n", ns, pod, "--all-containers=true", "--previous", "--tail=400"}, TimeoutSeconds: 28, EvidenceKey: "kubectl_focus_logs_previous"},
		}, steps...)
	}
	return steps
}

func cleanK8sNameFromMap(kv map[string]string, key, fallback string) string {
	if kv == nil {
		return fallback
	}
	v := strings.TrimSpace(kv[key])
	if v == "" {
		return fallback
	}
	if !k8sSafeNameRe.MatchString(v) {
		return fallback
	}
	return v
}

var k8sSafeNameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,253}$`)

func k8sAnalyzePodFlagIsIssueKeywordServer(pod string) bool {
	switch strings.ToLower(strings.TrimSpace(pod)) {
	case "", "pending", "crashloop", "crashloopbackoff", "instability":
		return true
	default:
		return false
	}
}

func allowedReadonlyDiagnosticCommand(argv []string) bool {
	if len(argv) == 0 || argv[0] != "kubectl" {
		return false
	}
	for _, a := range argv {
		if strings.TrimSpace(a) == "" || strings.ContainsAny(a, ";&|`$<>") {
			return false
		}
	}
	if len(argv) < 2 {
		return false
	}
	switch argv[1] {
	case "version":
		return argsSubset(argv[2:], []string{"--client=true", "-o", "yaml", "json"})
	case "config":
		return len(argv) == 3 && argv[2] == "current-context"
	case "get":
		return allowedKubectlGet(argv[2:])
	case "describe":
		return allowedKubectlDescribe(argv[2:])
	case "logs":
		return allowedKubectlLogs(argv[2:])
	default:
		return false
	}
}

func allowedKubectlGet(args []string) bool {
	if len(args) == 0 {
		return false
	}
	resource := args[0]
	if resource != "nodes" && resource != "pods" && resource != "events" && resource != "pod" {
		return false
	}
	return argsSubset(args[1:], []string{"-A", "--all-namespaces", "-n", "--namespace", "-o", "wide", "json", "yaml", "--sort-by=.metadata.creationTimestamp"}) &&
		allowedK8sFlagValues(args[1:])
}

func allowedKubectlDescribe(args []string) bool {
	if len(args) == 0 || args[0] != "pod" {
		return false
	}
	return argsSubset(args[1:], []string{"-n", "--namespace"}) && allowedK8sFlagValues(args[1:])
}

func allowedKubectlLogs(args []string) bool {
	return argsSubset(args, []string{"-n", "--namespace", "--all-containers=true", "--previous"}) && allowedK8sFlagValues(args)
}

func argsSubset(args []string, allowed []string) bool {
	set := map[string]struct{}{}
	for _, a := range allowed {
		set[a] = struct{}{}
	}
	for _, a := range args {
		if strings.HasPrefix(a, "--field-selector=") || strings.HasPrefix(a, "--tail=") {
			continue
		}
		if strings.HasPrefix(a, "-") {
			if _, ok := set[a]; !ok {
				return false
			}
		}
	}
	return true
}

func allowedK8sFlagValues(args []string) bool {
	expectValue := ""
	for _, a := range args {
		if expectValue != "" {
			if !k8sSafeNameRe.MatchString(a) && a != "wide" && a != "json" && a != "yaml" {
				return false
			}
			expectValue = ""
			continue
		}
		switch a {
		case "-n", "--namespace", "-o":
			expectValue = a
		default:
			if strings.HasPrefix(a, "--field-selector=") {
				v := strings.TrimPrefix(a, "--field-selector=")
				if !strings.HasPrefix(v, "involvedObject.name=") && !strings.HasPrefix(v, "metadata.name=") && !strings.HasPrefix(v, "status.phase=") {
					return false
				}
				if strings.ContainsAny(v, ";&|`$<>") {
					return false
				}
			}
			if strings.HasPrefix(a, "--tail=") && !regexp.MustCompile(`^--tail=[0-9]{1,5}$`).MatchString(a) {
				return false
			}
		}
	}
	return expectValue == ""
}
