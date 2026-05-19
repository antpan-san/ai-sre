package handlers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/models"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type aiDiagnoseRequest struct {
	Topic     string                        `json:"topic" binding:"required"`
	Context   map[string]string             `json:"context"`
	Command   string                        `json:"command"`
	RequestID string                        `json:"request_id"`
	Client    aiClientInfo                  `json:"client"`
	Intent    services.SkillExecutionIntent `json:"intent"`
}

type aiSkillPack struct {
	Name           string   `json:"name" yaml:"name"`
	DisplayName    string   `json:"display_name" yaml:"display_name"`
	Topics         []string `json:"topics" yaml:"topics"`
	MatchKeywords  []string `json:"match_keywords" yaml:"match_keywords"`
	Input          []string `json:"input" yaml:"input"`
	AnalysisSteps  []string `json:"analysis_steps" yaml:"analysis_steps"`
	OutputFormat   []string `json:"output_format" yaml:"output_format"`
	ExtraGuidance  string   `json:"extra_guidance,omitempty" yaml:"extra_guidance,omitempty"`
	PromptTemplate string   `json:"prompt_template,omitempty" yaml:"prompt_template,omitempty"`
}

type aiDiagnoseResponse struct {
	Source       string                 `json:"source"`
	Answer       string                 `json:"answer"`
	SkillName    string                 `json:"skill_name,omitempty"`
	SkillDisplay string                 `json:"skill_display,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SkillDraft   *aiSkillPack           `json:"skill_draft,omitempty"`
}

type aiEvolveRequest struct {
	Topic         string            `json:"topic" binding:"required"`
	Context       map[string]string `json:"context"`
	Answer        string            `json:"answer"`
	Feedback      string            `json:"feedback"`
	ExistingSkill string            `json:"existing_skill"`
}

// AIDiagnose runs server-side AI diagnosis and optionally returns a skill draft.
func AIDiagnose(c *gin.Context) {
	var req aiDiagnoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	topic := strings.TrimSpace(strings.ToLower(req.Topic))
	if topic == "" {
		response.BadRequest(c, "topic 不能为空")
		return
	}
	ident, ok := resolveAIIdentity(c)
	if !ok {
		return
	}
	intent := services.NormalizeSkillExecutionIntent(topic, req.Context, req.Intent)
	packKey := defaultString(intent.PackKey, skillPackForTopic(topic))
	commitQuota, quotaDecision, quotaOK := beginAIQuotaForIdentity(c, packKey, ident)
	if !quotaOK {
		recordAIExecution(ident, "analyze", "AI 诊断: "+topic, defaultString(req.Command, "ai-sre analyze "+topic), req.RequestID, packKey, models.ExecutionStatusFailed, "", "ai_free_quota_exhausted", req.Context, req.Client, quotaDecision)
		return
	}
	reg := services.DefaultSkillRegistry()
	matched := reg.Match(topic, req.Context)
	if ident != nil && ident.UserID != uuid.Nil {
		if overlay := services.UserDiagnosticSkillOverlay(ident.UserID, topic, intent.ProblemKey); overlay != nil {
			matched = services.MergeRegisteredSkills(matched, overlay)
		}
	}

	prompt := buildServerDiagnosePromptWithSkill(topic, req.Context, matched)
	answer, err := runServerDeepSeek(c.Request.Context(), prompt)
	if err != nil {
		logger.Error("AIDiagnose deepseek failed: %v", err)
		recordAIExecution(ident, "analyze", "AI 诊断: "+topic, defaultString(req.Command, "ai-sre analyze "+topic), req.RequestID, packKey, models.ExecutionStatusFailed, "", err.Error(), req.Context, req.Client, quotaDecision)
		response.ServerError(c, "服务端 AI 诊断失败: "+err.Error())
		return
	}
	commitQuota(true)

	skillName := ""
	skillDisplay := ""
	skillSource := ""
	if matched != nil {
		skillName = matched.Pack.Name
		skillDisplay = matched.Pack.DisplayName
		skillSource = string(matched.Source)
	} else {
		skillName = skillNameForTopic(topic)
		skillDisplay = "Auto evolved " + strings.ToUpper(topic) + " skill"
	}
	reqID := requestIDOrNow(req.RequestID)
	meta := map[string]interface{}{
		"request_id":           reqID,
		"topic":                topic,
		"fallback":             "server_deepseek",
		"skill_source":         skillSource,
		"normalized_node_path": intent.NodePath,
		"skill_key":            intent.SkillKey,
		"problem_key":          intent.ProblemKey,
		"capability_key":       intent.CapabilityKey,
		"execution_mode":       intent.ExecutionMode,
		"pack_key":             packKey,
	}
	if req.Context != nil {
		if s := strings.TrimSpace(req.Context["diagnosis_style"]); s != "" {
			meta["diagnosis_style"] = s
		}
	}
	recordAIExecution(ident, "analyze", "AI 诊断: "+topic, defaultString(req.Command, "ai-sre analyze "+topic), reqID, packKey, models.ExecutionStatusSuccess, answer, "", req.Context, req.Client, quotaDecision)

	// Fire-and-forget sample logging for self-iteration. Never block the response.
	go func(topic, name, requestID, answer string, ctxKV map[string]string) {
		defer func() { _ = recover() }()
		sample := services.DiagnoseSample{
			Topic:       topic,
			SkillName:   name,
			Style:       strings.TrimSpace(ctxKV["diagnosis_style"]),
			UserContext: stripBulkEvidenceForSample(ctxKV),
			EvidenceKey: evidenceKeyList(ctxKV),
			AnswerLen:   len(answer),
			AnswerHead:  headSample(answer, 600),
			AnswerTail:  tailSample(answer, 400),
			RequestID:   requestID,
		}
		if err := services.AppendDiagnoseSample(services.DefaultSkillRegistry(), sample); err != nil {
			logger.Error("AppendDiagnoseSample topic=%s failed: %v", topic, err)
		}
	}(topic, skillName, reqID, answer, req.Context)

	response.OK(c, aiDiagnoseResponse{
		Source:       "server-ai",
		Answer:       answer,
		SkillName:    skillName,
		SkillDisplay: skillDisplay,
		Metadata:     meta,
	})
}

// AISkillsEvolve generates skill draft from diagnosis sample.
func AISkillsEvolve(c *gin.Context) {
	var req aiEvolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	topic := strings.TrimSpace(strings.ToLower(req.Topic))
	if topic == "" {
		response.BadRequest(c, "topic 不能为空")
		return
	}
	draft := buildSkillDraftFromContext(topic, req.Context)
	if !isSkillDraftValid(draft) {
		response.ServerError(c, "技能草案生成失败：结果不符合最小 schema")
		return
	}
	response.OK(c, gin.H{
		"draft": draft,
		"metadata": gin.H{
			"topic":          topic,
			"existing_skill": req.ExistingSkill,
			"feedback":       strings.TrimSpace(req.Feedback),
		},
	})
}

type aiAskRequest struct {
	Question  string       `json:"question" binding:"required"`
	NoRAG     bool         `json:"no_rag"`
	Command   string       `json:"command"`
	RequestID string       `json:"request_id"`
	Client    aiClientInfo `json:"client"`
}

type aiRunbookRequest struct {
	Scenario  string            `json:"scenario" binding:"required"`
	Context   map[string]string `json:"context"`
	Command   string            `json:"command"`
	RequestID string            `json:"request_id"`
	Client    aiClientInfo      `json:"client"`
}

// AIAsk runs server-side Q&A for ai-sre `ask` when the client has no local LLM key.
func AIAsk(c *gin.Context) {
	var req aiAskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	ident, ok := resolveAIIdentity(c)
	if !ok {
		return
	}
	packKey := skillPackForText(req.Question)
	commitQuota, quotaDecision, quotaOK := beginAIQuotaForIdentity(c, packKey, ident)
	if !quotaOK {
		recordAIExecution(ident, "ask", "AI 问答", defaultString(req.Command, "ai-sre ask"), req.RequestID, packKey, models.ExecutionStatusFailed, "", "ai_free_quota_exhausted", map[string]string{"question": req.Question}, req.Client, quotaDecision)
		return
	}
	prompt := buildServerAskPrompt(req.Question, req.NoRAG)
	answer, err := runServerDeepSeek(c.Request.Context(), prompt)
	if err != nil {
		logger.Error("AIAsk deepseek failed: %v", err)
		recordAIExecution(ident, "ask", "AI 问答", defaultString(req.Command, "ai-sre ask"), req.RequestID, packKey, models.ExecutionStatusFailed, "", err.Error(), map[string]string{"question": req.Question}, req.Client, quotaDecision)
		response.ServerError(c, "服务端 AI 失败: "+err.Error())
		return
	}
	commitQuota(true)
	recordAIExecution(ident, "ask", "AI 问答", defaultString(req.Command, "ai-sre ask"), req.RequestID, packKey, models.ExecutionStatusSuccess, answer, "", map[string]string{"question": req.Question}, req.Client, quotaDecision)
	response.OK(c, gin.H{
		"answer": answer,
		"source": "server-ai",
	})
}

// AIRunbook runs server-side runbook generation for ai-sre `runbook`.
func AIRunbook(c *gin.Context) {
	var req aiRunbookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	ident, ok := resolveAIIdentity(c)
	if !ok {
		return
	}
	packKey := skillPackForText(req.Scenario)
	commitQuota, quotaDecision, quotaOK := beginAIQuotaForIdentity(c, packKey, ident)
	if !quotaOK {
		recordAIExecution(ident, "runbook", "AI Runbook", defaultString(req.Command, "ai-sre runbook"), req.RequestID, packKey, models.ExecutionStatusFailed, "", "ai_free_quota_exhausted", req.Context, req.Client, quotaDecision)
		return
	}
	prompt := buildServerRunbookPrompt(req.Scenario, req.Context)
	answer, err := runServerDeepSeek(c.Request.Context(), prompt)
	if err != nil {
		logger.Error("AIRunbook deepseek failed: %v", err)
		recordAIExecution(ident, "runbook", "AI Runbook", defaultString(req.Command, "ai-sre runbook"), req.RequestID, packKey, models.ExecutionStatusFailed, "", err.Error(), req.Context, req.Client, quotaDecision)
		response.ServerError(c, "服务端 AI 失败: "+err.Error())
		return
	}
	commitQuota(true)
	recordAIExecution(ident, "runbook", "AI Runbook", defaultString(req.Command, "ai-sre runbook"), req.RequestID, packKey, models.ExecutionStatusSuccess, answer, "", req.Context, req.Client, quotaDecision)
	response.OK(c, gin.H{
		"answer": answer,
		"source": "server-ai",
	})
}

const aiFreeDailyLimit = 5

func skillPackForTopic(topic string) string {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "k8s", "kubernetes":
		return models.SkillPackK8s
	case "kafka":
		return models.SkillPackKafka
	case "redis":
		return models.SkillPackRedis
	case "nginx":
		return models.SkillPackNginx
	case "mysql":
		return models.SkillPackMySQL
	case "postgresql", "postgres":
		return models.SkillPackPostgreSQL
	case "elasticsearch", "es":
		return models.SkillPackElasticsearch
	case "domain", "dns":
		return models.SkillPackDomain
	case "go_runtime", "go-runtime", "pod-go":
		return models.PackKeyRuntimeObserve
	default:
		return models.SkillPackK8s
	}
}

func skillPackForText(text string) string {
	s := strings.ToLower(text)
	switch {
	case strings.Contains(s, "kafka"):
		return models.SkillPackKafka
	case strings.Contains(s, "redis"):
		return models.SkillPackRedis
	case strings.Contains(s, "nginx"):
		return models.SkillPackNginx
	case strings.Contains(s, "mysql"):
		return models.SkillPackMySQL
	case strings.Contains(s, "postgres"):
		return models.SkillPackPostgreSQL
	case strings.Contains(s, "elastic") || strings.Contains(s, "es "):
		return models.SkillPackElasticsearch
	case strings.Contains(s, "go runtime") || strings.Contains(s, "go_runtime") || strings.Contains(s, "goroutine") || strings.Contains(s, "pprof"):
		return models.PackKeyRuntimeObserve
	default:
		return models.SkillPackK8s
	}
}

func aiQuotaDate() string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*3600)
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func buildServerAskPrompt(question string, noRAG bool) string {
	var b strings.Builder
	b.WriteString("你是资深 SRE 顾问，用中文回答用户问题。\n")
	b.WriteString("要求：结论先行；给出可执行、可验证的步骤；不要编造具体集群输出。\n")
	if noRAG {
		b.WriteString("（本请求关闭了知识库扩展，仅基于通用经验回答。）\n")
	}
	b.WriteString("\n用户问题：\n")
	b.WriteString(strings.TrimSpace(question))
	b.WriteString("\n")
	return b.String()
}

func buildServerRunbookPrompt(scenario string, kv map[string]string) string {
	var b strings.Builder
	b.WriteString("你是资深 SRE，请用中文输出一份可执行的 Runbook（Markdown 小节结构）。\n")
	b.WriteString("要求：现象确认 → 影响评估 → 应急止血 → 根因排查 → 根治与预防；每步附验证命令占位。\n\n")
	b.WriteString("场景：\n")
	b.WriteString(strings.TrimSpace(scenario))
	b.WriteString("\n")
	if len(kv) > 0 {
		keys := make([]string, 0, len(kv))
		for k := range kv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("\n附加上下文：\n")
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("- %s=%s\n", k, kv[k]))
		}
	}
	return b.String()
}

func kvForSkillDraft(kv map[string]string) map[string]string {
	if kv == nil {
		return nil
	}
	out := make(map[string]string, len(kv))
	for k, v := range kv {
		if strings.HasPrefix(k, "kubectl_") || strings.HasPrefix(k, "host_") {
			continue
		}
		if k == "diagnosis_style" || k == "prior_answer_round1" || k == "refinement_pass" {
			continue
		}
		out[k] = v
	}
	return out
}

func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortedEvidenceKeysForPrompt lists collected evidence keys; kubectl_focus_* and *_diagnose_json first.
func sortedEvidenceKeysForPrompt(evidence map[string]string) []string {
	if len(evidence) == 0 {
		return nil
	}
	var focus, jsonFirst, rest []string
	for k := range evidence {
		switch {
		case strings.HasPrefix(k, "kubectl_focus_"):
			focus = append(focus, k)
		case strings.HasSuffix(k, "_diagnose_json") || strings.HasSuffix(k, "_probe_json"):
			jsonFirst = append(jsonFirst, k)
		default:
			rest = append(rest, k)
		}
	}
	sort.Strings(focus)
	sort.Strings(jsonFirst)
	sort.Strings(rest)
	out := make([]string, 0, len(evidence))
	out = append(out, focus...)
	out = append(out, jsonFirst...)
	out = append(out, rest...)
	return out
}

func isCollectedEvidenceKey(k string) bool {
	switch {
	case strings.HasPrefix(k, "kubectl_"), strings.HasPrefix(k, "host_"),
		strings.HasPrefix(k, "redis_"), strings.HasPrefix(k, "kafka_"),
		strings.HasPrefix(k, "mysql_"), strings.HasPrefix(k, "postgresql_"),
		strings.HasPrefix(k, "nginx_"), strings.HasPrefix(k, "es_"),
		strings.HasPrefix(k, "elasticsearch_"), strings.HasPrefix(k, "domain_"):
		return true
	default:
		return false
	}
}

// buildServerDiagnosePrompt is kept for backwards compatibility (tests use it).
func buildServerDiagnosePrompt(topic string, kv map[string]string) string {
	return buildServerDiagnosePromptWithSkill(topic, kv, nil)
}

func buildServerDiagnosePromptWithSkill(topic string, kv map[string]string, matched *services.RegisteredSkill) string {
	style := ""
	if kv != nil {
		style = strings.TrimSpace(kv["diagnosis_style"])
	}
	switch style {
	case "evidence_root_cause":
		return buildEvidenceRootCausePromptWithSkill(topic, kv, false, matched)
	case "evidence_root_cause_refine":
		return buildEvidenceRootCausePromptWithSkill(topic, kv, true, matched)
	case "domain_connectivity":
		return buildDomainConnectivityPromptWithSkill(topic, kv, matched)
	default:
		return buildDefaultServerDiagnosePromptWithSkill(topic, kv, matched)
	}
}

func buildDomainConnectivityPromptWithSkill(topic string, kv map[string]string, matched *services.RegisteredSkill) string {
	var b strings.Builder
	probe := ""
	domain := ""
	if kv != nil {
		probe = strings.TrimSpace(kv["domain_probe_text"])
		domain = strings.TrimSpace(kv["domain"])
	}
	b.WriteString("你是资深网络/SRE，对公网或内网域名做连通性与证书诊断。\n\n")
	b.WriteString("硬性要求：\n")
	b.WriteString("1) **禁止**编造 Kubernetes、Ingress、Pod、kubectl 等内容；本任务不是 K8s 集群诊断。\n")
	b.WriteString("2) 必须基于下方「域名采集报告」中的 DNS / HTTP / TLS 原文；不得虚构未出现的 IP、端口或错误。\n")
	b.WriteString("3) 输出为**纯文本**（不要用 Markdown # 标题），固定小节名：\n")
	b.WriteString("【结论】一句话说明访问是否正常、主要问题在哪一层（DNS / HTTP / HTTPS / 证书）。\n")
	b.WriteString("【DNS】列出解析到的记录及含义。\n")
	b.WriteString("【HTTP】分别说明 http 与 https（若有）的状态、延迟、Server 头、重定向。\n")
	b.WriteString("【TLS】证书有效期、SAN、错误或 443 不可达原因。\n")
	b.WriteString("【建议】面向运维的修复方向（如开放 443、配置 TLS、修正 DNS），避免 kubectl/集群术语。\n")
	if matched != nil {
		b.WriteString("\n")
		writeSkillSection(&b, matched)
	}
	b.WriteString("\ntopic=" + topic)
	if domain != "" {
		b.WriteString(" domain=" + domain)
	}
	b.WriteString("\n\n")
	if probe != "" {
		b.WriteString("=== 域名采集报告（原文） ===\n")
		b.WriteString(probe)
		b.WriteString("\n")
	} else if kv != nil {
		b.WriteString("（缺少 domain_probe_text；请根据 context 中的 domain_probe_json 分析。）\n")
		for _, k := range sortedStringKeys(kv) {
			if k == "domain_probe_json" || strings.HasPrefix(k, "domain_") {
				b.WriteString(fmt.Sprintf("- %s=%s\n", k, kv[k]))
			}
		}
	}
	return b.String()
}

func buildDefaultServerDiagnosePromptWithSkill(topic string, kv map[string]string, matched *services.RegisteredSkill) string {
	var b strings.Builder
	b.WriteString("你是资深SRE，请输出可执行的中文诊断。\n")
	b.WriteString("要求：1) 先结论 2) 最可能原因排序 3) 基于已有 context 的验证要点（禁止让用户执行 probe/redis-cli/kubectl 补采集）4) 临时缓解与根治建议。\n")
	b.WriteString("topic=" + topic + "\n")
	writeSkillSection(&b, matched)
	if len(kv) > 0 {
		for _, k := range sortedStringKeys(kv) {
			b.WriteString(fmt.Sprintf("- %s=%s\n", k, kv[k]))
		}
	}
	return b.String()
}

func writeSkillSection(b *strings.Builder, matched *services.RegisteredSkill) {
	if matched == nil {
		return
	}
	pack := matched.Pack
	b.WriteString("\n【适用技能包】 ")
	b.WriteString(pack.Name)
	if pack.DisplayName != "" {
		b.WriteString(" — ")
		b.WriteString(pack.DisplayName)
	}
	b.WriteString(" (source=")
	b.WriteString(string(matched.Source))
	b.WriteString(")\n")
	if len(pack.AnalysisSteps) > 0 {
		b.WriteString("分析步骤（必须按顺序覆盖）：\n")
		for i, s := range pack.AnalysisSteps {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, s))
		}
	}
	if len(pack.OutputFormat) > 0 {
		b.WriteString("输出结构（必须使用以下小节标题作为 Markdown H2）：\n")
		for _, s := range pack.OutputFormat {
			b.WriteString("  - ## ")
			b.WriteString(s)
			b.WriteString("\n")
		}
	}
	if strings.TrimSpace(pack.ExtraGuidance) != "" {
		b.WriteString("额外约束：\n")
		b.WriteString(pack.ExtraGuidance)
		b.WriteString("\n")
	}
}

// buildEvidenceRootCausePrompt is kept for backwards compatibility (tests use it).
func buildEvidenceRootCausePrompt(topic string, kv map[string]string, refine bool) string {
	return buildEvidenceRootCausePromptWithSkill(topic, kv, refine, nil)
}

func buildEvidenceRootCausePromptWithSkill(topic string, kv map[string]string, refine bool, matched *services.RegisteredSkill) string {
	var b strings.Builder
	prior := ""
	user := map[string]string{}
	evidence := map[string]string{}
	if kv != nil {
		prior = strings.TrimSpace(kv["prior_answer_round1"])
		for k, v := range kv {
			switch {
			case k == "diagnosis_style":
				continue
			case k == "prior_answer_round1":
				continue
			case isCollectedEvidenceKey(k):
				evidence[k] = v
			default:
				user[k] = v
			}
		}
	}
	hasFocusEvidence := false
	for k := range evidence {
		if strings.HasPrefix(k, "kubectl_focus_") {
			hasFocusEvidence = true
			break
		}
	}
	b.WriteString("你是资深 SRE。下方「只读采集输出」由 ai-sre 在客户环境自动执行 probe/check 采集（kubectl、Redis INFO、Kafka 等），是**真实观测事实**。\n\n")
	if refine {
		b.WriteString("【第二轮：精炼】下面给出第一轮模型回答。你必须自检：若第一轮未引用「只读采集输出」中的**原文字句**作为证据，则完全重写结论；若已充分引用，则把根因写得更具体，并删除泛泛的排查教程。\n\n")
		b.WriteString("=== 第一轮模型回答（对照用） ===\n")
		b.WriteString(prior)
		b.WriteString("\n\n")
	}
	b.WriteString("硬性要求：\n")
	b.WriteString("1) **根因**必须完全可从「只读采集输出」中推得；禁止凭空虚构未出现的指标、日志或错误码。\n")
	b.WriteString("2) **禁止**要求用户再去执行 redis-cli / probe / kubectl / shell 等命令补采集；若证据不足，只能说明因认证失败、ACL 拒绝或网络不可达导致哪些证据缺失。\n")
	b.WriteString("3) **禁止**输出「你可以执行 ai-sre probe …」类提示；采集已由 CLI 完成。\n")
	b.WriteString("4) 输出必须是 Markdown，且**仅**包含以下小节（标题固定）：\n")
	b.WriteString("## 根因（一句话）\n")
	b.WriteString("## 关键证据（逐条用代码块或引号摘录采集原文中的关键行）\n")
	b.WriteString("## 修复要点（面向目标状态、组件与配置项，避免命令堆砌）\n")
	b.WriteString("5) 若采集输出不足以定论，在「根因」中明确写「信息不足：缺少 xxx」，不要编造。\n")
	if hasFocusEvidence {
		b.WriteString("6) 若存在 `kubectl_focus_*` 小节，根因与证据必须**优先**结合该 Pod 的 describe/events/logs，再结合集群全景采集。\n")
	}
	if matched != nil {
		b.WriteString("\n")
		writeSkillSection(&b, matched)
	}
	b.WriteString("\ntopic=" + topic + "\n\n")
	if len(user) > 0 {
		b.WriteString("用户通过 ai-sre 传入的标志上下文：\n")
		for _, k := range sortedStringKeys(user) {
			b.WriteString(fmt.Sprintf("- %s=%s\n", k, user[k]))
		}
		b.WriteString("\n")
	}
	if len(evidence) == 0 {
		b.WriteString("（未附带只读采集输出；仍按上述格式回答，并在根因中说明缺少采集数据。）\n")
		return b.String()
	}
	b.WriteString("## 只读采集输出（原文）\n")
	for _, k := range sortedEvidenceKeysForPrompt(evidence) {
		b.WriteString(fmt.Sprintf("\n### %s\n```text\n%s\n```\n", k, evidence[k]))
	}
	return b.String()
}

func runServerDeepSeek(ctx context.Context, userPrompt string) (string, error) {
	cfg := services.LoadServerAIConfig()
	return services.DiagnoseWithDeepSeek(ctx, cfg, userPrompt)
}

func buildSkillDraftFromContext(topic string, kv map[string]string) *aiSkillPack {
	draft := services.BuildSkillDraft(topic, kv)
	if draft == nil {
		return nil
	}
	return &aiSkillPack{
		Name:           draft.Name,
		DisplayName:    draft.DisplayName,
		Topics:         draft.Topics,
		MatchKeywords:  draft.MatchKeywords,
		Input:          draft.Input,
		AnalysisSteps:  draft.AnalysisSteps,
		OutputFormat:   draft.OutputFormat,
		ExtraGuidance:  draft.ExtraGuidance,
		PromptTemplate: draft.PromptTemplate,
	}
}

func isSkillDraftValid(p *aiSkillPack) bool {
	return services.ValidateSkillDraft(&services.SkillPack{
		Name:           p.Name,
		DisplayName:    p.DisplayName,
		Topics:         p.Topics,
		MatchKeywords:  p.MatchKeywords,
		Input:          p.Input,
		AnalysisSteps:  p.AnalysisSteps,
		OutputFormat:   p.OutputFormat,
		ExtraGuidance:  p.ExtraGuidance,
		PromptTemplate: p.PromptTemplate,
	})
}

func skillNameForTopic(topic string) string {
	return services.SkillNameForTopic(topic)
}

func requestIDOrNow(id string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
