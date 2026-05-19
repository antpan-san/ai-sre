package cli

import (
	"encoding/json"
	"os"
	"strings"
)

type checkStructuredResult struct {
	Topic           string
	Target          string
	Status          string
	Severity        string
	RootCause       string
	Evidence        []string
	Impact          string
	Recommendations []string
	UsedAI          bool
	EvidenceLevel   string
	SkillPack       string
	ExecutionID     string
}

func formatCheckStructuredText(r checkStructuredResult) string {
	var b strings.Builder
	if rc := strings.TrimSpace(r.RootCause); rc != "" {
		b.WriteString("【根因结论】\n")
		b.WriteString(rc)
		b.WriteByte('\n')
	}
	if len(r.Evidence) > 0 {
		b.WriteString("\n【关键证据】\n")
		for _, e := range r.Evidence {
			if s := strings.TrimSpace(e); s != "" {
				b.WriteString("- ")
				b.WriteString(s)
				b.WriteByte('\n')
			}
		}
	}
	if imp := strings.TrimSpace(r.Impact); imp != "" {
		b.WriteString("\n【影响判断】\n")
		b.WriteString(imp)
		b.WriteByte('\n')
	}
	if len(r.Recommendations) > 0 {
		b.WriteString("\n【修复建议】\n")
		for _, rec := range r.Recommendations {
			if s := strings.TrimSpace(rec); s != "" {
				b.WriteString("- ")
				b.WriteString(s)
				b.WriteByte('\n')
			}
		}
	}
	b.WriteString("\n【是否调用 AI】\n")
	if r.UsedAI {
		b.WriteString("是\n")
	} else {
		b.WriteString("否\n")
	}
	level := strings.TrimSpace(r.EvidenceLevel)
	if level == "" {
		level = "unknown"
	}
	b.WriteString("\n【证据完整度】\n")
	b.WriteString(level)
	b.WriteString("\n")
	return strings.TrimSpace(b.String())
}

func buildCheckJSONResult(topic, target string, diag *diagnoseResponse, ctx map[string]string, usedAI bool) map[string]interface{} {
	root, evidence, recs := splitAnswerSections(diag)
	ruleHit := diag != nil && strings.EqualFold(strings.TrimSpace(diag.Source), "local-rule")
	out := map[string]interface{}{
		"topic":              normalizeCheckTopicAlias(topic),
		"target":             target,
		"status":             "ok",
		"severity":           inferSeverity(diag),
		"root_cause":         root,
		"evidence":           evidence,
		"recommendations":    recs,
		"used_ai":            usedAI,
		"rule_hit":           ruleHit,
		"ai_source":          aiSourceLabel(diag),
		"evidence_complete":  evidenceCompletenessForContext(ctx),
		"skill_pack":         "",
		"execution_id":       ActiveExecutionRecordID(),
	}
	if diag != nil {
		out["skill_pack"] = diag.SkillName
		if strings.TrimSpace(diag.Answer) != "" && root == "" {
			out["root_cause"] = strings.TrimSpace(diag.Answer)
		}
	}
	return out
}

func printCheckJSONResult(topic, target string, diag *diagnoseResponse, ctx map[string]string, usedAI bool) error {
	used := usedAI
	if diag != nil && strings.EqualFold(strings.TrimSpace(diag.Source), "local-rule") {
		used = false
	}
	payload := buildCheckJSONResult(topic, target, diag, ctx, used)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func splitAnswerSections(diag *diagnoseResponse) (root string, evidence, recs []string) {
	if diag == nil {
		return "", nil, nil
	}
	text := strings.TrimSpace(diag.Answer)
	if text == "" {
		return "", nil, nil
	}
	sections := map[string]*[]string{
		"根因结论": nil,
		"关键证据": &evidence,
		"修复建议": &recs,
	}
	var current *[]string
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "【") && strings.HasSuffix(trim, "】") {
			title := strings.Trim(trim, "【】")
			if title == "根因结论" {
				current = nil
				continue
			}
			if p, ok := sections[title]; ok {
				current = p
				continue
			}
		}
		if current != nil && trim != "" && !strings.HasPrefix(trim, "-") {
			*current = append(*current, strings.TrimPrefix(trim, "- "))
		} else if current != nil && strings.HasPrefix(trim, "-") {
			*current = append(*current, strings.TrimSpace(strings.TrimPrefix(trim, "-")))
		} else if root == "" && trim != "" && current == nil {
			root = trim
		}
	}
	if root == "" {
		root = text
	}
	return root, evidence, recs
}

func inferSeverity(diag *diagnoseResponse) string {
	if diag == nil {
		return "info"
	}
	lower := strings.ToLower(diag.Answer + " " + diag.SkillName)
	for _, s := range []string{"critical", "严重", "fatal", "oom", "crash"} {
		if strings.Contains(lower, s) {
			return "critical"
		}
	}
	for _, s := range []string{"warn", "warning", "高", "拒绝", "失败"} {
		if strings.Contains(lower, s) {
			return "warning"
		}
	}
	return "info"
}
