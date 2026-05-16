package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	goruntime "github.com/panshuai/ai-sre/internal/go_runtime"
)

var (
	reRootCauseLine = regexp.MustCompile(`(?im)^\s*根因\s*[：:]\s*(.+)\s*$`)
	reEvidenceLine  = regexp.MustCompile(`(?im)^\s*证据\s*[：:]\s*(.+)\s*$`)
)

// shouldDeferToPlatformAI: no proc samples but kubectl probes exist — prefer integrated AI root cause.
func shouldDeferToPlatformAI(wr *goruntime.WatchReport) bool {
	if wr == nil || wr.SampleCount > 0 {
		return false
	}
	if wr.Target.Source != "kubernetes" {
		return false
	}
	return len(wr.ProbeBundle) > 0
}

func finalizeGoRuntimeDiagnosis(ctx context.Context, apiBase string, wr *goruntime.WatchReport) error {
	if wr == nil {
		return fmt.Errorf("无诊断数据")
	}
	goruntime.StripFindingHints(wr)

	if s, ok := goruntime.DeriveLocalDiagnosis(wr); ok && !shouldDeferToPlatformAI(wr) {
		goruntime.ApplyDiagnosis(wr, s.Level, s.Title, s.Evidence, "local")
		return nil
	}

	if strings.TrimSpace(apiBase) == "" {
		if err := finalizeGoRuntimeWithLocalAI(ctx, wr); err == nil {
			return nil
		}
		return fallbackDiagnosisFromSummary(wr)
	}

	topic := "go_runtime"
	kv := buildGoRuntimeAIContext(wr)
	intent := buildExecutionIntent("diagnose", topic, kv)
	resp, err := callServerDiagnose(ctx, diagnoseRequest{
		Topic:     topic,
		Context:   kv,
		Command:   strings.Join(os.Args, " "),
		RequestID: uuid.NewString(),
		Client:    opsfleetAIClient(),
		Intent:    intent,
	})
	if err != nil || resp == nil || strings.TrimSpace(resp.Answer) == "" {
		if err != nil {
			wr.Errors = append(wr.Errors, "平台 AI 分析失败: "+err.Error())
		}
		if localErr := finalizeGoRuntimeWithLocalAI(ctx, wr); localErr == nil {
			return nil
		}
		return fallbackDiagnosisFromSummary(wr)
	}
	rc, ev := parseAIDiagnosis(resp.Answer)
	if rc == "" {
		rc = strings.TrimSpace(resp.Answer)
	}
	if ev == "" {
		ev = probeEvidenceFallback(wr)
	}
	goruntime.ApplyDiagnosis(wr, "WARN", rc, ev, "ai")
	return nil
}

func finalizeGoRuntimeWithLocalAI(ctx context.Context, wr *goruntime.WatchReport) error {
	eng, err := bootstrap()
	if err != nil {
		return err
	}
	res, err := eng.Analyze(ctx, "go_runtime", buildGoRuntimeAIContext(wr), !noRAG)
	if err != nil || res == nil || strings.TrimSpace(res.Answer) == "" {
		if err == nil {
			err = fmt.Errorf("本地 AI 未返回诊断")
		}
		return err
	}
	rc, ev := parseAIDiagnosis(res.Answer)
	if rc == "" {
		rc = strings.TrimSpace(res.Answer)
	}
	if ev == "" {
		ev = probeEvidenceFallback(wr)
	}
	goruntime.ApplyDiagnosis(wr, "WARN", rc, ev, "local_ai")
	return nil
}

func fallbackDiagnosisFromSummary(wr *goruntime.WatchReport) error {
	if wr == nil {
		return fmt.Errorf("无法得出根因")
	}
	if strings.TrimSpace(wr.Summary.Title) != "" {
		goruntime.ApplyDiagnosis(wr, wr.Summary.Level, wr.Summary.Title, wr.Summary.Evidence, "local")
		return nil
	}
	top := topFindingFromWatch(wr)
	if top != nil {
		cause := strings.TrimSpace(top.Cause)
		if cause == "" {
			cause = top.Title
		}
		goruntime.ApplyDiagnosis(wr, "WARN", cause, top.Evidence, "local")
		return nil
	}
	return fmt.Errorf("无法得出根因")
}

func topFindingFromWatch(wr *goruntime.WatchReport) *goruntime.Finding {
	if wr == nil || len(wr.TrendFindings) == 0 {
		return nil
	}
	best := wr.TrendFindings[0]
	for i := range wr.TrendFindings {
		f := wr.TrendFindings[i]
		if severityRank(f.Severity) > severityRank(best.Severity) {
			best = f
		}
	}
	return &best
}

func severityRank(sev string) int {
	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "critical", "crit":
		return 3
	case "warn", "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func buildGoRuntimeAIContext(wr *goruntime.WatchReport) map[string]string {
	kv := map[string]string{
		"diagnosis_style": "evidence_root_cause",
		"record_kind":     "go_runtime",
		"issue":           "go_runtime_observe",
	}
	if wr != nil {
		if ns := strings.TrimSpace(wr.Target.Namespace); ns != "" {
			kv["namespace"] = ns
		}
		if pod := strings.TrimSpace(wr.Target.Pod); pod != "" {
			kv["pod"] = pod
		}
		if rk := strings.TrimSpace(wr.Target.ResourceKind); rk != "" {
			kv["resource_kind"] = rk
			kv[rk] = strings.TrimSpace(wr.Target.ResourceName)
		}
		if rn := strings.TrimSpace(wr.Target.ResourceName); rn != "" {
			kv["resource_name"] = rn
		}
		if tgt := strings.TrimSpace(wr.Target.Target); tgt != "" {
			kv["diagnose_target"] = tgt
		}
		for k, v := range wr.ProbeBundle {
			kv[k] = v
		}
		raw, err := json.Marshal(wr)
		if err == nil {
			kv["go_runtime_watch_json"] = truncateBytes(string(raw), 60_000)
		}
	}
	return kv
}

func parseAIDiagnosis(answer string) (rootCause, evidence string) {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", ""
	}
	if m := reRootCauseLine.FindStringSubmatch(answer); len(m) > 1 {
		rootCause = strings.TrimSpace(m[1])
	}
	if m := reEvidenceLine.FindStringSubmatch(answer); len(m) > 1 {
		evidence = strings.TrimSpace(m[1])
	}
	if rootCause != "" && evidence != "" {
		return rootCause, evidence
	}
	parts := splitAIDiagnosisParagraphs(answer)
	if rootCause == "" && len(parts) > 0 {
		rootCause = parts[0]
	}
	if evidence == "" && len(parts) > 1 {
		evidence = strings.Join(parts[1:], "\n")
	}
	if rootCause == "" {
		rootCause = aiDiagnosisFirstLine(answer)
	}
	return rootCause, evidence
}

func splitAIDiagnosisParagraphs(s string) []string {
	var out []string
	for _, block := range strings.Split(s, "\n\n") {
		block = strings.TrimSpace(block)
		if block != "" {
			out = append(out, block)
		}
	}
	if len(out) > 0 {
		return out
	}
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 0 {
		return nil
	}
	return []string{strings.TrimSpace(lines[0]), strings.TrimSpace(strings.Join(lines[1:], "\n"))}
}

func aiDiagnosisFirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func probeEvidenceFallback(wr *goruntime.WatchReport) string {
	if wr == nil || len(wr.ProbeBundle) == 0 {
		if wr != nil && wr.Summary.Evidence != "" {
			return wr.Summary.Evidence
		}
		return ""
	}
	text := ""
	for _, key := range []string{"kubectl_focus_events", "kubectl_focus_logs_current", "kubectl_focus_describe"} {
		if v := strings.TrimSpace(wr.ProbeBundle[key]); v != "" {
			text = truncateBytes(v, 2000)
			break
		}
	}
	return text
}
