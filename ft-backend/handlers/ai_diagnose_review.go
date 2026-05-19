package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var markdownH2 = regexp.MustCompile(`(?m)^#{1,3}\s+(.+?)\s*$`)

// finalizeDiagnoseAnswer runs heuristic checks and optional AI review, then normalizes to plain text.
func finalizeDiagnoseAnswer(ctx context.Context, topic string, kv map[string]string, draft string) string {
	topic = strings.ToLower(strings.TrimSpace(topic))
	issues := heuristicDiagnoseIssues(topic, kv, draft)
	out := draft
	if len(issues) > 0 {
		reviewPrompt := buildDiagnoseReviewPrompt(topic, kv, draft, issues)
		if reviewed, err := runServerDeepSeek(ctx, reviewPrompt); err == nil && strings.TrimSpace(reviewed) != "" {
			out = reviewed
		}
	}
	return normalizeDiagnosePlainText(out)
}

func needsDiagnoseReview(topic string, kv map[string]string) bool {
	if kv == nil {
		return false
	}
	style := strings.TrimSpace(kv["diagnosis_style"])
	if style == "middleware_evidence" || style == "linux_performance_evidence" || style == "evidence_root_cause" {
		return isMiddlewareEvidenceTopic(topic) || isLinuxPerformanceTopic(topic) || strings.HasPrefix(style, "evidence")
	}
	if isLinuxPerformanceTopic(topic) {
		return strings.TrimSpace(kv["linux_perf_probe_json"]) != ""
	}
	return isMiddlewareEvidenceTopic(topic) && strings.TrimSpace(kv["redis_diagnose_json"]) != ""
}

func isMiddlewareEvidenceTopic(topic string) bool {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "redis", "kafka", "mysql", "postgresql", "postgres", "nginx", "elasticsearch", "es":
		return true
	default:
		return false
	}
}

func heuristicDiagnoseIssues(topic string, kv map[string]string, answer string) []string {
	if kv == nil || strings.TrimSpace(answer) == "" {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "redis":
		return heuristicRedisDiagnoseIssues(kv, answer)
	default:
		return nil
	}
}

func heuristicRedisDiagnoseIssues(kv map[string]string, answer string) []string {
	raw := strings.TrimSpace(kv["redis_diagnose_json"])
	if raw == "" {
		return nil
	}
	var probe struct {
		Memory  map[string]any `json:"memory"`
		Clients map[string]any `json:"clients"`
		Stats   map[string]any `json:"stats"`
		Errors  []string       `json:"errors"`
	}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil
	}
	rejected := parseJSONNumber(probe.Clients["rejected_connections"])
	usedMem := parseJSONNumber(probe.Memory["used_memory"])
	frag := parseJSONFloat(probe.Memory["mem_fragmentation_ratio"])
	lower := strings.ToLower(answer)
	var issues []string
	if rejected > 0 && rejected <= 20 && (strings.Contains(lower, "碎片") || strings.Contains(lower, "fragmentation")) &&
		usedMem > 0 && usedMem < 50*1024*1024 && frag > 5 {
		issues = append(issues, fmt.Sprintf(
			"证据显示 rejected_connections=%v、used_memory=%v 字节、mem_fragmentation_ratio=%.2f；低占用场景下不宜将高碎片率单独定性为 rejected_connections 的主因，应优先说明拒绝连接更可能来自 bind/backlog/保护模式或瞬时峰值，碎片仅作次要风险。",
			rejected, usedMem, frag))
	}
	if rejected <= 5 && (strings.Contains(lower, "立即重启") || strings.Contains(lower, "必须重启")) {
		issues = append(issues, "rejected_connections 次数很少时不应建议立即重启作为首选手段，应给出更保守的验证与观察建议。")
	}
	if strings.Contains(lower, "redis-cli") || strings.Contains(lower, "ai-sre probe") || strings.Contains(lower, "请执行") && strings.Contains(lower, "采集") {
		issues = append(issues, "禁止要求用户执行 redis-cli/probe 等补采集命令。")
	}
	return issues
}

func buildDiagnoseReviewPrompt(topic string, kv map[string]string, draft string, issues []string) string {
	var b strings.Builder
	b.WriteString("你是 SRE 诊断复核员。下面初稿可能逻辑过度推断，请基于「只读采集 JSON」修订后输出终稿。\n\n")
	b.WriteString("复核问题：\n")
	for i, iss := range issues {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, iss))
	}
	b.WriteString("\n硬性要求：\n")
	b.WriteString("1) 仅修正逻辑与因果，不新增采集中不存在的指标。\n")
	b.WriteString("2) 输出为**纯文本**，小节标题固定且使用中文方括号：\n")
	b.WriteString("【根因与触发条件】\n【关键指标证据】\n【缓解与根治建议】\n")
	b.WriteString("3) 禁止使用 Markdown #/##、禁止 ** 加粗、禁止代码块。\n")
	b.WriteString("4) 禁止让用户执行 redis-cli/probe/kubectl。\n\n")
	b.WriteString("topic=" + topic + "\n\n")
	if raw := strings.TrimSpace(kv["redis_diagnose_json"]); raw != "" && len(raw) < 12000 {
		b.WriteString("=== redis_diagnose_json ===\n")
		b.WriteString(raw)
		b.WriteString("\n\n")
	}
	b.WriteString("=== 初稿 ===\n")
	b.WriteString(draft)
	b.WriteString("\n")
	return b.String()
}

func normalizeDiagnosePlainText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if m := markdownH2.FindStringSubmatch(trim); len(m) == 2 {
			title := strings.TrimSpace(m[1])
			title = strings.TrimSuffix(title, "（一句话）")
			out = append(out, "【"+title+"】")
			continue
		}
		if strings.HasPrefix(trim, "```") {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func parseJSONNumber(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}

func parseJSONFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	default:
		return 0
	}
}
